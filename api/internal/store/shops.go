package store

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"counter-drop/api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const shopColumns = `id, slug, name, address, status, online_state, pause_message, timezone, opens_at, closes_at, price_list, hold_days`

func scanShop(row pgx.Row) (domain.Shop, error) {
	var sh domain.Shop
	var prices []byte
	err := row.Scan(&sh.ID, &sh.Slug, &sh.Name, &sh.Address, &sh.Status, &sh.OnlineState, &sh.PauseMessage,
		&sh.Timezone, &sh.OpensAt, &sh.ClosesAt, &prices, &sh.HoldDays)
	if errors.Is(err, pgx.ErrNoRows) {
		return sh, ErrNotFound
	}
	if err != nil {
		return sh, err
	}
	if err := json.Unmarshal(prices, &sh.Prices); err != nil {
		return sh, err
	}
	return sh, nil
}

func (s *Store) loadLanes(ctx context.Context, q pgx.Tx, sh *domain.Shop) error {
	var rows pgx.Rows
	var err error
	const sql = `SELECT id, letter, name, rule FROM cd_lanes WHERE shop_id = $1 ORDER BY sort, letter`
	if q != nil {
		rows, err = q.Query(ctx, sql, sh.ID)
	} else {
		rows, err = s.pool.Query(ctx, sql, sh.ID)
	}
	if err != nil {
		return err
	}
	defer rows.Close()
	sh.Lanes = []domain.Lane{}
	for rows.Next() {
		var l domain.Lane
		if err := rows.Scan(&l.ID, &l.Letter, &l.Name, &l.Rule); err != nil {
			return err
		}
		sh.Lanes = append(sh.Lanes, l)
	}
	return rows.Err()
}

func (s *Store) GetShopBySlug(ctx context.Context, slug string) (domain.Shop, error) {
	sh, err := scanShop(s.pool.QueryRow(ctx, `SELECT `+shopColumns+` FROM cd_shops WHERE slug = $1`, strings.ToLower(slug)))
	if err != nil {
		return sh, err
	}
	return sh, s.loadLanes(ctx, nil, &sh)
}

func (s *Store) GetShopByID(ctx context.Context, id string) (domain.Shop, error) {
	sh, err := scanShop(s.pool.QueryRow(ctx, `SELECT `+shopColumns+` FROM cd_shops WHERE id = $1`, id))
	if err != nil {
		return sh, err
	}
	return sh, s.loadLanes(ctx, nil, &sh)
}

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,28}[a-z0-9]$`)

var reservedSlugs = map[string]bool{"admin": true, "api": true, "shop": true, "tv": true, "s": true, "t": true, "nearby": true, "www": true, "app": true, "help": true, "scan": true}

type CreateShopInput struct {
	Slug      string
	Name      string
	Address   string
	OwnerName string
	OwnerPIN  string // empty: the owner is pending and chooses a PIN with a setup link
}

// CreateShop adds a live shop with two lanes (A: B/W, B: Colour), default prices and an owner.
func (s *Store) CreateShop(ctx context.Context, in CreateShopInput) (domain.Shop, error) {
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	if !slugPattern.MatchString(in.Slug) || reservedSlugs[in.Slug] {
		return domain.Shop{}, fmt.Errorf("%w: slug must be 3–30 lowercase letters, digits or hyphens", domain.ErrValidation)
	}
	if n := len(strings.TrimSpace(in.Name)); n < 3 || n > 60 {
		return domain.Shop{}, fmt.Errorf("%w: shop name must be 3–60 characters", domain.ErrValidation)
	}
	id := newID("shop")
	prices, _ := json.Marshal(domain.DefaultPriceList())
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO cd_shops (id, slug, name, address, intake_paused, prices, wait_minutes, price_list)
			VALUES ($1, $2, $3, $4, false, '{}'::jsonb, 0, $5)`, id, in.Slug, strings.TrimSpace(in.Name), in.Address, prices); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO cd_lanes (id, shop_id, letter, name, rule, sort) VALUES
			($1, $2, 'A', 'B/W', 'bw', 0), ($3, $2, 'B', 'Colour', 'colour', 1)`, newID("lane"), id, newID("lane")); err != nil {
			return err
		}
		_, err := addStaffTx(ctx, tx, id, in.OwnerName, "owner", in.OwnerPIN)
		return err
	})
	if err != nil {
		if strings.Contains(err.Error(), "cd_shops_slug_key") {
			return domain.Shop{}, fmt.Errorf("%w: that link name is taken", domain.ErrValidation)
		}
		return domain.Shop{}, err
	}
	return s.GetShopByID(ctx, id)
}

// SeedDemo creates the demo shop "demo-print" with staff Owner (PIN 1234) and Kavita (PIN 1111) if missing.
func (s *Store) SeedDemo(ctx context.Context) error {
	if _, err := s.GetShopBySlug(ctx, "demo-print"); err == nil {
		var n int
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM cd_staff st JOIN cd_shops sh ON sh.id = st.shop_id WHERE sh.slug = 'demo-print'`).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
		sh, _ := s.GetShopBySlug(ctx, "demo-print")
		if err := s.AddStaff(ctx, sh.ID, "Owner", "owner", "1234"); err != nil {
			return err
		}
		return s.AddStaff(ctx, sh.ID, "Kavita", "staff", "1111")
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	sh, err := s.CreateShop(ctx, CreateShopInput{Slug: "demo-print", Name: "Demo Print Counter", Address: "Dadar West, Mumbai", OwnerName: "Owner", OwnerPIN: "1234"})
	if err != nil {
		return err
	}
	return s.AddStaff(ctx, sh.ID, "Kavita", "staff", "1111")
}

