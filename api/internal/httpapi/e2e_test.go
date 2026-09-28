package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"counter-drop/api/internal/config"
	"counter-drop/api/internal/domain"
	"counter-drop/api/internal/httpapi"
	"counter-drop/api/internal/realtime"
	"counter-drop/api/internal/storage"
	"counter-drop/api/internal/store"
	"counter-drop/api/internal/tasks"

	"github.com/jackc/pgx/v5"
)

type env struct {
	t       *testing.T
	url     string
	store   *store.Store
	deleter *tasks.Deleter
	clock   *fakeClock
	dir     string
}

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time          { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *fakeClock) Advance(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations")
}

// Most tests fire many requests from one address; the rate-limit test turns limits on itself.
var rateLimitInTests = false

func setup(t *testing.T) *env {
	t.Helper()
	dsn := os.Getenv("CD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("CD_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatal(err)
	}
	conn.Close(ctx)

	st, err := store.New(ctx, dsn, domain.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	// 11:00 IST on a weekday: the demo shop is open.
	clock := &fakeClock{t: time.Date(2026, 10, 1, 5, 30, 0, 0, time.UTC)}
	st.SetClock(clock.Now)
	if err := st.ApplyMigrations(ctx, migrationsDir()); err != nil {
		t.Fatal(err)
	}
	if err := st.SeedDemo(ctx); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewUnstartedServer(nil)
	base := "http://" + ts.Listener.Addr().String()
	dir := t.TempDir()
	local, err := storage.NewLocalStore(dir, base, []byte("test-signing-key-123456"), storage.Durations{})
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Load()
	cfg.UndoWindow = 10 * time.Minute
	cfg.RateLimit = rateLimitInTests
	hub := realtime.NewHub()
	srv := &httpapi.Server{Store: st, Objects: local, Local: local, Hub: hub, Logger: logger, Cfg: cfg}
	ts.Config.Handler = srv.Handler()
	ts.Start()
	t.Cleanup(ts.Close)
	return &env{t: t, url: base, store: st, clock: clock, dir: dir,
		deleter: &tasks.Deleter{Store: st, Objects: local, Hub: hub, Logger: logger}}
}

type resp struct {
	Status int
	Data   json.RawMessage `json:"data"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e *env) do(method, path string, body any, headers map[string]string) resp {
	e.t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, e.url+path, rdr)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	var r resp
	_ = json.NewDecoder(res.Body).Decode(&r)
	r.Status = res.StatusCode
	return r
}

func must[T any](t *testing.T, r resp, status int) T {
	t.Helper()
	if r.Status != status {
		msg := ""
		if r.Error != nil {
			msg = r.Error.Code + ": " + r.Error.Message
		}
		t.Fatalf("status %d, want %d (%s)", r.Status, status, msg)
	}
	var v T
	if err := json.Unmarshal(r.Data, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

type ticketT struct {
	Job       domain.Job   `json:"job"`
	Quote     domain.Quote `json:"quote"`
	Position  int          `json:"position"`
	HoldUntil *time.Time   `json:"holdUntil"`
}

type upload struct {
	FileID  string            `json:"fileId"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// dropJob runs the customer flow: create → PUT files → complete → submit.
func (e *env) dropJob(name string, pdfPages []int, colour bool) (ticketT, string) {
	t := e.t
	t.Helper()
	var files []map[string]any
	for i := range pdfPages {
		files = append(files, map[string]any{"clientId": fmt.Sprint(i), "filename": fmt.Sprintf("doc%d.pdf", i), "size": 100 + i, "mime": "application/pdf"})
	}
	created := must[struct {
		Ticket  ticketT  `json:"ticket"`
		Secret  string   `json:"secret"`
		Uploads []upload `json:"uploads"`
	}](t, e.do("POST", "/api/v1/cd/shops/demo-print/jobs", map[string]any{"customerName": name, "files": files}, nil), 201)
	h := map[string]string{"X-Ticket-Secret": created.Secret}
	if len(created.Uploads) != len(pdfPages) {
		t.Fatalf("uploads = %d", len(created.Uploads))
	}
	for i, u := range created.Uploads {
		req, _ := http.NewRequest("PUT", u.URL, bytes.NewReader(bytes.Repeat([]byte("x"), 100+i)))
		for k, v := range u.Headers {
			req.Header.Set(k, v)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil || res.StatusCode != 200 {
			t.Fatalf("PUT upload: %v %v", err, res.Status)
		}
		res.Body.Close()
		must[ticketT](t, e.do("POST", "/api/v1/cd/jobs/"+created.Ticket.Job.ID+"/files/"+u.FileID+"/complete", map[string]int{"pages": pdfPages[i]}, h), 200)
	}
	if colour {
		must[ticketT](t, e.do("PATCH", "/api/v1/cd/jobs/"+created.Ticket.Job.ID, map[string]any{"applyToAll": map[string]any{"copies": 1, "colour": true}}, h), 200)
	}
	tk := must[ticketT](t, e.do("GET", "/api/v1/cd/jobs/"+created.Ticket.Job.ID, nil, h), 200)
	sub := must[ticketT](t, e.do("POST", "/api/v1/cd/jobs/"+tk.Job.ID+"/submit", map[string]string{"priceVersion": tk.Quote.PriceVersion}, h), 200)
	return sub, created.Secret
}

func (e *env) login(name, pin string) map[string]string {
	e.t.Helper()
	out := must[struct {
		Token string `json:"token"`
	}](e.t, e.do("POST", "/api/v1/cd/shop/login", map[string]string{"shop": "demo-print", "name": name, "pin": pin}, nil), 200)
	return map[string]string{"Authorization": "Bearer " + out.Token}
}

func TestWalkInFlowEndToEnd(t *testing.T) {
	e := setup(t)

	shop := must[struct {
		Name        string `json:"name"`
		OnlineState string `json:"onlineState"`
		IsOpen      bool   `json:"isOpen"`
	}](t, e.do("GET", "/api/v1/cd/shops/demo-print", nil, nil), 200)
	if shop.OnlineState != "online" {
		t.Fatalf("shop: %+v", shop)
	}

	// Customer drops two PDFs (3 + 2 pages, B/W one side at ₹2) → ₹10, token A-01.
	tk, secret := e.dropJob("Priya", []int{3, 2}, false)
	if tk.Job.Token != "A-01" || tk.Job.State != domain.JobStateQueued || tk.Job.PriceTotal != 1000 {
		t.Fatalf("submitted: token %s state %s price %d", tk.Job.Token, tk.Job.State, tk.Job.PriceTotal)
	}
	// A colour job goes to lane B.
	tk2, _ := e.dropJob("Rahul", []int{1}, true)
	if tk2.Job.Token != "B-01" || tk2.Job.PriceTotal != 1000 {
		t.Fatalf("colour job: %s %d", tk2.Job.Token, tk2.Job.PriceTotal)
	}

	// Upload-only rule: walk-in jobs can't be paid online.
	if r := e.do("POST", "/api/v1/cd/jobs/"+tk.Job.ID+"/pay", nil, map[string]string{"X-Ticket-Secret": secret}); r.Status != 409 || r.Error.Code != "pay_not_allowed" {
		t.Fatalf("pay: %d", r.Status)
	}
	// Wrong secret is refused.
	if r := e.do("GET", "/api/v1/cd/jobs/"+tk.Job.ID, nil, map[string]string{"X-Ticket-Secret": "nope"}); r.Status != 403 {
		t.Fatalf("bad secret: %d", r.Status)
	}
	// Shop routes need a session.
	if r := e.do("GET", "/api/v1/cd/shop/queue", nil, nil); r.Status != 401 {
		t.Fatalf("queue without auth: %d", r.Status)
	}
	if r := e.do("POST", "/api/v1/cd/shop/login", map[string]string{"shop": "demo-print", "name": "Kavita", "pin": "9999"}, nil); r.Status != 401 {
		t.Fatalf("bad pin: %d", r.Status)
	}

	staff := e.login("Kavita", "1111")
	q := must[store.QueueSnapshot](t, e.do("GET", "/api/v1/cd/shop/queue", nil, staff), 200)
	if len(q.Jobs) != 2 || q.TodayCount != 2 {
		t.Fatalf("queue: %d jobs, today %d", len(q.Jobs), q.TodayCount)
	}

	// Claim oldest in lane A, open a file, mark ready.
	claimed := must[domain.Job](t, e.do("POST", "/api/v1/cd/shop/claim-next", map[string]string{"lane": "A"}, staff), 200)
	if claimed.ID != tk.Job.ID || claimed.State != domain.JobStateClaimed || claimed.ClaimedBy != "Kavita" {
		t.Fatalf("claim: %+v", claimed)
	}
	fu := must[struct {
		URL string `json:"url"`
	}](t, e.do("GET", "/api/v1/cd/shop/jobs/"+claimed.ID+"/files/"+claimed.Files[0].ID+"/url", nil, staff), 200)
	res, err := http.Get(fu.URL)
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("open file: %v %v", err, res.Status)
	}
	res.Body.Close()

	// Customer can no longer cancel once printing.
	if r := e.do("POST", "/api/v1/cd/jobs/"+tk.Job.ID+"/cancel", nil, map[string]string{"X-Ticket-Secret": secret}); r.Status != 409 {
		t.Fatalf("cancel after claim: %d", r.Status)
	}
	must[domain.Job](t, e.do("POST", "/api/v1/cd/shop/jobs/"+tk.Job.ID+"/ready", nil, staff), 200)

	// Lookup by token and collect with a paid flag.
	found := must[[]domain.Job](t, e.do("GET", "/api/v1/cd/shop/lookup?q=a1", nil, staff), 200)
	if len(found) != 1 || found[0].ID != tk.Job.ID {
		t.Fatalf("lookup: %+v", found)
	}
	col := must[domain.Job](t, e.do("POST", "/api/v1/cd/shop/jobs/"+tk.Job.ID+"/collected", map[string]string{"paid": "cash"}, staff), 200)
	if col.State != domain.JobStateCollected || col.PaidMethod != "cash" {
		t.Fatalf("collected: %+v", col)
	}

	// Undo within 10 minutes, collect again, then the worker deletes the files after the window.
	e.clock.Advance(5 * time.Minute)
	must[domain.Job](t, e.do("POST", "/api/v1/cd/shop/jobs/"+tk.Job.ID+"/undo", nil, staff), 200)
	must[domain.Job](t, e.do("POST", "/api/v1/cd/shop/jobs/"+tk.Job.ID+"/collected", nil, staff), 200)
	e.deleter.RunOnce(context.Background())
	if n := countFiles(t, e.dir); n != 3 {
		t.Fatalf("files deleted too early: %d left", n)
	}
	e.clock.Advance(11 * time.Minute)
	e.deleter.RunOnce(context.Background())
	after := must[ticketT](t, e.do("GET", "/api/v1/cd/jobs/"+tk.Job.ID, nil, map[string]string{"X-Ticket-Secret": secret}), 200)
	if after.Job.FilesDeletedAt == nil || after.Job.CustomerName != "" {
		t.Fatalf("receipt: deleted=%v name=%q", after.Job.FilesDeletedAt, after.Job.CustomerName)
	}
	for _, f := range after.Job.Files {
		if f.DeletedAt == nil || f.Filename != "" {
			t.Fatalf("file not cleared: %+v", f)
		}
	}
	if n := countFiles(t, e.dir); n != 1 {
		t.Fatalf("expected only the colour job's file on disk, found %d", n)
	}
	if r := e.do("POST", "/api/v1/cd/shop/jobs/"+tk.Job.ID+"/undo", nil, staff); r.Status != 409 {
		t.Fatalf("undo after deletion: %d", r.Status)
	}

	// Staff cancel needs a reason; the colour job is cancelled and its file removed.
	if r := e.do("POST", "/api/v1/cd/shop/jobs/"+tk2.Job.ID+"/cancel", map[string]string{}, staff); r.Status != 422 {
		t.Fatalf("cancel without reason: %d", r.Status)
	}
	must[domain.Job](t, e.do("POST", "/api/v1/cd/shop/jobs/"+tk2.Job.ID+"/cancel", map[string]string{"reason": "file_problem"}, staff), 200)
	e.deleter.RunOnce(context.Background())
	if n := countFiles(t, e.dir); n != 0 {
		t.Fatalf("cancelled job's file still on disk: %d", n)
	}
}

func countFiles(t *testing.T, dir string) int {
	n := 0
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && !strings.HasSuffix(p, ".type") {
			n++
		}
		return nil
	})
	return n
}

