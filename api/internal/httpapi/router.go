package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"counter-drop/api/internal/domain"
	"counter-drop/api/internal/store"
)

type Store interface {
	GetShopBySlug(slug string) (domain.Shop, error)
	CreateJob(shopSlug string, input store.CreateJobInput) (domain.Job, error)
	ListShopQueue(shopSlug string) (store.QueueSnapshot, error)
	GetJob(jobID string, secret string) (domain.Job, error)
	SubmitJob(jobID string, secret string) (domain.Job, error)
	ApplyJobAction(jobID string, action string) (domain.Job, error)
}

type UploadPresigner interface {
	PresignUpload(ctx context.Context, objectKey string, contentType string) (string, error)
}

type RouterConfig struct {
	Store   Store
	Uploads UploadPresigner
	Logger  *slog.Logger
}

type envelope struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, envelope{
			Success: true,
			Data: map[string]any{
				"status":    "healthy",
				"service":   "counter-drop-api",
				"timestamp": time.Now().UTC(),
			},
		})
	})

	mux.HandleFunc("GET /api/v1/cd/shops/{slug}", func(w http.ResponseWriter, r *http.Request) {
		shop, err := cfg.Store.GetShopBySlug(r.PathValue("slug"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Success: true, Data: shop})
	})

	mux.HandleFunc("POST /api/v1/cd/shops/{slug}/jobs", func(w http.ResponseWriter, r *http.Request) {
		var input store.CreateJobInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, envelope{Success: false, Error: "invalid JSON body"})
			return
		}
		if strings.TrimSpace(input.CustomerName) == "" {
			writeJSON(w, http.StatusBadRequest, envelope{Success: false, Error: "customerName is required"})
			return
		}
		if len(input.Files) == 0 {
			writeJSON(w, http.StatusBadRequest, envelope{Success: false, Error: "at least one file is required"})
			return
		}

		job, err := cfg.Store.CreateJob(r.PathValue("slug"), input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if err := attachUploadURLs(r.Context(), cfg.Uploads, &job); err != nil {
			writeJSON(w, http.StatusInternalServerError, envelope{Success: false, Error: "could not prepare upload URLs"})
			return
		}
		writeJSON(w, http.StatusCreated, envelope{Success: true, Data: job})
	})

	mux.HandleFunc("GET /api/v1/cd/shop/queue", func(w http.ResponseWriter, r *http.Request) {
		shopSlug := strings.TrimSpace(r.URL.Query().Get("shop"))
		if shopSlug == "" {
			writeJSON(w, http.StatusBadRequest, envelope{Success: false, Error: "shop query parameter is required"})
			return
		}

		queue, err := cfg.Store.ListShopQueue(shopSlug)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Success: true, Data: queue})
	})

	mux.HandleFunc("GET /api/v1/cd/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		job, err := cfg.Store.GetJob(r.PathValue("id"), r.URL.Query().Get("secret"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		job.Secret = ""
		writeJSON(w, http.StatusOK, envelope{Success: true, Data: job})
	})

	mux.HandleFunc("POST /api/v1/cd/jobs/{id}/submit", func(w http.ResponseWriter, r *http.Request) {
		job, err := cfg.Store.SubmitJob(r.PathValue("id"), r.URL.Query().Get("secret"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		job.Secret = ""
		writeJSON(w, http.StatusOK, envelope{Success: true, Data: job})
	})

	mux.HandleFunc("POST /api/v1/cd/shop/jobs/{id}/{action}", func(w http.ResponseWriter, r *http.Request) {
		job, err := cfg.Store.ApplyJobAction(r.PathValue("id"), r.PathValue("action"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		job.Secret = ""
		writeJSON(w, http.StatusOK, envelope{Success: true, Data: job})
	})

	return requestLogger(cfg.Logger, mux)
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func attachUploadURLs(ctx context.Context, uploads UploadPresigner, job *domain.Job) error {
	if uploads == nil {
		return nil
	}
	for i := range job.Files {
		url, err := uploads.PresignUpload(ctx, job.Files[i].ObjectKey, job.Files[i].Mime)
		if err != nil {
			return err
		}
		job.Files[i].UploadURL = url
	}
	return nil
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeJSON(w, http.StatusNotFound, envelope{Success: false, Error: "not found"})
	case errors.Is(err, store.ErrBadSecret):
		writeJSON(w, http.StatusForbidden, envelope{Success: false, Error: "invalid job secret"})
	case errors.Is(err, store.ErrIntakePaused):
		writeJSON(w, http.StatusConflict, envelope{Success: false, Error: "shop intake is paused"})
	case errors.Is(err, domain.ErrInvalidTransition):
		writeJSON(w, http.StatusConflict, envelope{Success: false, Error: "invalid job state transition"})
	default:
		writeJSON(w, http.StatusInternalServerError, envelope{Success: false, Error: "internal server error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