func (s *Store) SetShopState(ctx context.Context, shopID string, state domain.OnlineState, msg string) (domain.Shop, error) {
	if state != domain.ShopOnline && state != domain.ShopPaused && state != domain.ShopOffline {
		return domain.Shop{}, fmt.Errorf("%w: state must be online, paused or offline", domain.ErrValidation)
	}
	if len(msg) > 140 {
		msg = msg[:140]
	}
	if _, err := s.pool.Exec(ctx, `UPDATE cd_shops SET online_state = $2, pause_message = $3, intake_paused = ($2 <> 'online'), updated_at = now() WHERE id = $1`,
		shopID, string(state), msg); err != nil {
		return domain.Shop{}, err
	}
	return s.GetShopByID(ctx, shopID)
}

type ShopProfile struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	OpensAt  string `json:"opensAt"`
	ClosesAt string `json:"closesAt"`
	HoldDays int    `json:"holdDays,omitempty"` // 0 keeps the current value
}

var hhmm = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func (s *Store) UpdateShopSettings(ctx context.Context, shopID string, p ShopProfile, prices domain.PriceList) (domain.Shop, error) {
	if n := len(strings.TrimSpace(p.Name)); n < 3 || n > 60 {
		return domain.Shop{}, fmt.Errorf("%w: shop name must be 3–60 characters", domain.ErrValidation)
	}
	if !hhmm.MatchString(p.OpensAt) || !hhmm.MatchString(p.ClosesAt) || p.ClosesAt <= p.OpensAt {
		return domain.Shop{}, fmt.Errorf("%w: closing time must be after opening time (HH:MM)", domain.ErrValidation)
	}
	if err := prices.Validate(); err != nil {
		return domain.Shop{}, err
	}
	cur, err := s.GetShopByID(ctx, shopID)
	if err != nil {
		return domain.Shop{}, err
	}
	if p.HoldDays == 0 {
		p.HoldDays = cur.HoldDays
	}
	if p.HoldDays < 1 || p.HoldDays > 7 {
		return domain.Shop{}, fmt.Errorf("%w: uncollected jobs can be kept 1–7 days", domain.ErrValidation)
	}
	prices.Version = cur.Prices.Version + 1
	b, _ := json.Marshal(prices)
	if _, err := s.pool.Exec(ctx, `UPDATE cd_shops SET name = $2, address = $3, opens_at = $4, closes_at = $5, price_list = $6, hold_days = $7, updated_at = now() WHERE id = $1`,
		shopID, strings.TrimSpace(p.Name), strings.TrimSpace(p.Address), p.OpensAt, p.ClosesAt, b, p.HoldDays); err != nil {
		return domain.Shop{}, err
	}
	return s.GetShopByID(ctx, shopID)
}

// --- staff and sessions ---------------------------------------------------------

type Staff struct {
	ID     string `json:"id"`
	ShopID string `json:"shopId"`
	Name   string `json:"name"`
	Role   string `json:"role"`
}

type Principal struct {
	Staff     Staff
	ExpiresAt time.Time
}

var pinPattern = regexp.MustCompile(`^\d{4}$`)

func hashPIN(pin string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	const iter = 120_000
	key, err := pbkdf2.Key(sha256.New, pin, salt, iter, 32)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", iter, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func verifyPIN(pin, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[2])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, pin, salt, iter, len(want))
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}

