package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"counter-drop/api/internal/domain"
)

type queueT struct {
	Jobs           []domain.Job `json:"jobs"`
	CopiesToDelete []domain.Job `json:"copiesToDelete"`
}

func (e *env) fileLink(staff map[string]string, jobID, fileID, mode string) string {
	e.t.Helper()
	return must[struct {
		URL string `json:"url"`
	}](e.t, e.do("GET", "/api/v1/cd/shop/jobs/"+jobID+"/files/"+fileID+"/url?mode="+mode, nil, staff), 200).URL
}

func (e *env) act(staff map[string]string, jobID, action string, body map[string]string) domain.Job {
	e.t.Helper()
	return must[domain.Job](e.t, e.do("POST", "/api/v1/cd/shop/jobs/"+jobID+"/"+action, body, staff), 200)
}

func (e *env) ticket(jobID, secret string) ticketT {
	e.t.Helper()
	return must[ticketT](e.t, e.do("GET", "/api/v1/cd/jobs/"+jobID, nil, map[string]string{"X-Ticket-Secret": secret}), 200)
}

func TestShopCopiesFlow(t *testing.T) {
	e := setup(t)
	ctx := t.Context()
	staff := e.login("Kavita", "1111")
	tk, secret := e.dropJob("Riya", []int{3}, false)
	job, file := tk.Job.ID, tk.Job.Files[0].ID
	e.act(staff, job, "claim", nil)

	// Print: opens inline, counted, not a download.
	res, err := http.Get(e.fileLink(staff, job, file, "print"))
	if err != nil || res.StatusCode != 200 || !strings.HasPrefix(res.Header.Get("Content-Disposition"), "inline") {
		t.Fatalf("print link: %v %v %q", err, res.Status, res.Header.Get("Content-Disposition"))
	}
	res.Body.Close()
	got := e.ticket(job, secret).Job.Files[0]
	if got.PrintOpens != 1 || got.PrintedAt == nil || got.Downloads != 0 {
		t.Fatalf("after print: %+v", got)
	}

	// Download: saved as attachment; the customer sees who downloaded it.
	res, err = http.Get(e.fileLink(staff, job, file, "download"))
	if err != nil || res.StatusCode != 200 || !strings.HasPrefix(res.Header.Get("Content-Disposition"), "attachment") || res.Header.Get("Cache-Control") != "private, no-store" {
		t.Fatalf("download link: %v %q %q", err, res.Header.Get("Content-Disposition"), res.Header.Get("Cache-Control"))
	}
	res.Body.Close()
	got = e.ticket(job, secret).Job.Files[0]
	if got.Downloads != 1 || got.DownloadedBy != "Kavita" || got.DownloadedAt == nil {
		t.Fatalf("after download: %+v", got)
	}

	// Not finished yet: the customer can't ask for deletion, the shop can't confirm it.
	guest := map[string]string{"X-Ticket-Secret": secret}
	if r := e.do("POST", "/api/v1/cd/jobs/"+job+"/delete-request", nil, guest); r.Status != 409 {
		t.Fatalf("early delete-request: %d", r.Status)
	}
	if r := e.do("POST", "/api/v1/cd/shop/jobs/"+job+"/copies-deleted", nil, staff); r.Status != 422 {
		t.Fatalf("early copies-deleted: %d", r.Status)
	}

	e.act(staff, job, "ready", nil)
	e.act(staff, job, "collected", map[string]string{"paid": "upi"})

	// Collected with a downloaded copy → on the shop's to-do list.
	q := must[queueT](t, e.do("GET", "/api/v1/cd/shop/queue", nil, staff), 200)
	if len(q.CopiesToDelete) != 1 || q.CopiesToDelete[0].ID != job {
		t.Fatalf("copiesToDelete: %d", len(q.CopiesToDelete))
	}

	// Customer asks: our copy goes now (not after the 10-minute window), the shop is asked.
	tr := must[ticketT](t, e.do("POST", "/api/v1/cd/jobs/"+job+"/delete-request", nil, guest), 200)
	if tr.Job.CopiesDeleteRequestedAt == nil || tr.Job.CopiesDeletedAt != nil {
		t.Fatalf("after request: %+v", tr.Job)
	}
	e.deleter.RunOnce(ctx)
	tr = e.ticket(job, secret)
	if tr.Job.FilesDeletedAt == nil {
		t.Fatal("Counter Drop copy not deleted on request")
	}
	// Asking twice is harmless.
	must[ticketT](t, e.do("POST", "/api/v1/cd/jobs/"+job+"/delete-request", nil, guest), 200)

	// Shop confirms; the customer sees who and when; the to-do list empties.
	j := must[domain.Job](t, e.do("POST", "/api/v1/cd/shop/jobs/"+job+"/copies-deleted", nil, staff), 200)
	if j.CopiesDeletedAt == nil || j.CopiesDeletedBy != "Kavita" {
		t.Fatalf("confirm: %+v", j)
	}
	must[domain.Job](t, e.do("POST", "/api/v1/cd/shop/jobs/"+job+"/copies-deleted", nil, staff), 200)
	if q := must[queueT](t, e.do("GET", "/api/v1/cd/shop/queue", nil, staff), 200); len(q.CopiesToDelete) != 0 {
		t.Fatalf("to-do not cleared: %d", len(q.CopiesToDelete))
	}
	if tr := e.ticket(job, secret); tr.Job.CopiesDeletedBy != "Kavita" || tr.Job.PaidMethod != "upi" || tr.Job.PriceTotal != 600 {
		t.Fatalf("receipt data: %+v", tr.Job)
	}

	// Printed only (never downloaded): the request needs nothing from the shop.
	tk2, secret2 := e.dropJob("Aman", []int{2}, false)
	e.act(staff, tk2.Job.ID, "claim", nil)
	e.fileLink(staff, tk2.Job.ID, tk2.Job.Files[0].ID, "print")
	e.act(staff, tk2.Job.ID, "ready", nil)
	e.act(staff, tk2.Job.ID, "collected", map[string]string{"paid": "cash"})
	tr2 := must[ticketT](t, e.do("POST", "/api/v1/cd/jobs/"+tk2.Job.ID+"/delete-request", nil, map[string]string{"X-Ticket-Secret": secret2}), 200)
	if tr2.Job.CopiesDeletedAt == nil || tr2.Job.CopiesDeletedBy != "no-downloads" {
		t.Fatalf("no-download request: %+v", tr2.Job)
	}
	if r := e.do("POST", "/api/v1/cd/shop/jobs/"+tk2.Job.ID+"/copies-deleted", nil, staff); r.Status != 422 {
		t.Fatalf("confirming a job with no downloads: %d", r.Status)
	}
}

