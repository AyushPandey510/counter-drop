package ddbstore

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"counter-drop/api/internal/domain"
	"counter-drop/api/internal/store"
)

// These tests need a DynamoDB endpoint (DynamoDB Local or moto): CD_TEST_DYNAMODB_ENDPOINT.
// The end-to-end behaviour tests live in internal/httpapi and use the same endpoint.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	endpoint := os.Getenv("CD_TEST_DYNAMODB_ENDPOINT")
	if endpoint == "" {
		t.Skip("CD_TEST_DYNAMODB_ENDPOINT not set")
	}
	ctx := context.Background()
	st, err := New(ctx, Config{Table: fmt.Sprintf("cd-unit-%d", time.Now().UnixNano()), Region: "ap-south-1", Endpoint: endpoint}, domain.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.EnsureTable(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.DeleteTable(context.Background()) })
	st.SetClock(func() time.Time { return time.Date(2026, 10, 1, 5, 30, 0, 0, time.UTC) })
	return st
}

func newDraft(t *testing.T, st *Store) (domain.Shop, domain.Job) {
	t.Helper()
	ctx := context.Background()
	sh, err := st.CreateShop(ctx, store.CreateShopInput{Slug: "unit-shop", Name: "Unit Shop", OwnerName: "Owner", OwnerPIN: "5821"})
	if err != nil {
		t.Fatal(err)
	}
	j, _, err := st.CreateJob(ctx, sh, store.CreateJobInput{CustomerName: "Asha", Files: []store.NewFile{{Filename: "a.pdf", Size: 1000, Mime: "application/pdf"}}},
		store.Limits{MaxFileBytes: 1 << 20, MaxJobBytes: 2 << 20, MaxFiles: 5})
	if err != nil {
		t.Fatal(err)
	}
	return sh, j
}

// A write that loses the race must re-read and re-apply its change, keeping the other writer's change.
func TestMutateJobRetriesOnVersionConflict(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, j := newDraft(t, st)

	calls := 0
	_, err := st.mutateJob(ctx, j.ID, func(r *jobRec, now time.Time) (*jobEvent, error) {
		calls++
		if calls == 1 {
			// Another request changes the job between our read and our write.
			if _, err := st.mutateJob(ctx, j.ID, func(r2 *jobRec, _ time.Time) (*jobEvent, error) {
				r2.Job.CustomerName = "Other"
				return nil, nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		r.Job.PaidMethod = "cash"
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("mutation ran %d times, want 2 (one conflict, one retry)", calls)
	}
	got, _ := st.GetJob(ctx, j.ID)
	if got.CustomerName != "Other" || got.PaidMethod != "cash" {
		t.Fatalf("lost update: name %q paid %q", got.CustomerName, got.PaidMethod)
	}
}

// The sparse index keys decide which lists a job is on; check them through a whole lifecycle.
func TestReindexFollowsLifecycle(t *testing.T) {
	now := time.Date(2026, 10, 1, 5, 30, 0, 0, time.UTC)
	del := now.Add(10 * time.Minute)
	r := &jobRec{Job: domain.Job{ID: "job_1", ShopID: "shop_1", State: domain.JobStateUploading, UpdatedAt: now, BusinessDay: "2026-10-01"},
		Files: []fileRec{{F: domain.JobFile{ID: "file_1"}}}}

	r.reindex()
	if r.GSI3PK != "DRAFT" || r.GSI1PK != "" || r.GSI2PK != "" || r.GSI4PK != "" {
		t.Fatalf("draft keys wrong: %+v", r)
	}

	r.Job.State, r.Job.QueuedAt = domain.JobStateQueued, &now
	r.reindex()
	if r.GSI1PK != "S#shop_1#OPEN" || r.GSI2PK != "S#shop_1#D#2026-10-01" || r.GSI3PK != "" {
		t.Fatalf("queued keys wrong: %+v", r)
	}

	r.Job.State = domain.JobStateCollected
	r.Files[0].F.DeleteAfter = &del
	r.Files[0].F.Downloads = 1
	r.reindex()
	if r.GSI1PK != "" || r.GSI3PK != "COPIES#shop_1" || r.GSI4PK != "DUE" || r.GSI4SK != ts(del)+"#job_1" {
		t.Fatalf("collected keys wrong: %+v", r)
	}

	// A failed delete waits for its retry time.
	retry := del.Add(5 * time.Minute)
	r.Files[0].NextAttemptAt = &retry
	r.reindex()
	if r.GSI4SK != ts(retry)+"#job_1" {
		t.Fatalf("retry not honoured: %s", r.GSI4SK)
	}

	r.Files[0].F.DeletedAt = &retry
	r.Job.CopiesDeletedAt = &retry
	r.reindex()
	if r.GSI3PK != "" || r.GSI4PK != "" {
		t.Fatalf("finished job still listed: %+v", r)
	}
}

func TestTimestampsSortAsStrings(t *testing.T) {
	a := time.Date(2026, 10, 1, 5, 30, 0, 999, time.UTC)
	b := a.Add(time.Nanosecond)
	c := time.Date(2026, 10, 1, 11, 0, 0, 0, time.FixedZone("IST", 5*3600+1800)) // same instant as 05:30 UTC
	if !(ts(a) < ts(b)) || ts(c) != ts(time.Date(2026, 10, 1, 5, 30, 0, 0, time.UTC)) {
		t.Fatalf("ts order broken: %s %s %s", ts(a), ts(b), ts(c))
	}
}

// Names are unique among current staff; a removed name can be reused.
func TestStaffNameGuard(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	sh, _ := newDraft(t, st)
	if err := st.AddStaff(ctx, sh.ID, "Sana", "staff", "5821"); err != nil {
		t.Fatal(err)
	}
	if err := st.AddStaff(ctx, sh.ID, "sana", "staff", "5821"); err == nil {
		t.Fatal("duplicate name accepted")
	}
	m, err := st.StaffByName(ctx, sh.ID, "Sana")
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := st.OwnerID(ctx, sh.ID)
	if err := st.RemoveStaff(ctx, sh.ID, m.ID, owner); err != nil {
		t.Fatal(err)
	}
	if err := st.AddStaff(ctx, sh.ID, "Sana", "staff", "5821"); err != nil {
		t.Fatalf("name not reusable after removal: %v", err)
	}
	if _, err := st.CreateShop(ctx, store.CreateShopInput{Slug: "unit-shop", Name: "Again", OwnerName: "Xavier"}); err != store.ErrSlugTaken {
		t.Fatalf("slug reuse: %v", err)
	}
}
