package ratelimit

import (
	"testing"
	"time"
)

func TestBucket(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := New()
	l.now = func() time.Time { return now }
	for i := 0; i < 5; i++ {
		if ok, _ := l.Allow("k", 60, 5); !ok {
			t.Fatalf("burst request %d refused", i)
		}
	}
	ok, wait := l.Allow("k", 60, 5)
	if ok || wait <= 0 || wait > time.Second {
		t.Fatalf("6th: ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("other", 60, 5); !ok {
		t.Fatal("keys must be independent")
	}
	now = now.Add(time.Second) // 60/min → one token back per second
	if ok, _ := l.Allow("k", 60, 5); !ok {
		t.Fatal("token not refilled")
	}
	now = now.Add(20 * time.Minute)
	l.Sweep(10 * time.Minute)
	if len(l.buckets) != 0 {
		t.Fatalf("sweep left %d buckets", len(l.buckets))
	}
}
