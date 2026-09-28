// Package store is the Postgres persistence layer. Every state change runs in one transaction
// with the job row locked, so two counters can never act on the same job at once.
package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"counter-drop/api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrBadSecret         = errors.New("invalid ticket secret")
	ErrShopPaused        = errors.New("shop is paused")
	ErrShopOffline       = errors.New("shop is offline")
	ErrAlreadyClaimed    = errors.New("job already claimed")
	ErrLaneEmpty         = errors.New("nothing to claim")
	ErrUploadsIncomplete = errors.New("uploads incomplete")
	ErrPriceChanged      = errors.New("price changed")
	ErrBadPIN            = errors.New("wrong PIN")
	ErrLocked            = errors.New("too many attempts, try again later")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrNotEditable       = errors.New("job can no longer be changed")
	ErrSetupPending      = errors.New("PIN not set up yet")
	ErrLinkInvalid       = errors.New("setup link expired or already used")
	ErrWeakPIN           = errors.New("PIN too easy to guess")
	ErrSamePIN           = errors.New("new PIN is the same as the old one")
	ErrLastOwner         = errors.New("a shop needs at least one owner")
	ErrSelf              = errors.New("not allowed on your own account")
)

type Store struct {
	pool   *pgxpool.Pool
	now    func() time.Time
	policy domain.Policy
}

func New(ctx context.Context, databaseURL string, policy domain.Policy) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: pool, now: func() time.Time { return time.Now().UTC() }, policy: policy}, nil
}

// SetClock replaces the clock (tests).
func (s *Store) SetClock(now func() time.Time) { s.now = now }

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// ApplyMigrations runs every *.sql file in dir that has not been applied yet, each in its own transaction.
func (s *Store) ApplyMigrations(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %q: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	if _, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS cd_schema_migrations (
		version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}
	for _, file := range files {
		var applied bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM cd_schema_migrations WHERE version = $1)`, file).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			return err
		}
		err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, string(body)); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `INSERT INTO cd_schema_migrations (version) VALUES ($1)`, file)
			return err
		})
		if err != nil {
			return fmt.Errorf("apply migration %s: %w", file, err)
		}
	}
	return nil
}

// --- helpers -------------------------------------------------------------------

func newID(prefix string) string {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%x%s", prefix, time.Now().UTC().Unix(), hex.EncodeToString(b))
}

// NewSecret returns 32 random bytes as hex (ticket secrets, session tokens).
func NewSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func HashSecret(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func equalHash(rawSecret, storedHash string) bool {
	if rawSecret == "" || storedHash == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(HashSecret(rawSecret)), []byte(storedHash)) == 1
}
