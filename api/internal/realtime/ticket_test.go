package realtime

import (
	"strings"
	"testing"
	"time"
)

func TestTickets(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	now := time.Date(2026, 10, 1, 5, 30, 0, 0, time.UTC)
	tk, exp := SignTicket(key, "job:job_1", now)
	if exp.Sub(now) != TicketTTL {
		t.Fatalf("expiry %v", exp)
	}
	if topic, err := VerifyTicket(key, tk, now.Add(30*time.Second)); err != nil || topic != "job:job_1" {
		t.Fatalf("valid ticket refused: %q %v", topic, err)
	}
	if _, err := VerifyTicket(key, tk, now.Add(TicketTTL+time.Second)); err == nil {
		t.Fatal("expired ticket accepted")
	}
	if _, err := VerifyTicket([]byte("another-key-another-key-another!!"), tk, now); err == nil {
		t.Fatal("ticket signed with another key accepted")
	}
	// Changing the topic breaks the signature.
	p, sig, _ := strings.Cut(tk, ".")
	forged, _ := SignTicket(key, "shop:shop_9", now)
	fp, _, _ := strings.Cut(forged, ".")
	if _, err := VerifyTicket(key, fp+"."+sig, now); err == nil || p == fp {
		t.Fatal("forged topic accepted")
	}
	for _, bad := range []string{"", "x", "a.b", tk + "x"} {
		if _, err := VerifyTicket(key, bad, now); err == nil {
			t.Fatalf("garbage %q accepted", bad)
		}
	}
	if _, err := VerifyTicket(nil, tk, now); err == nil {
		t.Fatal("empty key accepted")
	}
	other, _ := SignTicket(key, "admin:all", now)
	if _, err := VerifyTicket(key, other, now); err == nil {
		t.Fatal("unknown topic kind accepted")
	}
}