func TestUploadChecksAndPausedShop(t *testing.T) {
	e := setup(t)
	// A job can't be submitted before its files are uploaded.
	created := must[struct {
		Ticket ticketT `json:"ticket"`
		Secret string  `json:"secret"`
	}](t, e.do("POST", "/api/v1/cd/shops/demo-print/jobs", map[string]any{"files": []map[string]any{{"filename": "a.pdf", "size": 10, "mime": "application/pdf"}}}, nil), 201)
	h := map[string]string{"X-Ticket-Secret": created.Secret}
	if r := e.do("POST", "/api/v1/cd/jobs/"+created.Ticket.Job.ID+"/submit", map[string]string{}, h); r.Status != 409 || r.Error.Code != "uploads_incomplete" {
		t.Fatalf("submit before upload: %d", r.Status)
	}
	// Completing a file that was never uploaded fails.
	if r := e.do("POST", "/api/v1/cd/jobs/"+created.Ticket.Job.ID+"/files/"+created.Ticket.Job.Files[0].ID+"/complete", map[string]int{"pages": 1}, h); r.Status != 409 {
		t.Fatalf("complete missing upload: %d", r.Status)
	}
	// Disallowed file type.
	if r := e.do("POST", "/api/v1/cd/shops/demo-print/jobs", map[string]any{"files": []map[string]any{{"filename": "a.exe", "size": 10, "mime": "application/x-msdownload"}}}, nil); r.Status != 422 {
		t.Fatalf("bad type: %d", r.Status)
	}
	// Paused shop refuses new jobs.
	owner := e.login("Owner", "1234")
	must[domain.Shop](t, e.do("PUT", "/api/v1/cd/shop/state", map[string]string{"state": "paused", "message": "Back in 10 min"}, owner), 200)
	if r := e.do("POST", "/api/v1/cd/shops/demo-print/jobs", map[string]any{"files": []map[string]any{{"filename": "a.pdf", "size": 10, "mime": "application/pdf"}}}, nil); r.Status != 409 || r.Error.Code != "shop_paused" {
		t.Fatalf("paused: %d", r.Status)
	}
	// Staff (not owner) cannot change prices.
	staff := e.login("Kavita", "1111")
	if r := e.do("PUT", "/api/v1/cd/shop/settings", map[string]any{}, staff); r.Status != 403 {
		t.Fatalf("staff settings: %d", r.Status)
	}
}

