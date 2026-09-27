package domain

import (
	"testing"
	"time"
)

func TestJobApplyHappyPath(t *testing.T) {
	job := Job{State: JobStateNew}
	now := time.Now().UTC()

	for _, action := range []string{"claim", "ready", "collected"} {
		if err := job.Apply(action, now); err != nil {
			t.Fatalf("Apply(%q) failed: %v", action, err)
		}
	}

	if job.State != JobStateCollected {
		t.Fatalf("state = %q, want %q", job.State, JobStateCollected)
	}
}

func TestJobApplyRejectsInvalidTransition(t *testing.T) {
	job := Job{State: JobStateNew}

	if err := job.Apply("ready", time.Now().UTC()); err == nil {
		t.Fatal("Apply(\"ready\") succeeded from new state, want error")
	}
}
