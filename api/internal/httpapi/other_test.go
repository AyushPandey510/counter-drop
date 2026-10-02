package httpapi_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"counter-drop/api/internal/domain"
)

// draftJob uploads PDFs with the given page counts but doesn't send the job.
func (e *env) draftJob(name string, pdfPages []int) (ticketT, map[string]string) {
	t := e.t
	t.Helper()
	var files []map[string]any
	for i := range pdfPages {
		files = append(files, map[string]any{"clientId": fmt.Sprint(i), "filename": fmt.Sprintf("form%d.pdf", i), "size": 100 + i, "mime": "application/pdf"})
	}
	created := must[struct {
		Ticket  ticketT  `json:"ticket"`
		Secret  string   `json:"secret"`
		Uploads []upload `json:"uploads"`
	}](t, e.do("POST", "/api/v1/cd/shops/demo-print/jobs", map[string]any{"customerName": name, "files": files}, nil), 201)
	h := map[string]string{"X-Ticket-Secret": created.Secret}
	for i, u := range created.Uploads {
		req, _ := http.NewRequest("PUT", u.URL, bytes.NewReader(bytes.Repeat([]byte("x"), 100+i)))
		for k, v := range u.Headers {
			req.Header.Set(k, v)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil || res.StatusCode != 200 {
			t.Fatalf("PUT upload: %v", err)
		}
		res.Body.Close()
		must[ticketT](t, e.do("POST", "/api/v1/cd/jobs/"+created.Ticket.Job.ID+"/files/"+u.FileID+"/complete", map[string]int{"pages": pdfPages[i]}, h), 200)
	}
	return must[ticketT](t, e.do("GET", "/api/v1/cd/jobs/"+created.Ticket.Job.ID, nil, h), 200), h
}

// A drop can mix printed files and "Other" files (attach to a form, laminate…). Other files aren't
// priced by the price list: the shop must enter a price before marking the job ready, and may
// choose not to give a receipt.
func TestOtherFilesPricedByShop(t *testing.T) {
	e := setup(t)
	tk, h := e.draftJob("Meena", []int{3, 2})
	jobPath := "/api/v1/cd/jobs/" + tk.Job.ID
	other := tk.Job.Files[1].ID

	// Too long a note is refused; a normal note is kept and print options are dropped.
	long := strings.Repeat("क", domain.MaxNoteRunes+1)
	if r := e.do("PATCH", jobPath, map[string]any{"files": []map[string]any{{"fileId": other, "settings": map[string]any{"other": true, "note": long}}}}, h); r.Status != 422 {
		t.Fatalf("long note: %d", r.Status)
	}
	tk = must[ticketT](t, e.do("PATCH", jobPath, map[string]any{"files": []map[string]any{{"fileId": other, "settings": map[string]any{
		"other": true, "note": "  Attach to admission form  ", "colour": true, "copies": 5}}}}, h), 200)
	f := tk.Job.Files[1].Settings
	if !f.Other || f.Note != "Attach to admission form" || f.Colour || f.Copies != 1 {
		t.Fatalf("other settings: %+v", f)
	}
	// "Apply to all" print options leave the Other file alone.
	tk = must[ticketT](t, e.do("PATCH", jobPath, map[string]any{"applyToAll": map[string]any{"copies": 1, "colour": false}}, h), 200)
	if !tk.Job.Files[1].Settings.Other || tk.Job.Files[1].Settings.Note != "Attach to admission form" {
		t.Fatalf("apply to all changed the Other file: %+v", tk.Job.Files[1].Settings)
	}
	// Only the 3-page file is priced (₹6 at ₹2 a side); the other is left to the shop.
	if tk.Quote.TotalPaise != 600 || tk.Quote.OtherFiles != 1 || !tk.Quote.Lines[1].Other || tk.Quote.PagesTotal != 3 {
		t.Fatalf("quote: %+v", tk.Quote)
	}
	sub := must[ticketT](t, e.do("POST", jobPath+"/submit", map[string]string{"priceVersion": tk.Quote.PriceVersion}, h), 200)
	if sub.Job.Token != "A-01" || sub.Job.OtherPrice != nil {
		t.Fatalf("submitted: %+v", sub.Job)
	}

	staff := e.login("Kavita", "1111")
	must[domain.Job](t, e.do("POST", "/api/v1/cd/shop/claim-next", map[string]string{"lane": "A"}, staff), 200)
	ready := "/api/v1/cd/shop/jobs/" + tk.Job.ID + "/ready"
	if r := e.do("POST", ready, nil, staff); r.Status != 422 || r.Error.Code != "price_required" {
		t.Fatalf("ready without price: %d %+v", r.Status, r.Error)
	}
	if r := e.do("POST", ready, map[string]any{"otherPricePaise": -100}, staff); r.Status != 422 {
		t.Fatalf("negative price: %d", r.Status)
	}
	j := must[domain.Job](t, e.do("POST", ready, map[string]any{"otherPricePaise": 1500}, staff), 200)
	if j.State != domain.JobStateReady || j.OtherPrice == nil || *j.OtherPrice != 1500 || j.PriceTotal != 2100 {
		t.Fatalf("ready: state %s other %v total %d", j.State, j.OtherPrice, j.PriceTotal)
	}
	// The customer sees the total, then the shop collects without a receipt.
	cust := must[ticketT](t, e.do("GET", jobPath, nil, h), 200)
	if cust.Job.PriceTotal != 2100 {
		t.Fatalf("customer total: %d", cust.Job.PriceTotal)
	}
	col := must[domain.Job](t, e.do("POST", "/api/v1/cd/shop/jobs/"+tk.Job.ID+"/collected", map[string]any{"paid": "upi", "noReceipt": true}, staff), 200)
	if !col.NoReceipt || col.PaidMethod != "upi" || col.PriceTotal != 2100 {
		t.Fatalf("collected: %+v", col)
	}

	// After deletion the note goes with the file name.
	e.clock.Advance(11 * time.Minute)
	e.deleter.RunOnce(context.Background())
	after := must[ticketT](t, e.do("GET", jobPath, nil, h), 200)
	if after.Job.FilesDeletedAt == nil || after.Job.Files[1].Settings.Note != "" || !after.Job.Files[1].Settings.Other {
		t.Fatalf("after deletion: %+v", after.Job.Files[1])
	}

	// A print-only job still goes ready without a price, and gets a receipt by default.
	tk2, _ := e.dropJob("Ravi", []int{1}, false)
	must[domain.Job](t, e.do("POST", "/api/v1/cd/shop/claim-next", map[string]string{"lane": "A"}, staff), 200)
	must[domain.Job](t, e.do("POST", "/api/v1/cd/shop/jobs/"+tk2.Job.ID+"/ready", nil, staff), 200)
	c2 := must[domain.Job](t, e.do("POST", "/api/v1/cd/shop/jobs/"+tk2.Job.ID+"/collected", nil, staff), 200)
	if c2.NoReceipt || c2.OtherPrice != nil || c2.PriceTotal != 200 {
		t.Fatalf("print-only: %+v", c2)
	}
}
