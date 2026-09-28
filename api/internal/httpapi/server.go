// Package httpapi exposes the Counter Drop REST API under /api/v1/cd.
package httpapi

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"counter-drop/api/internal/config"
	"counter-drop/api/internal/realtime"
	"counter-drop/api/internal/storage"
	"counter-drop/api/internal/store"
)

type Server struct {
	Store   *store.Store
	Objects storage.ObjectStore
	Local   *storage.LocalStore // non-nil when files are kept on local disk
	Hub     *realtime.Hub
	Logger  *slog.Logger
	Cfg     config.Config
}

func (s *Server) limits() store.Limits {
	return store.Limits{MaxFileBytes: s.Cfg.MaxFileBytes, MaxJobBytes: s.Cfg.MaxJobBytes, MaxFiles: s.Cfg.MaxFiles}
}

// Handler builds the full HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	const p = "/api/v1/cd"

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeData(w, 200, map[string]any{"status": "ok", "time": time.Now().UTC()})
	})
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		if err := s.Store.Ping(r.Context()); err != nil {
			failCode(w, 503, "not_ready", "database unavailable")
			return
		}
		writeData(w, 200, map[string]any{"status": "ready"})
	})

	// Guest (customer) routes — upload only, pay at the counter.
	mux.HandleFunc("GET "+p+"/shops/{slug}", s.getShop)
	mux.HandleFunc("POST "+p+"/shops/{slug}/jobs", s.createJob)
	mux.HandleFunc("GET "+p+"/jobs/{id}", s.guest(s.getTicket))
	mux.HandleFunc("PATCH "+p+"/jobs/{id}", s.guest(s.updateJob))
	mux.HandleFunc("POST "+p+"/jobs/{id}/files", s.guest(s.addFiles))
	mux.HandleFunc("DELETE "+p+"/jobs/{id}/files/{fileId}", s.guest(s.removeFile))
	mux.HandleFunc("POST "+p+"/jobs/{id}/files/{fileId}/complete", s.guest(s.completeFile))
	mux.HandleFunc("POST "+p+"/jobs/{id}/submit", s.guest(s.submitJob))
	mux.HandleFunc("POST "+p+"/jobs/{id}/cancel", s.guest(s.cancelJob))
	mux.HandleFunc("POST "+p+"/jobs/{id}/pay", s.guest(s.payJob))
	mux.HandleFunc("GET "+p+"/jobs/{id}/events", s.jobEvents)

	// Shop routes.
	mux.HandleFunc("GET "+p+"/shop/staff-names", s.staffNames)
	mux.HandleFunc("POST "+p+"/shop/login", s.login)
	mux.HandleFunc("POST "+p+"/shop/logout", s.staff(s.logout))
	mux.HandleFunc("GET "+p+"/shop/me", s.staff(s.me))
	mux.HandleFunc("GET "+p+"/shop/queue", s.staff(s.queue))
	mux.HandleFunc("POST "+p+"/shop/claim-next", s.staff(s.claimNext))
	mux.HandleFunc("POST "+p+"/shop/jobs/{id}/{action}", s.staff(s.jobAction))
	mux.HandleFunc("GET "+p+"/shop/jobs/{id}/files/{fileId}/url", s.staff(s.fileURL))
	mux.HandleFunc("GET "+p+"/shop/lookup", s.staff(s.lookup))
	mux.HandleFunc("PUT "+p+"/shop/state", s.staff(s.setState))
	mux.HandleFunc("GET "+p+"/shop/settings", s.staff(s.getSettings))
	mux.HandleFunc("PUT "+p+"/shop/settings", s.owner(s.putSettings))
	mux.HandleFunc("GET "+p+"/shop/events", s.shopEvents)
	mux.HandleFunc("GET "+p+"/shop/deletion-health", s.owner(s.deletionHealth))

	if s.Local != nil {
		mux.Handle(p+"/files/", s.Local.Handler())
	}

	var h http.Handler = mux
	if s.Cfg.WebDir != "" {
		h = spaFallback(s.Cfg.WebDir, mux)
	}
	return chain(h, s.Logger, s.Cfg.WebOrigins)
}

// spaFallback serves the built PWA for non-API paths, with index.html for client-side routes.
func spaFallback(dir string, api http.Handler) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" || r.URL.Path == "/ready" {
			api.ServeHTTP(w, r)
			return
		}
		path := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			if strings.Contains(r.URL.Path, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			files.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}
