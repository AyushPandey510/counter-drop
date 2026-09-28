package httpapi_test

import (
	"bytes"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

// A file the customer removes before sending must never reach the counter: not listed, not openable,
// not priced, not counted on the receipt — and it is erased from storage.
func TestRemovedFileNeverReachesCounter(t *testing.T) {
	e := setup(t)
	files := []map[string]any{
		{"clientId": "a", "filename": "keep.pdf", "size": 100, "mime": "application/pdf"},
		{"clientId": "b", "filename": "private.pdf", "size": 101, "mime": "application/pdf"},
	}
	created := must[struct {
		Ticket  ticketT  `json:"ticket"`
		Secret  string   `json:"secret"`
		Uploads []upload `json:"uploads"`
	}](t, e.do("POST", "/api/v1/cd/shops/demo-print/jobs", map[string]any{"files": files}, nil), 201)
	h := map[string]string{"X-Ticket-Secret": created.Secret}
	jobID := created.Ticket.Job.ID
	for i, u := range created.Uploads {
		req, _ := http.NewRequest("PUT", u.URL, bytes.NewReader(bytes.Repeat([]byte("x"), 100+i)))
		for k, v := range u.Headers {
			req.Header.Set(k, v)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil || res.StatusCode != 200 {
			t.Fatalf("PUT: %v", err)
		}
		res.Body.Close()
		must[ticketT](t, e.do("POST", "/api/v1/cd/jobs/"+jobID+"/files/"+u.FileID+"/complete", map[string]int{"pages": 3}, h), 200)
	}
	removedID := created.Uploads[1].FileID

	tk := must[ticketT](t, e.do("DELETE", "/api/v1/cd/jobs/"+jobID+"/files/"+removedID, nil, h), 200)
	if len(tk.Job.Files) != 1 || tk.Job.Files[0].Filename != "keep.pdf" {
		t.Fatalf("customer still sees removed file: %+v", tk.Job.Files)
	}
	if tk.Quote.TotalPaise != 600 {
		t.Fatalf("quote %d, want 600 (3 pages × ₹2)", tk.Quote.TotalPaise)
	}
	// Removing twice is a 404, not a silent success.
	if r := e.do("DELETE", "/api/v1/cd/jobs/"+jobID+"/files/"+removedID, nil, h); r.Status != 404 {
		t.Fatalf("second remove: %d", r.Status)
	}
	sub := must[ticketT](t, e.do("POST", "/api/v1/cd/jobs/"+jobID+"/submit", map[string]string{"priceVersion": tk.Quote.PriceVersion}, h), 200)
	if len(sub.Job.Files) != 1 || sub.Job.PriceTotal != 600 {
		t.Fatalf("submitted job: files=%d total=%d", len(sub.Job.Files), sub.Job.PriceTotal)
	}

	staff := e.login("Kavita", "1111")
	q := must[struct {
		Jobs []struct {
			ID    string `json:"id"`
			Files []struct {
				ID string `json:"id"`
			} `json:"files"`
		} `json:"jobs"`
	}](t, e.do("GET", "/api/v1/cd/shop/queue", nil, staff), 200)
	for _, j := range q.Jobs {
		if j.ID == jobID && len(j.Files) != 1 {
			t.Fatalf("board lists %d files, want 1", len(j.Files))
		}
	}
	must[map[string]any](t, e.do("POST", "/api/v1/cd/shop/jobs/"+jobID+"/claim", map[string]any{}, staff), 200)
	if r := e.do("GET", "/api/v1/cd/shop/jobs/"+jobID+"/files/"+removedID+"/url", nil, staff); r.Status != 404 {
		t.Fatalf("staff could get a link to the removed file: %d", r.Status)
	}

	// The worker erases it (and its content-type sidecar) from storage; the kept file stays.
	e.deleter.RunOnce(t.Context())
	var removedLeft, keptLeft int
	_ = filepath.WalkDir(e.dir, func(p string, d fs.DirEntry, _ error) error {
		if d != nil && !d.IsDir() {
			if strings.Contains(p, removedID) {
				removedLeft++
			}
			if strings.Contains(p, created.Uploads[0].FileID) {
				keptLeft++
			}
		}
		return nil
	})
	if removedLeft != 0 || keptLeft == 0 {
		t.Fatalf("on disk: removed file parts %d (want 0), kept file parts %d (want >0)", removedLeft, keptLeft)
	}
}
