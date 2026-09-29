package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

type liveT struct {
	Mode   string `json:"mode"`
	URL    string `json:"url"`
	Ticket string `json:"ticket"`
}

// readEvent waits for the next message and returns its type.
func readEvent(t *testing.T, c *websocket.Conn) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, b, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("no live update: %v", err)
	}
	var ev struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal(b, &ev)
	return ev.Type
}

func TestLiveUpdatesDefaultToSSE(t *testing.T) {
	e := setup(t)
	tk, secret := e.dropJob("Asha", []int{2}, false)
	lt := must[liveT](t, e.do("POST", "/api/v1/cd/jobs/"+tk.Job.ID+"/live", nil, map[string]string{"X-Ticket-Secret": secret}), 200)
	if lt.Mode != "sse" || lt.Ticket != "" {
		t.Fatalf("default mode = %+v, want sse without a ticket", lt)
	}
	if r := e.do("POST", "/api/v1/cd/jobs/"+tk.Job.ID+"/live", nil, map[string]string{"X-Ticket-Secret": "wrong"}); r.Status == 200 {
		t.Fatal("ticket issued without the job secret")
	}
}

// Local WebSocket mode: the same ticket flow and messages as API Gateway on AWS.
func TestLiveUpdatesOverLocalWebSocket(t *testing.T) {
	liveModeInTests = "local"
	defer func() { liveModeInTests = "" }()
	e := setup(t)
	ctx := context.Background()
	wsBase := "ws" + strings.TrimPrefix(e.url, "http") + "/api/v1/cd/ws?t="

	tk, secret := e.dropJob("Asha", []int{2}, false)
	guest := map[string]string{"X-Ticket-Secret": secret}
	staff := e.login("Kavita", "1111")

	jl := must[liveT](t, e.do("POST", "/api/v1/cd/jobs/"+tk.Job.ID+"/live", nil, guest), 200)
	sl := must[liveT](t, e.do("POST", "/api/v1/cd/shop/live", nil, staff), 200)
	if jl.Mode != "ws" || jl.URL != "" || jl.Ticket == "" || sl.Ticket == "" {
		t.Fatalf("local tickets: %+v %+v", jl, sl)
	}
	if r := e.do("POST", "/api/v1/cd/shop/live", nil, nil); r.Status != 401 {
		t.Fatalf("shop ticket without sign-in: %d", r.Status)
	}

	customer, _, err := websocket.Dial(ctx, wsBase+jl.Ticket, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer customer.CloseNow()
	board, _, err := websocket.Dial(ctx, wsBase+sl.Ticket, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer board.CloseNow()
	time.Sleep(100 * time.Millisecond) // let both subscriptions register

	e.act(staff, tk.Job.ID, "claim", nil)
	if got := readEvent(t, customer); got != "job.updated" {
		t.Fatalf("customer got %q", got)
	}
	if got := readEvent(t, board); got != "queue.changed" {
		t.Fatalf("board got %q", got)
	}

	// A ticket only opens its own topic, and a bad or missing ticket opens nothing.
	for _, bad := range []string{"", "forged.ticket", jl.Ticket + "x"} {
		_, res, err := websocket.Dial(ctx, wsBase+bad, nil)
		if err == nil || res == nil || res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("dial with %q: err=%v", bad, err)
		}
	}
}
