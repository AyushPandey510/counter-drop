package store

import "counter-drop/api/internal/domain"

type CreateJobInput struct {
	CustomerName string         `json:"customerName"`
	Settings     map[string]any `json:"settings"`
	Files        []JobFileInput `json:"files"`
}

type JobFileInput struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Mime     string `json:"mime"`
}

type QueueSnapshot struct {
	Shop domain.Shop  `json:"shop"`
	Jobs []domain.Job `json:"jobs"`
}
