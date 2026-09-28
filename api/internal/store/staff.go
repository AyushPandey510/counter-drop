package store

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"counter-drop/api/internal/domain"

	"github.com/jackc/pgx/v5"
)

// Staff accounts are managed with one-time setup links: whoever creates an account never learns its PIN.
// A link is a random 144-bit token shown once; only its SHA-256 is stored.

// StaffMember is a staff row as the owner sees it.
type StaffMember struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	Pending   bool       `json:"pending"` // no PIN yet: waiting for the setup link to be used
	PinSetAt  *time.Time `json:"pinSetAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// SetupLink is returned once, when created. Token goes in the URL fragment: /shop/setup#<token>.
type SetupLink struct {
	Token     string    `json:"-"`
	Purpose   string    `json:"purpose"` // setup | reset
	ExpiresAt time.Time `json:"expiresAt"`
}

// SetupInfo is what the setup page shows before the person chooses a PIN.
type SetupInfo struct {
	ShopName  string    `json:"shopName"`
	ShopSlug  string    `json:"shopSlug"`
	StaffName string    `json:"staffName"`
	Role      string    `json:"role"`
	Purpose   string    `json:"purpose"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func newLinkToken() string {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// ListStaff returns current staff, owners first.
func (s *Store) ListStaff(ctx context.Context, shopID string) ([]StaffMember, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, role, pin_hash = '', pin_set_at, created_at FROM cd_staff
		WHERE shop_id = $1 AND active ORDER BY (role = 'owner') DESC, lower(name)`, shopID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StaffMember{}
	for rows.Next() {
		var m StaffMember
		if err := rows.Scan(&m.ID, &m.Name, &m.Role, &m.Pending, &m.PinSetAt, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// StaffByName finds a current staff member of a shop (used by the admin CLI).
func (s *Store) StaffByName(ctx context.Context, shopID, name string) (StaffMember, error) {
	var m StaffMember
	err := s.pool.QueryRow(ctx, `SELECT id, name, role, pin_hash = '', pin_set_at, created_at FROM cd_staff
		WHERE shop_id = $1 AND active AND lower(name) = lower($2)`, shopID, strings.TrimSpace(name)).
		Scan(&m.ID, &m.Name, &m.Role, &m.Pending, &m.PinSetAt, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, ErrNotFound
	}
	return m, err
}

func (s *Store) staffInShop(ctx context.Context, q pgx.Tx, shopID, staffID string) (StaffMember, error) {
	var m StaffMember
	err := q.QueryRow(ctx, `SELECT id, name, role, pin_hash = '', pin_set_at, created_at FROM cd_staff
		WHERE id = $1 AND shop_id = $2 AND active FOR UPDATE`, staffID, shopID).
		Scan(&m.ID, &m.Name, &m.Role, &m.Pending, &m.PinSetAt, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, ErrNotFound
	}
	return m, err
}

// InviteStaff adds a pending staff member and returns their setup link.
func (s *Store) InviteStaff(ctx context.Context, shopID, name, role, createdBy string, ttl time.Duration) (StaffMember, SetupLink, error) {
	var m StaffMember
	var link SetupLink
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		id, err := addStaffTx(ctx, tx, shopID, name, role, "")
		if err != nil {
			return err
		}
		if m, err = s.staffInShop(ctx, tx, shopID, id); err != nil {
			return err
		}
		link, err = s.issueLinkTx(ctx, tx, shopID, id, "setup", createdBy, ttl)
		return err
	})
	return m, link, err
}

// IssueSetupLink creates a fresh link for an existing staff member and cancels any earlier unused link.
// purpose "reset" also clears their PIN and signs them out everywhere (lost or leaked PIN).
func (s *Store) IssueSetupLink(ctx context.Context, shopID, staffID, purpose, createdBy string, ttl time.Duration) (SetupLink, error) {
	if purpose != "setup" && purpose != "reset" {
		return SetupLink{}, fmt.Errorf("%w: purpose must be setup or reset", domain.ErrValidation)
	}
	var link SetupLink
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		m, err := s.staffInShop(ctx, tx, shopID, staffID)
		if err != nil {
			return err
		}
		if purpose == "reset" && !m.Pending {
			if _, err := tx.Exec(ctx, `UPDATE cd_staff SET pin_hash = '', pin_set_at = NULL, failed_attempts = 0, locked_until = NULL WHERE id = $1`, staffID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE cd_sessions SET revoked_at = $2 WHERE staff_id = $1 AND revoked_at IS NULL`, staffID, s.now()); err != nil {
				return err
			}
		}
		link, err = s.issueLinkTx(ctx, tx, shopID, staffID, purpose, createdBy, ttl)
		return err
	})
	return link, err
}

func (s *Store) issueLinkTx(ctx context.Context, tx pgx.Tx, shopID, staffID, purpose, createdBy string, ttl time.Duration) (SetupLink, error) {
	now := s.now()
	// Only the newest link works.
	if _, err := tx.Exec(ctx, `UPDATE cd_setup_links SET expires_at = $2 WHERE staff_id = $1 AND used_at IS NULL AND expires_at > $2`, staffID, now); err != nil {
		return SetupLink{}, err
	}
	link := SetupLink{Token: newLinkToken(), Purpose: purpose, ExpiresAt: now.Add(ttl)}
	_, err := tx.Exec(ctx, `INSERT INTO cd_setup_links (token_hash, staff_id, shop_id, purpose, created_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`, HashSecret(link.Token), staffID, shopID, purpose, createdBy, link.ExpiresAt)
	return link, err
}