func TestActiveJobsKeepFilesAndStaleJobsClose(t *testing.T) {
	e := setup(t)
	ctx := t.Context()
	staff := e.login("Kavita", "1111")

	tk, secret := e.dropJob("Late", []int{4}, false)
	job := tk.Job.ID
	e.act(staff, job, "claim", nil)

	// Past closing time and overnight: an unfinished job keeps its files.
	e.clock.Advance(20 * time.Hour)
	e.deleter.RunOnce(ctx)
	tr := e.ticket(job, secret)
	if tr.Job.State != domain.JobStateClaimed || tr.Job.FilesDeletedAt != nil || tr.Job.Files[0].DeletedAt != nil {
		t.Fatalf("active job lost files: state=%s deleted=%v", tr.Job.State, tr.Job.FilesDeletedAt)
	}
	if tr.HoldUntil == nil {
		t.Fatal("ticket has no collect-by time")
	}

	// Undo after collected: the job is active again and keeps its files past the undo window.
	staff = e.login("Kavita", "1111") // the clock jump expired the 12-hour session
	owner := e.login("Owner", "1234")
	e.act(staff, job, "ready", nil)
	e.act(staff, job, "collected", nil)
	e.act(staff, job, "undo", nil)
	e.clock.Advance(30 * time.Minute)
	e.deleter.RunOnce(ctx)
	if tr := e.ticket(job, secret); tr.Job.FilesDeletedAt != nil {
		t.Fatal("files deleted after undo")
	}

	// Owner shortens the hold to 1 day; bad values are refused.
	cur := must[domain.Shop](t, e.do("GET", "/api/v1/cd/shop/settings", nil, owner), 200)
	body := map[string]any{"profile": map[string]any{"name": cur.Name, "address": cur.Address, "opensAt": cur.OpensAt, "closesAt": cur.ClosesAt, "holdDays": 9}, "prices": cur.Prices}
	if r := e.do("PUT", "/api/v1/cd/shop/settings", body, owner); r.Status != 422 {
		t.Fatalf("holdDays 9: %d", r.Status)
	}
	body["profile"].(map[string]any)["holdDays"] = 1
	if sh := must[domain.Shop](t, e.do("PUT", "/api/v1/cd/shop/settings", body, owner), 200); sh.HoldDays != 1 {
		t.Fatalf("holdDays = %d", sh.HoldDays)
	}

	// More than a day after it was queued, the ready-but-uncollected job closes itself and its files go.
	e.clock.Advance(5 * time.Hour)
	if jobs, err := e.store.ExpireUncollected(ctx); err != nil || len(jobs) != 1 {
		t.Fatalf("expire: %d jobs, err %v", len(jobs), err)
	}
	e.deleter.RunOnce(ctx) // closes the job, schedules deletion
	e.deleter.RunOnce(ctx) // deletes the files
	tr = e.ticket(job, secret)
	if tr.Job.State != domain.JobStateCancelled || tr.Job.CancelReason != "not_collected" || tr.Job.FilesDeletedAt == nil {
		t.Fatalf("stale job: state=%s reason=%q deleted=%v", tr.Job.State, tr.Job.CancelReason, tr.Job.FilesDeletedAt)
	}
}
