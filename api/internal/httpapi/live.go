package httpapi

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"counter-drop/api/internal/realtime"

	"github.com/coder/websocket"
)

// Live updates. The client first asks for a ticket (authenticated like any other call), then:
//
//	mode "sse"  opens the Server-Sent Events stream (/jobs/{id}/events, /shop/events) as before
//	mode "ws"   opens a WebSocket to url (API Gateway on AWS, or this API's /ws in local mode) with ?t=<ticket>
//
// Either way the messages are the same pointer events, and the client refetches on each one.

type liveTicket struct {
	Mode      string     `json:"mode"`
	URL       string     `json:"url,omitempty"` // empty in local mode: same origin as the API, path /ws
	Ticket    string     `json:"ticket,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

func (s *Server) liveTicketFor(topic string) liveTicket {
	if s.Cfg.Realtime.WSURL == "" {
		return liveTicket{Mode: "sse"}
	}
	tk, exp := realtime.SignTicket(s.realtimeKey(), topic, time.Now())
	lt := liveTicket{Mode: "ws", Ticket: tk, ExpiresAt: &exp}
	if s.Cfg.Realtime.WSURL != "local" {
		lt.URL = s.Cfg.Realtime.WSURL
	}
	return lt
}

func (s *Server) realtimeKey() []byte { return []byte(s.Cfg.Realtime.Key) }

// POST /jobs/{id}/live (guest)
func (s *Server) jobLive(w http.ResponseWriter, r *http.Request) {
	writeData(w, 200, s.liveTicketFor(realtime.JobTopic(r.PathValue("id"))))
}

// POST /shop/live (staff)
func (s *Server) shopLive(w http.ResponseWriter, r *http.Request) {
	writeData(w, 200, s.liveTicketFor(realtime.ShopTopic(principal(r.Context()).Staff.ShopID)))
}

// GET /ws?t=<ticket> — local stand-in for the API Gateway WebSocket (CD_REALTIME_WS_URL=local), fed by
// the in-process hub. Same ticket check, same messages, so the browser code is exercised as on AWS.
func (s *Server) localWS(w http.ResponseWriter, r *http.Request) {
	topic, err := realtime.VerifyTicket(s.realtimeKey(), r.URL.Query().Get("t"), time.Now())
	if err != nil {
		failCode(w, 401, "unauthorized", "Live-update ticket missing or expired.")
		return
	}
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: s.wsOrigins()})
	if err != nil {
		return // Accept has already written the error response
	}
	defer c.CloseNow()
	ctx := c.CloseRead(r.Context()) // discards heartbeats; ctx ends when the client goes away
	events, cancel := s.Hub.Subscribe(topic)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			c.Close(websocket.StatusNormalClosure, "")
			return
		case b := <-events:
			wctx, done := context.WithTimeout(ctx, 10*time.Second)
			err := c.Write(wctx, websocket.MessageText, b)
			done()
			if err != nil {
				return
			}
		}
	}
}

// wsOrigins lists browser origins allowed to open the local WebSocket: the web app's origins.
func (s *Server) wsOrigins() []string {
	var out []string
	for _, raw := range append([]string{s.Cfg.PublicWebURL, s.Cfg.PublicAPIURL}, s.Cfg.WebOrigins...) {
		if u, err := url.Parse(strings.TrimSpace(raw)); err == nil && u.Host != "" {
			out = append(out, u.Host)
		}
	}
	return out
}

// wsOrigin is the origin the Content-Security-Policy must allow for live updates.
func (s *Server) wsOrigin() string {
	switch ws := s.Cfg.Realtime.WSURL; ws {
	case "":
		return ""
	case "local":
		o := originOf(s.Cfg.PublicAPIURL)
		o = strings.Replace(strings.Replace(o, "https://", "wss://", 1), "http://", "ws://", 1)
		return o
	default:
		return originOf(ws)
	}
}
