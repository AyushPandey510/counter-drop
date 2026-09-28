package domain

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestParsePageRange(t *testing.T) {
	cases := []struct {
		expr  string
		pages int
		want  []int
		err   bool
	}{
		{"", 3, []int{1, 2, 3}, false},
		{"1-3,5", 6, []int{1, 2, 3, 5}, false},
		{" 2 , 2-3 ", 4, []int{2, 3}, false},
		{"5", 5, []int{5}, false},
		{"0", 5, nil, true},
		{"3-1", 5, nil, true},
		{"1-6", 5, nil, true},
		{"1,,2", 5, nil, true},
		{"a", 5, nil, true},
		{"1-", 5, nil, true},
	}
	for _, c := range cases {
		got, err := ParsePageRange(c.expr, c.pages)
		if (err != nil) != c.err {
			t.Fatalf("%q: err = %v, want error %v", c.expr, err, c.err)
		}
		if !c.err && !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%q: got %v, want %v", c.expr, got, c.want)
		}
	}
}

// FSD §9.1 worked example: 12-page PDF, pages 1–10, both sides, 2 copies at ₹3/sheet = ₹30,
// plus a 1-page image in colour at ₹10 = ₹40 total.
func TestComputeQuoteWorkedExample(t *testing.T) {
	pl := PriceList{BWOne: 200, BWBoth: 300, ColourOne: 1000, ColourBoth: 1800}
	q, err := ComputeQuote(pl, []QuoteFile{
		{ID: "a", Kind: "pdf", Pages: 12, Settings: FileSettings{Copies: 2, BothSides: true, PageRange: "1-10"}},
		{ID: "b", Kind: "image", Pages: 1, Settings: FileSettings{Copies: 1, Colour: true, BothSides: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if q.Lines[0].AmountPaise != 3000 || q.Lines[0].Sheets != 5 {
		t.Fatalf("pdf line = %+v", q.Lines[0])
	}
	if q.Lines[1].AmountPaise != 1000 || q.Lines[1].BothSides {
		t.Fatalf("image line = %+v (images are always one side)", q.Lines[1])
	}
	if q.TotalPaise != 4000 || q.PagesTotal != 21 {
		t.Fatalf("total = %d pages = %d", q.TotalPaise, q.PagesTotal)
	}
}

func TestComputeQuoteRulesAndRounding(t *testing.T) {
	pl := PriceList{BWOne: 150, ColourOne: 0, MinCharge: 500}
	// Both sides without a sheet rate costs 2 × one side per sheet: 3 pages → 2 sheets × ₹3 = ₹6.
	q, _ := ComputeQuote(pl, []QuoteFile{{ID: "a", Kind: "pdf", Pages: 3, Settings: FileSettings{Copies: 1, BothSides: true}}})
	if q.SubtotalPaise != 600 || q.TotalPaise != 600 {
		t.Fatalf("both-sides fallback: %+v", q)
	}
	// Minimum charge applies: 1 page × ₹1.50 → ₹5.
	q, _ = ComputeQuote(pl, []QuoteFile{{ID: "a", Kind: "pdf", Pages: 1, Settings: FileSettings{Copies: 1}}})
	if q.TotalPaise != 500 || q.MinChargePaise != 500 {
		t.Fatalf("min charge: %+v", q)
	}
	// Rounding to the nearest rupee, half up: 3 pages × ₹1.50 = ₹4.50 → min charge ₹5; 7 × 150 = 1050 → 1100.
	q, _ = ComputeQuote(PriceList{BWOne: 150}, []QuoteFile{{ID: "a", Kind: "pdf", Pages: 7, Settings: FileSettings{Copies: 1}}})
	if q.TotalPaise != 1100 {
		t.Fatalf("rounding: %d", q.TotalPaise)
	}
	// Colour not offered.
	if _, err := ComputeQuote(pl, []QuoteFile{{ID: "a", Kind: "pdf", Pages: 1, Settings: FileSettings{Copies: 1, Colour: true}}}); !errors.Is(err, ErrOptionUnavailable) {
		t.Fatalf("colour unavailable: %v", err)
	}
	// Unknown pages → price confirmed at the counter.
	q, _ = ComputeQuote(pl, []QuoteFile{{ID: "a", Kind: "pdf", Pages: 0, Settings: FileSettings{Copies: 1}}})
	if !q.PagesToConfirm || q.TotalPaise != 0 {
		t.Fatalf("unknown pages: %+v", q)
	}
	// Copies out of range.
	if _, err := ComputeQuote(pl, []QuoteFile{{ID: "a", Kind: "pdf", Pages: 1, Settings: FileSettings{Copies: 100}}}); !errors.Is(err, ErrValidation) {
		t.Fatalf("copies: %v", err)
	}
}

func TestPriceVersionChangesWithSettings(t *testing.T) {
	pl := DefaultPriceList()
	f := []QuoteFile{{ID: "a", Kind: "pdf", Pages: 4, Settings: FileSettings{Copies: 1}}}
	a, _ := ComputeQuote(pl, f)
	b, _ := ComputeQuote(pl, f)
	f[0].Settings.Copies = 2
	c, _ := ComputeQuote(pl, f)
	if a.PriceVersion != b.PriceVersion || a.PriceVersion == c.PriceVersion {
		t.Fatalf("versions: %s %s %s", a.PriceVersion, b.PriceVersion, c.PriceVersion)
	}
}

var (
	staff  = Actor{Type: ActorStaff, Name: "Kavita"}
	guest  = Actor{Type: ActorGuest}
	system = Actor{Type: ActorSystem}
)

func TestTransitionHappyPathAndUndo(t *testing.T) {
	p := DefaultPolicy()
	now := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	j := Job{State: JobStateUploading}
	steps := []struct {
		a     Action
		actor Actor
		want  JobState
	}{
		{ActionSubmit, guest, JobStateQueued},
		{ActionClaim, staff, JobStateClaimed},
		{ActionRelease, staff, JobStateQueued},
		{ActionClaim, staff, JobStateClaimed},
		{ActionReady, staff, JobStateReady},
		{ActionCollected, staff, JobStateCollected},
	}
	for _, s := range steps {
		fx, err := Transition(&j, s.a, s.actor, "", now, p)
		if err != nil || j.State != s.want {
			t.Fatalf("%s: state %s err %v", s.a, j.State, err)
		}
		if s.a == ActionSubmit && !fx.IssueToken {
			t.Fatal("submit must issue a token")
		}
		if s.a == ActionCollected && (fx.ScheduleDelete == nil || !fx.ScheduleDelete.Equal(now.Add(10*time.Minute))) {
			t.Fatalf("collected must schedule deletion in 10 min, got %v", fx.ScheduleDelete)
		}
	}
	// Undo within the window.
	fx, err := Transition(&j, ActionUndo, staff, "", now.Add(9*time.Minute), p)
	if err != nil || j.State != JobStateReady || !fx.ClearDelete {
		t.Fatalf("undo: %v %s", err, j.State)
	}
	// Undo after the window fails.
	_, _ = Transition(&j, ActionCollected, staff, "", now, p)
	if _, err := Transition(&j, ActionUndo, staff, "", now.Add(11*time.Minute), p); !errors.Is(err, ErrWindowExpired) {
		t.Fatalf("late undo: %v", err)
	}
}

func TestTransitionMatrix(t *testing.T) {
	// For every state × action × actor, only these combinations are allowed.
	allowed := map[string]JobState{
		"uploading/submit/guest":   JobStateQueued,
		"uploading/edit/guest":     JobStateUploading,
		"queued/edit/guest":        JobStateQueued,
		"queued/claim/staff":       JobStateClaimed,
		"claimed/release/staff":    JobStateQueued,
		"claimed/ready/staff":      JobStateReady,
		"ready/collected/staff":    JobStateCollected,
		"collected/undo/staff":     JobStateReady,
		"uploading/cancel/guest":   JobStateCancelled,
		"queued/cancel/guest":      JobStateCancelled,
		"queued/cancel/staff":      JobStateCancelled,
		"claimed/cancel/staff":     JobStateCancelled,
		"ready/cancel/staff":       JobStateCancelled,
		"uploading/cancel/system":  JobStateCancelled,
		"queued/cancel/system":     JobStateCancelled,
		"claimed/cancel/system":    JobStateCancelled,
		"ready/cancel/system":      JobStateCancelled,
		"uploading/abandon/system": JobStateCancelled,
		// edit by staff/system is a no-op check that the job is still editable
		"uploading/edit/staff":  JobStateUploading,
		"queued/edit/staff":     JobStateQueued,
		"uploading/edit/system": JobStateUploading,
		"queued/edit/system":    JobStateQueued,
	}
	states := []JobState{JobStateUploading, JobStateQueued, JobStateClaimed, JobStateReady, JobStateCollected, JobStateCancelled}
	actions := []Action{ActionSubmit, ActionEdit, ActionClaim, ActionRelease, ActionReady, ActionCollected, ActionUndo, ActionCancel, ActionAbandon}
	actors := map[string]Actor{"guest": guest, "staff": staff, "system": system}
	now := time.Now()
	for _, st := range states {
		for _, a := range actions {
			for an, actor := range actors {
				j := Job{State: st, CollectedAt: &now}
				_, err := Transition(&j, a, actor, "reason", now, DefaultPolicy())
				key := string(st) + "/" + string(a) + "/" + an
				want, ok := allowed[key]
				if ok && (err != nil || j.State != want) {
					t.Errorf("%s: expected %s, got %s (%v)", key, want, j.State, err)
				}
				if !ok && err == nil {
					t.Errorf("%s: expected an error, got state %s", key, j.State)
				}
			}
		}
	}
}

func TestStaffCancelNeedsReason(t *testing.T) {
	j := Job{State: JobStateQueued}
	if _, err := Transition(&j, ActionCancel, staff, "", time.Now(), DefaultPolicy()); !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestTokensAndWait(t *testing.T) {
	if FormatToken("A", 7) != "A-07" || FormatToken("B", 123) != "B-123" {
		t.Fatal("token format")
	}
	late := time.Date(2026, 10, 1, 19, 0, 0, 0, time.UTC) // 00:30 IST next day
	if BusinessDay(late, IST) != "2026-10-02" {
		t.Fatalf("business day in IST: %s", BusinessDay(late, IST))
	}
	if w := EstimateWait(nil, 3, 1); w.JobsAhead != 0 || w.HighMinutes != 0 {
		t.Fatalf("empty wait: %+v", w)
	}
	w := EstimateWait([]int{20, 20, 0, 0}, 3, 1) // 4×3 + 2 = 14 min → 10–15
	if w.LowMinutes != 10 || w.HighMinutes != 15 || w.JobsAhead != 4 {
		t.Fatalf("wait: %+v", w)
	}
	if w := EstimateWait([]int{0, 0, 0, 0}, 3, 2); w.LowMinutes != 5 || w.HighMinutes != 10 {
		t.Fatalf("two counters: %+v", w)
	}
}

func TestShopHours(t *testing.T) {
	sh := Shop{OpensAt: "09:00", ClosesAt: "21:30", Timezone: "Asia/Kolkata"}
	at := func(h, m int) time.Time { return time.Date(2026, 10, 1, h, m, 0, 0, IST) }
	if sh.IsOpen(at(8, 59)) || !sh.IsOpen(at(9, 0)) || !sh.IsOpen(at(21, 29)) || sh.IsOpen(at(21, 30)) {
		t.Fatal("opening hours")
	}
	if got := sh.ClosingTime(at(12, 0)).In(IST); got.Hour() != 21 || got.Minute() != 30 {
		t.Fatalf("closing: %v", got)
	}
	sh.Lanes = []Lane{{Letter: "A", Rule: "bw"}, {Letter: "B", Rule: "colour"}}
	if l, _ := sh.LaneFor(true); l.Letter != "B" {
		t.Fatal("colour lane")
	}
	if l, _ := sh.LaneFor(false); l.Letter != "A" {
		t.Fatal("bw lane")
	}
}

func TestWeakPIN(t *testing.T) {
	for _, p := range []string{"0000", "1111", "1234", "0123", "6789", "9876", "3210", "1212", "4545", "2580", "123", "12345"} {
		if !WeakPIN(p) {
			t.Errorf("%s should be weak", p)
		}
	}
	for _, p := range []string{"4821", "7302", "1357", "9027", "5190"} {
		if WeakPIN(p) {
			t.Errorf("%s should be allowed", p)
		}
	}
}
