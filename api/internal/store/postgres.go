package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"counter-drop/api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	queryTimeout         = 5 * time.Second
	collectedDeleteGrace = 15 * time.Minute
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}

func (s *PostgresStore) ApplyMigrations(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS cd_schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}

	for _, file := range files {
		version := file
		var applied bool
		err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM cd_schema_migrations WHERE version = $1)`, version).Scan(&applied)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if applied {
			continue
		}

		body, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, string(body)); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `INSERT INTO cd_schema_migrations (version) VALUES ($1)`, version)
			return err
		})
		if err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
	}

	return nil
}

func (s *PostgresStore) SeedDemoShop(ctx context.Context) error {
	prices := map[string]int{
		"bw_page_paise":    200,
		"color_page_paise": 1000,
	}
	pricesJSON, err := json.Marshal(prices)
	if err != nil {
		return err
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO cd_shops (id, slug, name, intake_paused, prices, wait_minutes)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (slug) DO NOTHING
	`, "shop_demo", "demo-print", "Demo Print Counter", false, pricesJSON, 5)
	return err
}

func (s *PostgresStore) GetShopBySlug(slug string) (domain.Shop, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	var shop domain.Shop
	var pricesJSON []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, slug, name, intake_paused, prices, wait_minutes
		FROM cd_shops
		WHERE slug = $1
	`, slug).Scan(&shop.ID, &shop.Slug, &shop.Name, &shop.IntakePaused, &pricesJSON, &shop.WaitMinutes)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Shop{}, ErrNotFound
	}
	if err != nil {
		return domain.Shop{}, err
	}
	if err := json.Unmarshal(pricesJSON, &shop.Prices); err != nil {
		return domain.Shop{}, err
	}
	return shop, nil
}

func (s *PostgresStore) CreateJob(shopSlug string, input CreateJobInput) (domain.Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	var job domain.Job
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var shop domain.Shop
		err := tx.QueryRow(ctx, `
			SELECT id, slug, name, intake_paused
			FROM cd_shops
			WHERE slug = $1
		`, shopSlug).Scan(&shop.ID, &shop.Slug, &shop.Name, &shop.IntakePaused)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if shop.IntakePaused {
			return ErrIntakePaused
		}

		var tokenNo int
		err = tx.QueryRow(ctx, `
			INSERT INTO cd_token_counters (shop_id, prefix, last_no)
			VALUES ($1, 'A', 1)
			ON CONFLICT (shop_id, prefix)
			DO UPDATE SET last_no = cd_token_counters.last_no + 1
			RETURNING last_no
		`, shop.ID).Scan(&tokenNo)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		settings := input.Settings
		if settings == nil {
			settings = map[string]any{}
		}
		settingsJSON, err := json.Marshal(settings)
		if err != nil {
			return err
		}

		job = domain.Job{
			ID:           newID("job"),
			ShopID:       shop.ID,
			Token:        fmt.Sprintf("A-%02d", tokenNo),
			Secret:       randomSecret(),
			CustomerName: input.CustomerName,
			Settings:     settings,
			Files:        make([]domain.JobFile, 0, len(input.Files)),
			State:        domain.JobStateNew,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO cd_jobs (id, shop_id, token, secret, customer_name, settings, state, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, job.ID, job.ShopID, job.Token, job.Secret, job.CustomerName, settingsJSON, job.State, job.CreatedAt, job.UpdatedAt)
		if err != nil {
			return err
		}

		for _, file := range input.Files {
			jobFile := domain.JobFile{
				ID:           newID("file"),
				Filename:     file.Filename,
				Size:         file.Size,
				Mime:         file.Mime,
				UploadStatus: domain.UploadStatusPending,
				DeleteStatus: domain.DeleteStatusActive,
			}
			jobFile.ObjectKey = fmt.Sprintf("cd/%s/%s/%s", job.ShopID, job.ID, jobFile.ID)
			_, err = tx.Exec(ctx, `
				INSERT INTO cd_job_files (id, job_id, filename, size_bytes, mime, pages, object_key, upload_status, delete_status)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`, jobFile.ID, job.ID, jobFile.Filename, jobFile.Size, jobFile.Mime, jobFile.Pages, jobFile.ObjectKey, jobFile.UploadStatus, jobFile.DeleteStatus)
			if err != nil {
				return err
			}
			job.Files = append(job.Files, jobFile)
		}

		return nil
	})
	if err != nil {
		return domain.Job{}, err
	}
	return job, nil
}

func (s *PostgresStore) ListShopQueue(shopSlug string) (QueueSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	shop, err := s.GetShopBySlug(shopSlug)
	if err != nil {
		return QueueSnapshot{}, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, shop_id, token, secret, customer_name, settings, state, created_at, updated_at, claimed_at, ready_at, collected_at
		FROM cd_jobs
		WHERE shop_id = $1
		ORDER BY
			CASE state
				WHEN 'new' THEN 1
				WHEN 'claimed' THEN 2
				WHEN 'ready' THEN 3
				WHEN 'collected' THEN 4
				WHEN 'cancelled' THEN 5
				ELSE 6
			END,
			created_at ASC
	`, shop.ID)
	if err != nil {
		return QueueSnapshot{}, err
	}
	defer rows.Close()

	jobs := []domain.Job{}
	for rows.Next() {
		job, err := s.scanJob(rows)
		if err != nil {
			return QueueSnapshot{}, err
		}
		job.Secret = ""
		job.Files, err = s.getFiles(ctx, job.ID)
		if err != nil {
			return QueueSnapshot{}, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return QueueSnapshot{}, err
	}

	return QueueSnapshot{Shop: shop, Jobs: jobs}, nil
}

