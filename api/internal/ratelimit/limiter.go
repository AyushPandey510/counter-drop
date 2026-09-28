// Package ratelimit is a small in-memory token-bucket limiter keyed by string (e.g. "login|203.0.113.7").
// It is enough for a single API instance; with several instances, move the buckets to Redis.
package ratelimit

import (
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
}

type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	now     func() time.Time
}

func New() *Limiter {
	return &Limiter{buckets: map[string]*bucket{}, now: time.Now}
}

// Allow takes one token from key's bucket, which refills at perMinute and holds up to burst.
// When empty it returns false and how long until the next token.
func (l *Limiter) Allow(key string, perMinute, burst int) (bool, time.Duration) {
	now := l.now()
	rate := float64(perMinute) / 60 // tokens per second
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(burst), last: now}
		l.buckets[key] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * rate
	if b.tokens > float64(burst) {
		b.tokens = float64(burst)
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / rate * float64(time.Second))
	return false, wait
}

// Sweep drops buckets idle for longer than idle (they would be full again anyway).
func (l *Limiter) Sweep(idle time.Duration) {
	cutoff := l.now().Add(-idle)
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, b := range l.buckets {
		if b.last.Before(cutoff) {
			delete(l.buckets, k)
		}
	}
}

// Run sweeps every interval until stop is closed.
func (l *Limiter) Run(interval time.Duration, stop <-chan struct{}) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			l.Sweep(10 * time.Minute)
		case <-stop:
			return
		}
	}
}
