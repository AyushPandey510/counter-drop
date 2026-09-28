package httpapi

import (
	"net/http"
	"time"

	"counter-drop/api/internal/store"
)

// linkOut is how a new setup link leaves the API. It is shown once; only its hash is stored.
type linkOut struct {
	Purpose   string    `json:"purpose"`
	ExpiresAt time.Time `json:"expiresAt"`
	SetupPath string    `json:"setupPath"` // /shop/setup#<token>; the web app prefixes its own origin
	SetupURL  string    `json:"setupUrl"`  // full link using CD_PUBLIC_WEB_URL
}

func (s *Server) linkOut(l store.SetupLink) linkOut {
	return linkOut{Purpose: l.Purpose, ExpiresAt: l.ExpiresAt, SetupPath: "/shop/setup#" + l.Token, SetupURL: s.Cfg.SetupURL(l.Token)}
}

// setupInfo: POST so the token travels in the body, never in a logged URL.
func (s *Server) setupInfo(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := decode(r, &body); err != nil {
		s.fail(w, r, err)
		return
	}
	info, err := s.Store.SetupLinkInfo(r.Context(), body.Token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeData(w, 200, info)
}

// completeSetup sets the chosen PIN and signs the person in (same response as login).
func (s *Server) completeSetup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
		PIN   string `json:"pin"`
	}
	if err := decode(r, &body); err != nil {
		s.fail(w, r, err)
		return
	}
	token, p, err := s.Store.CompleteSetup(r.Context(), body.Token, body.PIN, s.Cfg.SessionTTL)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	sh, err := s.Store.GetShopByID(r.Context(), p.Staff.ShopID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.Logger.Info("staff setup completed", "shop", sh.Slug, "staff_id", p.Staff.ID)
	writeData(w, 200, map[string]any{"token": token, "staff": p.Staff, "expiresAt": p.ExpiresAt, "shop": sh})
}

func (s *Server) changePIN(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CurrentPIN string `json:"currentPin"`
		NewPIN     string `json:"newPin"`
	}
	if err := decode(r, &body); err != nil {
		s.fail(w, r, err)
		return
	}
	p := principal(r.Context())
	if err := s.Store.ChangePIN(r.Context(), p.Staff.ID, bearer(r), body.CurrentPIN, body.NewPIN); err != nil {
		s.fail(w, r, err)
		return
	}
	writeData(w, 200, map[string]bool{"ok": true})
}

func (s *Server) listStaff(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListStaff(r.Context(), principal(r.Context()).Staff.ShopID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeData(w, 200, map[string]any{"staff": list})
}

func (s *Server) inviteStaff(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	if err := decode(r, &body); err != nil {
		s.fail(w, r, err)
		return
	}
	if body.Role == "" {
		body.Role = "staff"
	}
	p := principal(r.Context())
	m, link, err := s.Store.InviteStaff(r.Context(), p.Staff.ShopID, body.Name, body.Role, p.Staff.ID, s.Cfg.SetupLinkTTL)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeData(w, 201, map[string]any{"staff": m, "link": s.linkOut(link)})
}

// staffLink issues a new link: "setup" if they never finished, otherwise "reset" (clears the old PIN).
func (s *Server) staffLink(w http.ResponseWriter, r *http.Request) {
	p := principal(r.Context())
	id := r.PathValue("id")
	if id == p.Staff.ID {
		s.fail(w, r, store.ErrSelf)
		return
	}
	list, err := s.Store.ListStaff(r.Context(), p.Staff.ShopID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	purpose := ""
	for _, m := range list {
		if m.ID == id {
			purpose = "reset"
			if m.Pending {
				purpose = "setup"
			}
		}
	}
	if purpose == "" {
		s.fail(w, r, store.ErrNotFound)
		return
	}
	link, err := s.Store.IssueSetupLink(r.Context(), p.Staff.ShopID, id, purpose, p.Staff.ID, s.Cfg.SetupLinkTTL)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeData(w, 200, map[string]any{"link": s.linkOut(link)})
}

func (s *Server) removeStaff(w http.ResponseWriter, r *http.Request) {
	p := principal(r.Context())
	if err := s.Store.RemoveStaff(r.Context(), p.Staff.ShopID, r.PathValue("id"), p.Staff.ID); err != nil {
		s.fail(w, r, err)
		return
	}
	writeData(w, 200, map[string]bool{"ok": true})
}