// addStaffTx adds a staff member. An empty PIN makes them "pending": they choose a PIN with a setup link.
func addStaffTx(ctx context.Context, tx pgx.Tx, shopID, name, role, pin string) (string, error) {
	name = strings.TrimSpace(name)
	if n := len([]rune(name)); n < 2 || n > 30 {
		return "", fmt.Errorf("%w: name must be 2–30 characters", domain.ErrValidation)
	}
	if role != "owner" && role != "staff" {
		return "", fmt.Errorf("%w: role must be owner or staff", domain.ErrValidation)
	}
	hash := ""
	var pinSetAt *time.Time
	if pin != "" {
		if !pinPattern.MatchString(pin) {
			return "", fmt.Errorf("%w: PIN must be 4 digits", domain.ErrValidation)
		}
		h, err := hashPIN(pin)
		if err != nil {
			return "", err
		}
		hash = h
		now := time.Now()
		pinSetAt = &now
	}
	id := newID("staff")
	_, err := tx.Exec(ctx, `INSERT INTO cd_staff (id, shop_id, name, role, pin_hash, pin_set_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, shopID, name, role, hash, pinSetAt)
	if err != nil && strings.Contains(err.Error(), "uq_cd_staff_active_name") {
		return "", fmt.Errorf("%w: someone called %q already works here", domain.ErrValidation, name)
	}
	return id, err
}

// AddStaff adds someone with a known PIN (demo seed and tests). Real shops use InviteStaff.
func (s *Store) AddStaff(ctx context.Context, shopID, name, role, pin string) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		_, err := addStaffTx(ctx, tx, shopID, name, role, pin)
		return err
	})
}

// StaffNames lists active staff for the login tiles.
func (s *Store) StaffNames(ctx context.Context, slug string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT st.name FROM cd_staff st JOIN cd_shops sh ON sh.id = st.shop_id
		WHERE sh.slug = $1 AND st.active AND st.pin_hash <> '' ORDER BY (st.role = 'owner') DESC, lower(st.name)`, strings.ToLower(slug))
	if err != nil {
		return nil, err
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	if names == nil {
		names = []string{} // shop exists but nobody has finished setup yet
	}
	return names, nil
}

// Login checks a staff PIN and returns a new session token (shown once).
func (s *Store) Login(ctx context.Context, slug, name, pin string, ttl time.Duration) (string, Principal, error) {
	var st Staff
	err := s.pool.QueryRow(ctx, `SELECT st.id, st.shop_id, st.name, st.role
		FROM cd_staff st JOIN cd_shops sh ON sh.id = st.shop_id
		WHERE sh.slug = $1 AND st.name = $2 AND st.active`, strings.ToLower(slug), name).
		Scan(&st.ID, &st.ShopID, &st.Name, &st.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", Principal{}, ErrBadPIN
	}
	if err != nil {
		return "", Principal{}, err
	}
	if err := s.checkPIN(ctx, st.ID, pin); err != nil {
		return "", Principal{}, err
	}
	return s.newSession(ctx, s.pool, st, ttl)
}

// checkPIN verifies a staff member's PIN with the lockout rule: 5 wrong tries lock the account for 15 minutes.
func (s *Store) checkPIN(ctx context.Context, staffID, pin string) error {
	now := s.now()
	var hash string
	var failed int
	var lockedUntil *time.Time
	err := s.pool.QueryRow(ctx, `SELECT pin_hash, failed_attempts, locked_until FROM cd_staff WHERE id = $1 AND active`, staffID).
		Scan(&hash, &failed, &lockedUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrBadPIN
	}
	if err != nil {
		return err
	}
	if hash == "" {
		return ErrSetupPending
	}
	if lockedUntil != nil && now.Before(*lockedUntil) {
		return ErrLocked
	}
	if !verifyPIN(pin, hash) {
		failed++
		var lock *time.Time
		if failed >= 5 {
			t := now.Add(15 * time.Minute)
			lock = &t
			failed = 0
		}
		_, _ = s.pool.Exec(ctx, `UPDATE cd_staff SET failed_attempts = $2, locked_until = $3 WHERE id = $1`, staffID, failed, lock)
		if lock != nil {
			return ErrLocked
		}
		return ErrBadPIN
	}
	_, err = s.pool.Exec(ctx, `UPDATE cd_staff SET failed_attempts = 0, locked_until = NULL WHERE id = $1`, staffID)
	return err
}

type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (s *Store) newSession(ctx context.Context, q execer, st Staff, ttl time.Duration) (string, Principal, error) {
	token := NewSecret()
	exp := s.now().Add(ttl)
	if _, err := q.Exec(ctx, `INSERT INTO cd_sessions (token_hash, staff_id, shop_id, expires_at) VALUES ($1, $2, $3, $4)`,
		HashSecret(token), st.ID, st.ShopID, exp); err != nil {
		return "", Principal{}, err
	}
	return token, Principal{Staff: st, ExpiresAt: exp}, nil
}

func (s *Store) Session(ctx context.Context, token string) (Principal, error) {
	if token == "" {
		return Principal{}, ErrUnauthorized
	}
	var p Principal
	err := s.pool.QueryRow(ctx, `SELECT st.id, st.shop_id, st.name, st.role, se.expires_at
		FROM cd_sessions se JOIN cd_staff st ON st.id = se.staff_id
		WHERE se.token_hash = $1 AND se.revoked_at IS NULL AND se.expires_at > $2 AND st.active`,
		HashSecret(token), s.now()).Scan(&p.Staff.ID, &p.Staff.ShopID, &p.Staff.Name, &p.Staff.Role, &p.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Principal{}, ErrUnauthorized
	}
	return p, err
}

func (s *Store) Logout(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `UPDATE cd_sessions SET revoked_at = now() WHERE token_hash = $1`, HashSecret(token))
	return err
}
