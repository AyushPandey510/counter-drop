package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"counter-drop/api/internal/domain"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrIntakePaused = errors.New("shop intake is paused")
	ErrBadSecret    = errors.New("invalid job secret")
)

type MemoryStore struct {
	mu           sync.RWMutex
	shopsByID    map[string]domain.Shop
	shopIDBySlug map[string]string
	jobs         map[string]domain.Job
	tokenSeq     map[string]int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		shopsByID:    map[string]domain.Shop{},
		shopIDBySlug: map[string]string{},
		jobs:         map[string]domain.Job{},
		tokenSeq:     map[string]int{},
	}
}

func (s *MemoryStore) SeedDemoShop() {
	shop := domain.Shop{
		ID:           "shop_demo",
		Slug:         "demo-print",
		Name:         "Demo Print Counter",
		IntakePaused: false,
		Prices: map[string]int{
			"bw_page_paise":    200,
			"color_page_paise": 1000,
		},
		WaitMinutes: 5,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.shopsByID[shop.ID] = shop
	s.shopIDBySlug[shop.Slug] = shop.ID
}

func (s *MemoryStore) GetShopBySlug(slug string) (domain.Shop, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	shopID, ok := s.shopIDBySlug[slug]
	if !ok {
		return domain.Shop{}, ErrNotFound
	}

	return s.shopsByID[shopID], nil
}

func (s *MemoryStore) CreateJob(shopSlug string, input CreateJobInput) (domain.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	shopID, ok := s.shopIDBySlug[shopSlug]
	if !ok {
		return domain.Job{}, ErrNotFound
	}

	shop := s.shopsByID[shopID]
	if shop.IntakePaused {
		return domain.Job{}, ErrIntakePaused
	}

	now := time.Now().UTC()
	s.tokenSeq[shopID]++

	files := make([]domain.JobFile, 0, len(input.Files))
	for _, file := range input.Files {
		fileID := newID("file")
		files = append(files, domain.JobFile{
			ID:           fileID,
			Filename:     file.Filename,
			Size:         file.Size,
			Mime:         file.Mime,
			ObjectKey:    fmt.Sprintf("cd/%s/%s/%s", shopID, "pending", fileID),
			UploadStatus: domain.UploadStatusPending,
			DeleteStatus: domain.DeleteStatusActive,
		})
	}

	jobID := newID("job")
	for i := range files {
		files[i].ObjectKey = fmt.Sprintf("cd/%s/%s/%s", shopID, jobID, files[i].ID)
	}

	job := domain.Job{
		ID:           jobID,
		ShopID:       shopID,
		Token:        fmt.Sprintf("A-%02d", s.tokenSeq[shopID]),
		Secret:       randomSecret(),
		CustomerName: input.CustomerName,
		Settings:     input.Settings,
		Files:        files,
		State:        domain.JobStateNew,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.jobs[job.ID] = job
	return job, nil
}

func (s *MemoryStore) GetJob(jobID string, secret string) (domain.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return domain.Job{}, ErrNotFound
	}
	if secret == "" || secret != job.Secret {
		return domain.Job{}, ErrBadSecret
	}

	return job, nil
}

func (s *MemoryStore) ListShopQueue(shopSlug string) (QueueSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	shopID, ok := s.shopIDBySlug[shopSlug]
	if !ok {
		return QueueSnapshot{}, ErrNotFound
	}

	jobs := make([]domain.Job, 0)
	for _, job := range s.jobs {
		if job.ShopID != shopID {
			continue
		}
		job.Secret = ""
		jobs = append(jobs, job)
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
	})

	return QueueSnapshot{Shop: s.shopsByID[shopID], Jobs: jobs}, nil
}

func (s *MemoryStore) SubmitJob(jobID string, secret string) (domain.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return domain.Job{}, ErrNotFound
	}
	if secret == "" || secret != job.Secret {
		return domain.Job{}, ErrBadSecret
	}

	for i := range job.Files {
		job.Files[i].UploadStatus = domain.UploadStatusUploaded
	}
	job.UpdatedAt = time.Now().UTC()
	s.jobs[job.ID] = job
	return job, nil
}

func (s *MemoryStore) ApplyJobAction(jobID string, action string) (domain.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return domain.Job{}, ErrNotFound
	}

	now := time.Now().UTC()
	if err := job.Apply(action, now); err != nil {
		return domain.Job{}, err
	}
	scheduleFileDeletion(&job, action, now)

	s.jobs[jobID] = job
	return job, nil
}

func newID(prefix string) string {
	return prefix + "_" + randomHex(8)
}

func randomSecret() string {
	return randomHex(16)
}

func randomHex(bytes int) string {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

func scheduleFileDeletion(job *domain.Job, action string, now time.Time) {
	var deleteAfter *time.Time
	switch action {
	case "collected":
		value := now.Add(15 * time.Minute)
		deleteAfter = &value
	case "cancel":
		value := now
		deleteAfter = &value
	default:
		return
	}

	for i := range job.Files {
		if job.Files[i].DeletedAt != nil {
			continue
		}
		job.Files[i].DeleteStatus = domain.DeleteStatusPending
		job.Files[i].DeleteAfter = deleteAfter
	}
}
