package ratelimit

import (
	"context"
	"time"
)

// Shared counts requests per key in fixed one-minute windows kept where every API instance can see
// them (DynamoDB on AWS, where each Lambda instance would otherwise have its own in-memory buckets).
// It is used for the rules that guard against guessing and abuse: sign-in, setup links, staff names
// and new jobs.
type Shared interface {
	// Hit records one request for key in the window containing now and returns the window's count.
	Hit(ctx context.Context, key string, now time.Time) (int, error)
}

// Window is the length of a shared counting window.
const Window = time.Minute

// WindowStart is the start of the window containing t.
func WindowStart(t time.Time) time.Time { return t.UTC().Truncate(Window) }