// SetupLinkInfo checks a link without using it.
func (s *Store) SetupLinkInfo(ctx context.Context, token string) (SetupInfo, error) {
	var in SetupInfo
	err := s.pool.QueryRow(ctx, `SELECT sh.name, sh.slug, st.name, st.role, l.purpose, l.expires_at
		FROM cd_setup_links l JOIN cd_staff st ON st.id = l.staff_id JOIN cd_shops sh ON sh.id = l.shop_id
		WHERE l.token_hash = $1 AND l.used_at IS NULL AND l.expires_at > $2 AND st.active`,
		HashSecret(strings.TrimSpace(token)), s.now()).
		Scan(&in.ShopName, &in.ShopSlug, &in.StaffName, &in.Role, &in.Purpose, &in.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return in, ErrLinkInvalid
	}
	return in, err
}

// CompleteSetup uses a link once: sets the chosen PIN and signs the person in on this device.
func (s *Store) CompleteSetup(ctx context.Context, token, pin string, sessionTTL time.Duration) (string, Principal, error) {
	if err := checkNewPIN(pin); err != nil {
		return "", Principal{}, err
	}
	hash, err := hashPIN(pin)
	if err != nil {
		return "", Principal{}, err
	}
	var session string
	var p Principal
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		now := s.now()
		var st Staff
		err := tx.QueryRow(ctx, `SELECT st.id, st.shop_id, st.name, st.role
			FROM cd_setup_links l JOIN cd_staff st ON st.id = l.staff_id
			WHERE l.token_hash = $1 AND l.used_at IS NULL AND l.expires_at > $2 AND st.active
			FOR UPDATE OF l`, HashSecret(strings.TrimSpace(token)), now).Scan(&st.ID, &st.ShopID, &st.Name, &st.Role)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLinkInvalid
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE cd_setup_links SET used_at = $2 WHERE token_hash = $1`, HashSecret(strings.TrimSpace(token)), now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE cd_staff SET pin_hash = $2, pin_set_at = $3, failed_attempts = 0, locked_until = NULL WHERE id = $1`,
			st.ID, hash, now); err != nil {
			return err
		}
		session, p, err = s.newSession(ctx, tx, st, sessionTTL)
		return err
	})
	return session, p, err
}

// ChangePIN lets a signed-in person change their own PIN. Their other sessions are signed out.
func (s *Store) ChangePIN(ctx context.Context, staffID, currentSession, currentPIN, newPIN string) error {
	if err := checkNewPIN(newPIN); err != nil {
		return err
	}
	if currentPIN == newPIN {
		return ErrSamePIN
	}
	if err := s.checkPIN(ctx, staffID, currentPIN); err != nil {
		return err
	}
	hash, err := hashPIN(newPIN)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		now := s.now()
		if _, err := tx.Exec(ctx, `UPDATE cd_staff SET pin_hash = $2, pin_set_at = $3 WHERE id = $1`, staffID, hash, now); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE cd_sessions SET revoked_at = $3 WHERE staff_id = $1 AND token_hash <> $2 AND revoked_at IS NULL`,
			staffID, HashSecret(currentSession), now)
		return err
	})
}

// RemoveStaff deactivates someone: they are signed out and their links stop working.
// Nobody can remove themselves, and the last owner can't be removed.
func (s *Store) RemoveStaff(ctx context.Context, shopID, staffID, byStaffID string) error {
	if staffID == byStaffID {
		return ErrSelf
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// Serialise removals per shop so two owners can't remove each other at the same moment.
		if _, err := tx.Exec(ctx, `SELECT 1 FROM cd_shops WHERE id = $1 FOR UPDATE`, shopID); err != nil {
			return err
		}
		m, err := s.staffInShop(ctx, tx, shopID, staffID)
		if err != nil {
			return err
		}
		if m.Role == "owner" {
			var owners int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM cd_staff WHERE shop_id = $1 AND active AND role = 'owner'`, shopID).Scan(&owners); err != nil {
				return err
			}
			if owners <= 1 {
				return ErrLastOwner
			}
		}
		now := s.now()
		if _, err := tx.Exec(ctx, `UPDATE cd_staff SET active = false, removed_at = $2 WHERE id = $1`, staffID, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE cd_sessions SET revoked_at = $2 WHERE staff_id = $1 AND revoked_at IS NULL`, staffID, now); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE cd_setup_links SET expires_at = $2 WHERE staff_id = $1 AND used_at IS NULL AND expires_at > $2`, staffID, now)
		return err
	})
}

// OwnerID returns the first owner of a shop (used right after CreateShop to issue the owner's link).
func (s *Store) OwnerID(ctx context.Context, shopID string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id FROM cd_staff WHERE shop_id = $1 AND active AND role = 'owner' ORDER BY created_at LIMIT 1`, shopID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}

func checkNewPIN(pin string) error {
	if !pinPattern.MatchString(pin) {
		return fmt.Errorf("%w: PIN must be 4 digits", domain.ErrValidation)
	}
	if domain.WeakPIN(pin) {
		return ErrWeakPIN
	}
	return nil
}
