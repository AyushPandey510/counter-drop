package domain

import (
	"errors"
	"time"
)

type JobState string

const (
	JobStateNew       JobState = "new"
	JobStateClaimed   JobState = "claimed"
	JobStateReady     JobState = "ready"
	JobStateCollected JobState = "collected"
	JobStateCancelled JobState = "cancelled"
)

type UploadStatus string

const (
	UploadStatusPending  UploadStatus = "pending"
	UploadStatusUploaded UploadStatus = "uploaded"
)

type DeleteStatus string

const (
	DeleteStatusActive  DeleteStatus = "active"
	DeleteStatusPending DeleteStatus = "pending"
	DeleteStatusDeleted DeleteStatus = "deleted"
	DeleteStatusFailed  DeleteStatus = "failed"
)

var ErrInvalidTransition = errors.New("invalid job state transition")

type JobFile struct {
	ID           string       `json:"id"`
	Filename     string       `json:"filename"`
	Size         int64        `json:"size"`
	Mime         string       `json:"mime"`
	Pages        int          `json:"pages"`
	ObjectKey    string       `json:"objectKey,omitempty"`
	UploadStatus UploadStatus `json:"uploadStatus"`
	DeleteStatus DeleteStatus `json:"deleteStatus"`
	DeleteAfter  *time.Time   `json:"deleteAfter,omitempty"`
	DeletedAt    *time.Time   `json:"deletedAt,omitempty"`
	UploadURL    string       `json:"uploadUrl,omitempty"`
}

type Job struct {
	ID           string         `json:"id"`
	ShopID       string         `json:"shopId"`
	Token        string         `json:"token"`
	Secret       string         `json:"secret,omitempty"`
	CustomerName string         `json:"customerName"`
	Settings     map[string]any `json:"settings"`
	Files        []JobFile      `json:"files"`
	State        JobState       `json:"state"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	ClaimedAt    *time.Time     `json:"claimedAt,omitempty"`
	ReadyAt      *time.Time     `json:"readyAt,omitempty"`
	CollectedAt  *time.Time     `json:"collectedAt,omitempty"`
}

func (j *Job) Apply(action string, now time.Time) error {
	switch action {
	case "claim":
		if j.State != JobStateNew {
			return ErrInvalidTransition
		}
		j.State = JobStateClaimed
		j.ClaimedAt = &now
	case "ready":
		if j.State != JobStateClaimed {
			return ErrInvalidTransition
		}
		j.State = JobStateReady
		j.ReadyAt = &now
	case "collected":
		if j.State != JobStateReady {
			return ErrInvalidTransition
		}
		j.State = JobStateCollected
		j.CollectedAt = &now
	case "cancel":
		if j.State == JobStateCollected || j.State == JobStateCancelled {
			return ErrInvalidTransition
		}
		j.State = JobStateCancelled
	case "release":
		if j.State != JobStateClaimed {
			return ErrInvalidTransition
		}
		j.State = JobStateNew
		j.ClaimedAt = nil
	default:
		return ErrInvalidTransition
	}

	j.UpdatedAt = now
	return nil
}