func TestConcurrentClaimsNeverDuplicate(t *testing.T) {
	e := setup(t)
	const jobs = 40
	for i := 0; i < jobs; i++ {
		e.dropJob(fmt.Sprintf("C%d", i), []int{1}, false)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	seen := map[string]int{}
	var claims atomic.Int64
	for w := 0; w < 3; w++ {
		h := e.login("Kavita", "1111")
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				r := e.do("POST", "/api/v1/cd/shop/claim-next", map[string]string{"lane": "A"}, h)
				if r.Status == 404 {
					return
				}
				if r.Status != 200 {
					t.Errorf("claim status %d", r.Status)
					return
				}
				var j domain.Job
				_ = json.Unmarshal(r.Data, &j)
				mu.Lock()
				seen[j.ID]++
				mu.Unlock()
				claims.Add(1)
			}
		}()
	}
	wg.Wait()
	if claims.Load() != jobs || len(seen) != jobs {
		t.Fatalf("claims %d distinct %d, want %d", claims.Load(), len(seen), jobs)
	}
	for id, n := range seen {
		if n != 1 {
			t.Fatalf("job %s claimed %d times", id, n)
		}
	}
}

func TestTokensResetDaily(t *testing.T) {
	e := setup(t)
	tk, _ := e.dropJob("A", []int{1}, false)
	tk2, _ := e.dropJob("B", []int{1}, false)
	if tk.Job.Token != "A-01" || tk2.Job.Token != "A-02" {
		t.Fatalf("tokens %s %s", tk.Job.Token, tk2.Job.Token)
	}
	e.clock.Advance(24 * time.Hour)
	tk3, _ := e.dropJob("C", []int{1}, false)
	if tk3.Job.Token != "A-01" {
		t.Fatalf("next day token %s", tk3.Job.Token)
	}
}
