package store

import (
	"context"
	"time"

	"counter-drop/api/internal/domain"

	"github.com/jackc/pgx/v5"
)

type DueFile struct {
	ID    string
	JobID string
	Key   string
}

// ClaimDueFiles returns up to limit files whose delete_after has passed (FSD §12 worker step 1).
func (s *Store) ClaimDueFiles(ctx context.Context, limit int) ([]DueFile, error) {
	now := s.now()
	rows, err := s.pool.Query(ctx, `SELECT id, job_id, COALESCE(object_key, '') FROM cd_job_files
		WHERE deleted_at IS NULL AND delete_after IS NOT NULL AND delete_after <= $1
		  AND (next_attempt_at IS NULL OR next_attempt_at <= $1)
		ORDER BY delete_after LIMIT $2`, now, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (DueFile, error) {
		var f DueFile
		err := r.Scan(&f.ID, &f.JobID, &f.Key)
		return f, err
	})
}

// MarkFileDeleted records a successful delete and drops the file name. It returns the job's shop
// and whether every file of the job is now gone (the job's name is cleared then).
func (s *Store) MarkFileDeleted(ctx context.Context, f DueFile) (shopID string, jobDone bool, err error) {
	now := s.now()
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE cd_job_files SET delete_status = 'deleted', deleted_at = $2, object_key = NULL,
			filename = '', last_delete_error = '' WHERE id = $1`, f.ID, now); err != nil {
			return err
		}
		var remaining int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM cd_job_files WHERE job_id = $1 AND deleted_at IS NULL`, f.JobID).Scan(&remaining); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT shop_id FROM cd_jobs WHERE id = $1`, f.JobID).Scan(&shopID); err != nil {
			return err
		}
		if remaining == 0 {
			jobDone = true
			_, err := tx.Exec(ctx, `UPDATE cd_jobs SET customer_name = NULL, files_deleted_at = $2, updated_at = $2
				WHERE id = $1 AND files_deleted_at IS NULL`, f.JobID, now)
			return err
		}
		return nil
	})
	return shopID, jobDone, err
}

// MarkFileDeleteFailed schedules a retry with backoff (1, 5, 15 minutes).
func (s *Store) MarkFileDeleteFailed(ctx context.Context, fileID string, cause error) (attempts int, err error) {
	backoff := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute}
	err = s.pool.QueryRow(ctx, `UPDATE cd_job_files SET delete_attempts = delete_attempts + 1, delete_status = 'failed',
		last_delete_error = left($2, 300) WHERE id = $1 RETURNING delete_attempts`, fileID, cause.Error()).Scan(&attempts)
	if err != nil {
		return 0, err
	}
	wait := backoff[len(backoff)-1]
	if attempts-1 < len(backoff) {
		wait = backoff[attempts-1]
	}
	_, err = s.pool.Exec(ctx, `UPDATE cd_job_files SET next_attempt_at = $2 WHERE id = $1`, fileID, s.now().Add(wait))
	return attempts, err
}

// AbandonDrafts cancels walk-in drafts with no activity for the abandon window and schedules
// their files for deletion. Returns the affected jobs.
func (s *Store) AbandonDrafts(ctx context.Context) ([]domain.Job, error) {
	cutoff := s.now().Add(-s.policy.AbandonAfter)
	rows, err := s.pool.Query(ctx, `SELECT id FROM cd_jobs WHERE state = 'uploading' AND updated_at < $1 LIMIT 200`, cutoff)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	var out []domain.Job
	for _, id := range ids {
		err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
			j, err := lockJob(ctx, tx, id)
			if err != nil {
				return err
			}
			if err := s.apply(ctx, tx, &j, ActInput{Action: domain.ActionAbandon, Actor: domain.Actor{Type: domain.ActorSystem, Name: "system"}}); err != nil {
				return err
			}
			out = append(out, j)
			return nil
		})
		if err != nil && err != domain.ErrInvalidTransition {
			return out, err
		}
	}
	return out, nil
}

type DeletionHealth struct {
	DueNow          int        `json:"dueNow"`
	Deleted24h      int        `json:"deleted24h"`
	Failed          int        `json:"failed"`
	OldestUndeleted *time.Time `json:"oldestUndeleted,omitempty"`
}

func (s *Store) DeletionHealth(ctx context.Context) (DeletionHealth, error) {
	now := s.now()
	var h DeletionHealth
	err := s.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE deleted_at IS NULL AND delete_after <= $1),
		count(*) FILTER (WHERE deleted_at > $1 - interval '24 hours'),
		count(*) FILTER (WHERE deleted_at IS NULL AND delete_status = 'failed'),
		min(created_at) FILTER (WHERE deleted_at IS NULL)
		FROM cd_job_files`, now).Scan(&h.DueNow, &h.Deleted24h, &h.Failed, &h.OldestUndeleted)
	return h, err
}