func (s *PostgresStore) GetJob(jobID string, secret string) (domain.Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	job, err := s.getJob(ctx, jobID)
	if err != nil {
		return domain.Job{}, err
	}
	if secret == "" || secret != job.Secret {
		return domain.Job{}, ErrBadSecret
	}
	return job, nil
}

func (s *PostgresStore) SubmitJob(jobID string, secret string) (domain.Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	var submitted domain.Job
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		job, err := s.getJobForUpdate(ctx, tx, jobID)
		if err != nil {
			return err
		}
		if secret == "" || secret != job.Secret {
			return ErrBadSecret
		}

		now := time.Now().UTC()
		_, err = tx.Exec(ctx, `
			UPDATE cd_job_files
			SET upload_status = $2
			WHERE job_id = $1
		`, job.ID, domain.UploadStatusUploaded)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE cd_jobs
			SET updated_at = $2
			WHERE id = $1
		`, job.ID, now)
		if err != nil {
			return err
		}
		job.UpdatedAt = now
		for i := range job.Files {
			job.Files[i].UploadStatus = domain.UploadStatusUploaded
		}
		submitted = job
		return nil
	})
	if err != nil {
		return domain.Job{}, err
	}
	return submitted, nil
}

func (s *PostgresStore) ApplyJobAction(jobID string, action string) (domain.Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	var updated domain.Job
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		job, err := s.getJobForUpdate(ctx, tx, jobID)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		if err := job.Apply(action, now); err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			UPDATE cd_jobs
			SET state = $2, updated_at = $3, claimed_at = $4, ready_at = $5, collected_at = $6
			WHERE id = $1
		`, job.ID, job.State, job.UpdatedAt, job.ClaimedAt, job.ReadyAt, job.CollectedAt)
		if err != nil {
			return err
		}

		if err := s.scheduleFileDeletionTx(ctx, tx, &job, action, now); err != nil {
			return err
		}

		updated = job
		return nil
	})
	if err != nil {
		return domain.Job{}, err
	}
	return updated, nil
}

func (s *PostgresStore) scheduleFileDeletionTx(ctx context.Context, tx pgx.Tx, job *domain.Job, action string, now time.Time) error {
	var deleteAfter time.Time
	switch action {
	case "collected":
		deleteAfter = now.Add(collectedDeleteGrace)
	case "cancel":
		deleteAfter = now
	default:
		return nil
	}

	_, err := tx.Exec(ctx, `
		UPDATE cd_job_files
		SET delete_status = $2, delete_after = $3
		WHERE job_id = $1 AND deleted_at IS NULL
	`, job.ID, domain.DeleteStatusPending, deleteAfter)
	if err != nil {
		return err
	}

	for i := range job.Files {
		if job.Files[i].DeletedAt != nil {
			continue
		}
		job.Files[i].DeleteStatus = domain.DeleteStatusPending
		value := deleteAfter.UTC()
		job.Files[i].DeleteAfter = &value
	}
	return nil
}

func (s *PostgresStore) getJob(ctx context.Context, jobID string) (domain.Job, error) {
	job, err := s.scanJob(s.pool.QueryRow(ctx, `
		SELECT id, shop_id, token, secret, customer_name, settings, state, created_at, updated_at, claimed_at, ready_at, collected_at
		FROM cd_jobs
		WHERE id = $1
	`, jobID))
	if err != nil {
		return domain.Job{}, err
	}
	job.Files, err = s.getFiles(ctx, job.ID)
	if err != nil {
		return domain.Job{}, err
	}
	return job, nil
}

func (s *PostgresStore) getJobForUpdate(ctx context.Context, tx pgx.Tx, jobID string) (domain.Job, error) {
	job, err := s.scanJob(tx.QueryRow(ctx, `
		SELECT id, shop_id, token, secret, customer_name, settings, state, created_at, updated_at, claimed_at, ready_at, collected_at
		FROM cd_jobs
		WHERE id = $1
		FOR UPDATE
	`, jobID))
	if err != nil {
		return domain.Job{}, err
	}
	job.Files, err = s.getFilesTx(ctx, tx, job.ID)
	if err != nil {
		return domain.Job{}, err
	}
	return job, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func (s *PostgresStore) scanJob(row scanner) (domain.Job, error) {
	var job domain.Job
	var settingsJSON []byte
	err := row.Scan(
		&job.ID,
		&job.ShopID,
		&job.Token,
		&job.Secret,
		&job.CustomerName,
		&settingsJSON,
		&job.State,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.ClaimedAt,
		&job.ReadyAt,
		&job.CollectedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Job{}, ErrNotFound
	}
	if err != nil {
		return domain.Job{}, err
	}
	if err := json.Unmarshal(settingsJSON, &job.Settings); err != nil {
		return domain.Job{}, err
	}
	if job.Settings == nil {
		job.Settings = map[string]any{}
	}
	normalizeJobTimes(&job)
	return job, nil
}

func (s *PostgresStore) getFiles(ctx context.Context, jobID string) ([]domain.JobFile, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, filename, size_bytes, mime, pages, object_key, upload_status, delete_status, delete_after, deleted_at
		FROM cd_job_files
		WHERE job_id = $1
		ORDER BY created_at, id
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFiles(rows)
}

func (s *PostgresStore) getFilesTx(ctx context.Context, tx pgx.Tx, jobID string) ([]domain.JobFile, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, filename, size_bytes, mime, pages, object_key, upload_status, delete_status, delete_after, deleted_at
		FROM cd_job_files
		WHERE job_id = $1
		ORDER BY created_at, id
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFiles(rows)
}

func scanFiles(rows pgx.Rows) ([]domain.JobFile, error) {
	files := []domain.JobFile{}
	for rows.Next() {
		var file domain.JobFile
		var deleteAfter pgtype.Timestamptz
		var deletedAt pgtype.Timestamptz
		if err := rows.Scan(
			&file.ID,
			&file.Filename,
			&file.Size,
			&file.Mime,
			&file.Pages,
			&file.ObjectKey,
			&file.UploadStatus,
			&file.DeleteStatus,
			&deleteAfter,
			&deletedAt,
		); err != nil {
			return nil, err
		}
		if deleteAfter.Valid {
			value := deleteAfter.Time.UTC()
			file.DeleteAfter = &value
		}
		if deletedAt.Valid {
			value := deletedAt.Time.UTC()
			file.DeletedAt = &value
		}
		files = append(files, file)
	}
	return files, rows.Err()
}
func normalizeJobTimes(job *domain.Job) {
	job.CreatedAt = job.CreatedAt.UTC()
	job.UpdatedAt = job.UpdatedAt.UTC()
	if job.ClaimedAt != nil {
		claimedAt := job.ClaimedAt.UTC()
		job.ClaimedAt = &claimedAt
	}
	if job.ReadyAt != nil {
		readyAt := job.ReadyAt.UTC()
		job.ReadyAt = &readyAt
	}
	if job.CollectedAt != nil {
		collectedAt := job.CollectedAt.UTC()
		job.CollectedAt = &collectedAt
	}
}
