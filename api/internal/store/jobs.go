package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"counter-drop/api/internal/domain"
	"counter-drop/api/internal/storage"

	"github.com/jackc/pgx/v5"
)

const jobColumns = `j.id, j.shop_id, j.channel, COALESCE(j.lane_id, ''), COALESCE(l.letter, ''), COALESCE(j.token, ''),
	COALESCE(j.customer_name, ''), j.state, j.price_total_paise, j.pages_total, j.pages_to_confirm, j.ready_by,
	j.claimed_by, j.cancel_reason, j.paid_method, COALESCE(j.business_day::text, ''), j.created_at, j.updated_at,
	j.queued_at, j.claimed_at, j.ready_at, j.collected_at, j.cancelled_at, j.files_deleted_at,
	j.copies_delete_requested_at, j.copies_deleted_at, j.copies_deleted_by`

const jobFrom = ` FROM cd_jobs j LEFT JOIN cd_lanes l ON l.id = j.lane_id `

func scanJob(row pgx.Row) (domain.Job, error) {
	var j domain.Job
	var state string
	err := row.Scan(&j.ID, &j.ShopID, &j.Channel, &j.LaneID, &j.Lane, &j.Token, &j.CustomerName, &state,
		&j.PriceTotal, &j.PagesTotal, &j.PagesToConfirm, &j.ReadyBy, &j.ClaimedBy, &j.CancelReason, &j.PaidMethod,
		&j.BusinessDay, &j.CreatedAt, &j.UpdatedAt, &j.QueuedAt, &j.ClaimedAt, &j.ReadyAt, &j.CollectedAt,
		&j.CancelledAt, &j.FilesDeletedAt, &j.CopiesDeleteRequestedAt, &j.CopiesDeletedAt, &j.CopiesDeletedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return j, ErrNotFound
	}
	j.State = domain.ParseJobState(state)
	j.Files = []domain.JobFile{}
	return j, err
}

const fileColumns = `id, job_id, filename, size_bytes, mime, pages, pages_status, settings, COALESCE(object_key, ''),
	upload_status, delete_status, delete_after, deleted_at, printed_at, print_opens, downloaded_at, downloaded_by, downloads`

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// attachFiles loads the files of many jobs in one query (no N+1).
func attachFiles(ctx context.Context, q querier, jobs []*domain.Job) error {
	if len(jobs) == 0 {
		return nil
	}
	ids := make([]string, len(jobs))
	byID := map[string]*domain.Job{}
	for i, j := range jobs {
		ids[i] = j.ID
		byID[j.ID] = j
	}
	rows, err := q.Query(ctx, `SELECT `+fileColumns+` FROM cd_job_files WHERE job_id = ANY($1) AND removed_at IS NULL ORDER BY created_at, id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var f domain.JobFile
		var jobID string
		var settings []byte
		if err := rows.Scan(&f.ID, &jobID, &f.Filename, &f.Size, &f.Mime, &f.Pages, &f.PagesStatus, &settings,
			&f.ObjectKey, &f.UploadStatus, &f.DeleteStatus, &f.DeleteAfter, &f.DeletedAt,
			&f.PrintedAt, &f.PrintOpens, &f.DownloadedAt, &f.DownloadedBy, &f.Downloads); err != nil {
			return err
		}
		_ = json.Unmarshal(settings, &f.Settings)
		f.Settings = domain.NormaliseSettings(f.Settings, domain.FileKind(f.Mime))
		if f.DeletedAt != nil {
			f.Filename = "" // names are dropped once files are gone (FS-12.3)
		}
		if j := byID[jobID]; j != nil {
			j.Files = append(j.Files, f)
		}
	}
	return rows.Err()
}

func (s *Store) GetJob(ctx context.Context, id string) (domain.Job, error) {
	j, err := scanJob(s.pool.QueryRow(ctx, `SELECT `+jobColumns+jobFrom+` WHERE j.id = $1`, id))
	if err != nil {
		return j, err
	}
	return j, attachFiles(ctx, s.pool, []*domain.Job{&j})
}

func lockJob(ctx context.Context, tx pgx.Tx, id string) (domain.Job, error) {
	j, err := scanJob(tx.QueryRow(ctx, `SELECT `+jobColumns+jobFrom+` WHERE j.id = $1 FOR UPDATE OF j`, id))
	if err != nil {
		return j, err
	}
	return j, attachFiles(ctx, tx, []*domain.Job{&j})
}

// VerifySecret checks a guest's ticket secret in constant time.
func (s *Store) VerifySecret(ctx context.Context, jobID, secret string) error {
	var hash *string
	err := s.pool.QueryRow(ctx, `SELECT secret_hash FROM cd_jobs WHERE id = $1`, jobID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if hash == nil || !equalHash(secret, *hash) {
		return ErrBadSecret
	}
	return nil
}

// --- create / upload / edit -------------------------------------------------------

type NewFile struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Mime     string `json:"mime"`
}

type CreateJobInput struct {
	CustomerName string
	Files        []NewFile
}

type Limits struct {
	MaxFileBytes int64
	MaxJobBytes  int64
	MaxFiles     int
}

func validName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if len([]rune(name)) > 20 {
		return "", fmt.Errorf("%w: first name must be 20 characters or fewer", domain.ErrValidation)
	}
	return name, nil
}

func validateFiles(files []NewFile, lim Limits, existing int64, existingCount int) error {
	if len(files) == 0 {
		return fmt.Errorf("%w: choose at least one file", domain.ErrValidation)
	}
	if existingCount+len(files) > lim.MaxFiles {
		return fmt.Errorf("%w: up to %d files per job", domain.ErrValidation, lim.MaxFiles)
	}
	total := existing
	for _, f := range files {
		if !domain.AllowedMimes[f.Mime] {
			return fmt.Errorf("%w: we can print PDF and photos (JPG, PNG, HEIC, WebP)", domain.ErrValidation)
		}
		if f.Size <= 0 || f.Size > lim.MaxFileBytes {
			return fmt.Errorf("%w: each file must be under %d MB", domain.ErrValidation, lim.MaxFileBytes>>20)
		}
		if n := len(strings.TrimSpace(f.Filename)); n == 0 || n > 200 {
			return fmt.Errorf("%w: file name missing or too long", domain.ErrValidation)
		}
		total += f.Size
	}
	if total > lim.MaxJobBytes {
		return fmt.Errorf("%w: files must add up to under %d MB", domain.ErrValidation, lim.MaxJobBytes>>20)
	}
	return nil
}

func shopAccepting(sh domain.Shop, now time.Time) error {
	if sh.Status != "live" {
		return ErrNotFound
	}
	switch sh.OnlineState {
	case domain.ShopPaused:
		return ErrShopPaused
	case domain.ShopOffline:
		return ErrShopOffline
	}
	return nil
}

// CreateJob starts a walk-in draft (state uploading, invisible to the shop). It returns the raw
// ticket secret once; only its hash is stored.
func (s *Store) CreateJob(ctx context.Context, sh domain.Shop, in CreateJobInput, lim Limits) (domain.Job, string, error) {
	now := s.now()
	if err := shopAccepting(sh, now); err != nil {
		return domain.Job{}, "", err
	}
	name, err := validName(in.CustomerName)
	if err != nil {
		return domain.Job{}, "", err
	}
	if err := validateFiles(in.Files, lim, 0, 0); err != nil {
		return domain.Job{}, "", err
	}
	secret := NewSecret()
	jobID := newID("job")
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO cd_jobs (id, shop_id, channel, secret_hash, customer_name, settings, state, created_at, updated_at, business_day)
			VALUES ($1, $2, 'walkin', $3, $4, '{}'::jsonb, 'uploading', $5, $5, $6)`,
			jobID, sh.ID, HashSecret(secret), name, now, domain.BusinessDay(now, sh.Location())); err != nil {
			return err
		}
		return insertFiles(ctx, tx, sh.ID, jobID, in.Files)
	})
	if err != nil {
		return domain.Job{}, "", err
	}
	j, err := s.GetJob(ctx, jobID)
	return j, secret, err
}

func insertFiles(ctx context.Context, tx pgx.Tx, shopID, jobID string, files []NewFile) error {
	for _, f := range files {
		id := newID("file")
		key, err := storage.FileKey(shopID, jobID, id)
		if err != nil {
			return err
		}
		settings, _ := json.Marshal(domain.DefaultFileSettings())
		pagesStatus := domain.PagesPending
		pages := 0
		if domain.FileKind(f.Mime) == "image" {
			pagesStatus, pages = domain.PagesCounted, 1
		}
		if _, err := tx.Exec(ctx, `INSERT INTO cd_job_files (id, job_id, filename, size_bytes, mime, pages, pages_status, settings, object_key, upload_status, delete_status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'pending', 'active', clock_timestamp())`,
			id, jobID, strings.TrimSpace(f.Filename), f.Size, f.Mime, pages, pagesStatus, settings, key); err != nil {
			return err
		}
	}
	return nil
}

// AddFiles adds files to a draft or to a queued job that no counter has claimed.
func (s *Store) AddFiles(ctx context.Context, jobID string, files []NewFile, lim Limits) (domain.Job, error) {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		j, err := lockJob(ctx, tx, jobID)
		if err != nil {
			return err
		}
		if j.State != domain.JobStateUploading {
			return ErrNotEditable
		}
		var size int64
		live := 0
		for _, f := range j.Files {
			if f.DeleteStatus == domain.DeleteStatusActive {
				size += f.Size
				live++
			}
		}
		if err := validateFiles(files, lim, size, live); err != nil {
			return err
		}
		return insertFiles(ctx, tx, j.ShopID, j.ID, files)
	})
	if err != nil {
		return domain.Job{}, err
	}
	return s.GetJob(ctx, jobID)
}

// FileUploaded records a finished upload. pages comes from the client (pdf.js) for PDFs;
// 0 means the client could not count them and staff will confirm at the counter.
func (s *Store) FileUploaded(ctx context.Context, jobID, fileID string, pages int) (domain.JobFile, error) {
	var f domain.JobFile
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		j, err := lockJob(ctx, tx, jobID)
		if err != nil {
			return err
		}
		if j.State != domain.JobStateUploading {
			return ErrNotEditable
		}
		found := false
		for _, x := range j.Files {
			if x.ID == fileID && x.DeleteStatus == domain.DeleteStatusActive {
				f, found = x, true
			}
		}
		if !found {
			return ErrNotFound
		}
		status := domain.PagesCounted
		if domain.FileKind(f.Mime) == "image" {
			pages = 1
		} else if pages <= 0 || pages > 2000 {
			pages, status = 0, domain.PagesUnknown
		}
		sh, err := s.GetShopByID(ctx, j.ShopID)
		if err != nil {
			return err
		}
		now := s.now()
		expiry := sh.ClosingTime(now)
		if cap := now.Add(24 * time.Hour); expiry.After(cap) || !expiry.After(now) {
			expiry = cap
		}
		_, err = tx.Exec(ctx, `UPDATE cd_job_files SET upload_status = 'uploaded', pages = $3, pages_status = $4, delete_after = $5
			WHERE id = $2 AND job_id = $1`, jobID, fileID, pages, status, expiry)
		f.Pages, f.PagesStatus, f.UploadStatus = pages, status, domain.UploadStatusUploaded
		return err
	})
	return f, err
}

// RemoveFile drops a file from a draft; the deletion worker removes the object.
func (s *Store) RemoveFile(ctx context.Context, jobID, fileID string) (domain.Job, error) {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		j, err := lockJob(ctx, tx, jobID)
		if err != nil {
			return err
		}
		if j.State != domain.JobStateUploading {
			return ErrNotEditable
		}
		// Removed files vanish from the job at once (nobody can see or open them) and are erased on the next worker run.
		tag, err := tx.Exec(ctx, `UPDATE cd_job_files SET delete_status = 'pending', delete_after = $3, removed_at = $3
			WHERE id = $2 AND job_id = $1 AND delete_status = 'active' AND removed_at IS NULL`, jobID, fileID, s.now())
		if err == nil && tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return err
	})
	if err != nil {
		return domain.Job{}, err
	}
	return s.GetJob(ctx, jobID)
}

type FileSettingsUpdate struct {
	FileID   string              `json:"fileId"`
	Settings domain.FileSettings `json:"settings"`
}

// UpdateJob changes the name and per-file settings while the customer can still edit.
func (s *Store) UpdateJob(ctx context.Context, jobID string, name *string, files []FileSettingsUpdate) (domain.Job, error) {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		j, err := lockJob(ctx, tx, jobID)
		if err != nil {
			return err
		}
		if j.State != domain.JobStateUploading && !(j.State == domain.JobStateQueued && j.ClaimedAt == nil) {
			return ErrNotEditable
		}
		sh, err := s.GetShopByID(ctx, j.ShopID)
		if err != nil {
			return err
		}
		if name != nil {
			n, err := validName(*name)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE cd_jobs SET customer_name = $2, updated_at = $3 WHERE id = $1`, jobID, n, s.now()); err != nil {
				return err
			}
		}
		for _, u := range files {
			var target *domain.JobFile
			for i := range j.Files {
				if j.Files[i].ID == u.FileID && j.Files[i].DeleteStatus == domain.DeleteStatusActive {
					target = &j.Files[i]
				}
			}
			if target == nil {
				return ErrNotFound
			}
			st := domain.NormaliseSettings(u.Settings, domain.FileKind(target.Mime))
			if _, err := domain.ComputeQuote(sh.Prices, []domain.QuoteFile{{ID: target.ID, Kind: domain.FileKind(target.Mime), Pages: target.Pages, Settings: st}}); err != nil {
				return err
			}
			b, _ := json.Marshal(st)
			if _, err := tx.Exec(ctx, `UPDATE cd_job_files SET settings = $3 WHERE id = $2 AND job_id = $1`, jobID, u.FileID, b); err != nil {
				return err
			}
			target.Settings = st
		}
		if j.State == domain.JobStateQueued {
			q, err := QuoteFor(sh, j)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `UPDATE cd_jobs SET price_total_paise = $2, pages_total = $3, pages_to_confirm = $4, updated_at = $5 WHERE id = $1`,
				jobID, q.TotalPaise, q.PagesTotal, q.PagesToConfirm, s.now())
			return err
		}
		return nil
	})
	if err != nil {
		return domain.Job{}, err
	}
	return s.GetJob(ctx, jobID)
}

// QuoteFor prices the job's active files with the shop's current price list.
func QuoteFor(sh domain.Shop, j domain.Job) (domain.Quote, error) {
	var files []domain.QuoteFile
	for _, f := range j.Files {
		if f.DeletedAt != nil || (j.State == domain.JobStateUploading && f.DeleteStatus != domain.DeleteStatusActive) {
			continue
		}
		files = append(files, domain.QuoteFile{ID: f.ID, Kind: domain.FileKind(f.Mime), Pages: f.Pages, Settings: f.Settings})
	}
	return domain.ComputeQuote(sh.Prices, files)
}

// --- submit and transitions ---------------------------------------------------------

// Submit moves a draft into the shop's queue: checks uploads and price, assigns a lane and
// today's token, and estimates the ready-by time.
func (s *Store) Submit(ctx context.Context, jobID, priceVersion, name string) (domain.Job, domain.Quote, error) {
	var quote domain.Quote
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		j, err := lockJob(ctx, tx, jobID)
		if err != nil {
			return err
		}
		sh, err := s.GetShopByID(ctx, j.ShopID)
		if err != nil {
			return err
		}
		now := s.now()
		if err := shopAccepting(sh, now); err != nil {
			return err
		}
		live := 0
		anyColour := false
		for _, f := range j.Files {
			if f.DeleteStatus != domain.DeleteStatusActive {
				continue
			}
			live++
			if f.UploadStatus != domain.UploadStatusUploaded {
				return ErrUploadsIncomplete
			}
			anyColour = anyColour || f.Settings.Colour
		}
		if live == 0 {
			return ErrUploadsIncomplete
		}
		active := j
		active.Files = nil
		for _, f := range j.Files {
			if f.DeleteStatus == domain.DeleteStatusActive {
				active.Files = append(active.Files, f)
			}
		}
		quote, err = QuoteFor(sh, active)
		if err != nil {
			return err
		}
		if priceVersion != "" && priceVersion != quote.PriceVersion {
			return ErrPriceChanged
		}
		fx, err := domain.Transition(&j, domain.ActionSubmit, domain.Actor{Type: domain.ActorGuest}, "", now, s.policy)
		if err != nil {
			return err
		}
		if fx.IssueToken {
			lane, ok := sh.LaneFor(anyColour)
			if !ok {
				return fmt.Errorf("shop has no lanes")
			}
			day := domain.BusinessDay(now, sh.Location())
			var n int
			if err := tx.QueryRow(ctx, `INSERT INTO cd_token_counters_v2 (shop_id, lane_letter, business_day, last_no) VALUES ($1, $2, $3, 1)
				ON CONFLICT (shop_id, lane_letter, business_day) DO UPDATE SET last_no = cd_token_counters_v2.last_no + 1
				RETURNING last_no`, sh.ID, lane.Letter, day).Scan(&n); err != nil {
				return err
			}
			j.LaneID, j.Lane, j.Token, j.BusinessDay = lane.ID, lane.Letter, domain.FormatToken(lane.Letter, n), day
		}
		wait, err := waitFor(ctx, tx, sh.ID, now)
		if err != nil {
			return err
		}
		own := 3 + float64(quote.PagesTotal)/20
		readyBy := now.Add(time.Duration(float64(wait.HighMinutes)+own) * time.Minute)
		if name != "" {
			if n, err := validName(name); err == nil {
				j.CustomerName = n
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE cd_jobs SET state = 'queued', queued_at = $2, lane_id = $3, token = $4, business_day = $5,
			price_total_paise = $6, pages_total = $7, pages_to_confirm = $8, ready_by = $9, customer_name = $10, updated_at = $2 WHERE id = $1`,
			j.ID, now, j.LaneID, j.Token, j.BusinessDay, quote.TotalPaise, quote.PagesTotal, quote.PagesToConfirm, readyBy, j.CustomerName); err != nil {
			return err
		}
		// While the job is in line, printing or ready its files are kept (no deletion at closing time);
		// they are scheduled for deletion when it is collected, cancelled or closed as not collected.
		if _, err := tx.Exec(ctx, `UPDATE cd_job_files SET delete_after = NULL WHERE job_id = $1 AND delete_status = 'active'`, j.ID); err != nil {
			return err
		}
		return logEvent(ctx, tx, j, domain.JobStateUploading, domain.ActionSubmit, domain.Actor{Type: domain.ActorGuest}, "")
	})
	if err != nil {
		return domain.Job{}, domain.Quote{}, err
	}
	j, err := s.GetJob(ctx, jobID)
	return j, quote, err
}

func logEvent(ctx context.Context, tx pgx.Tx, j domain.Job, from domain.JobState, a domain.Action, actor domain.Actor, reason string) error {
	_, err := tx.Exec(ctx, `INSERT INTO cd_job_events (job_id, shop_id, from_state, to_state, action, actor_type, actor_name, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, j.ID, j.ShopID, string(from), string(j.State), string(a), string(actor.Type), actor.Name, reason)
	return err
}

type ActInput struct {
	Action domain.Action
	Actor  domain.Actor
	ShopID string // required for staff: the job must belong to this shop
	Reason string
	Paid   string // "", cash, upi — recorded on collected (reports only)
}

// Act applies an action to a job atomically.
func (s *Store) Act(ctx context.Context, jobID string, in ActInput) (domain.Job, error) {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		j, err := lockJob(ctx, tx, jobID)
		if err != nil {
			return err
		}
		if in.Actor.Type == domain.ActorStaff && j.ShopID != in.ShopID {
			return ErrNotFound
		}
		if in.Actor.Type == domain.ActorStaff && j.State == domain.JobStateUploading {
			return ErrNotFound
		}
		return s.apply(ctx, tx, &j, in)
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidTransition) && in.Action == domain.ActionClaim {
			if cur, e := s.GetJob(ctx, jobID); e == nil && cur.State == domain.JobStateClaimed {
				return cur, ErrAlreadyClaimed
			}
		}
		return domain.Job{}, err
	}
	return s.GetJob(ctx, jobID)
}

func (s *Store) apply(ctx context.Context, tx pgx.Tx, j *domain.Job, in ActInput) error {
	from := j.State
	now := s.now()
	fx, err := domain.Transition(j, in.Action, in.Actor, strings.TrimSpace(in.Reason), now, s.policy)
	if err != nil {
		return err
	}
	if in.Action == domain.ActionCollected && (in.Paid == "cash" || in.Paid == "upi") {
		j.PaidMethod = in.Paid
	}
	tag, err := tx.Exec(ctx, `UPDATE cd_jobs SET state = $3, updated_at = $4, claimed_at = $5, claimed_by = $6, ready_at = $7,
		collected_at = $8, cancelled_at = $9, cancel_reason = $10, paid_method = $11 WHERE id = $1 AND state = $2`,
		j.ID, string(from), string(j.State), now, j.ClaimedAt, j.ClaimedBy, j.ReadyAt, j.CollectedAt, j.CancelledAt, j.CancelReason, j.PaidMethod)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInvalidTransition
	}
	if fx.ScheduleDelete != nil {
		if _, err := tx.Exec(ctx, `UPDATE cd_job_files SET delete_status = 'pending',
			delete_after = CASE WHEN delete_after IS NULL OR delete_after > $2 THEN $2 ELSE delete_after END
			WHERE job_id = $1 AND deleted_at IS NULL`, j.ID, *fx.ScheduleDelete); err != nil {
			return err
		}
	}
	if fx.ClearDelete {
		if _, err := tx.Exec(ctx, `UPDATE cd_job_files SET delete_status = 'active', delete_after = NULL
			WHERE job_id = $1 AND deleted_at IS NULL AND removed_at IS NULL`, j.ID); err != nil {
			return err
		}
	}
	return logEvent(ctx, tx, *j, from, in.Action, in.Actor, in.Reason)
}

// ClaimNext claims the oldest queued job in a lane (or across all lanes when lane is "").
func (s *Store) ClaimNext(ctx context.Context, shopID, lane string, actor domain.Actor) (domain.Job, error) {
	var id string
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT j.id FROM cd_jobs j LEFT JOIN cd_lanes l ON l.id = j.lane_id
			WHERE j.shop_id = $1 AND j.state = 'queued' AND ($2 = '' OR l.letter = $2)
			ORDER BY j.queued_at, j.id LIMIT 1 FOR UPDATE OF j SKIP LOCKED`, shopID, lane).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLaneEmpty
		}
		if err != nil {
			return err
		}
		j, err := lockJob(ctx, tx, id)
		if err != nil {
			return err
		}
		return s.apply(ctx, tx, &j, ActInput{Action: domain.ActionClaim, Actor: actor, ShopID: shopID})
	})
	if err != nil {
		return domain.Job{}, err
	}
	return s.GetJob(ctx, id)
}

// --- queue, lookup, wait ----------------------------------------------------------------

type QueueSnapshot struct {
	Shop domain.Shop  `json:"shop"`
	Jobs []domain.Job `json:"jobs"`
	// Finished jobs whose files the shop downloaded and hasn't yet confirmed deleting (last 30 days).
	CopiesToDelete []domain.Job        `json:"copiesToDelete"`
	TodayCount     int                 `json:"todayCount"`
	Wait           domain.WaitEstimate `json:"wait"`
	UndoWindow     int                 `json:"undoWindowSeconds"`
	ServerTime     time.Time           `json:"serverTime"`
}

// Queue returns today's live board: queued, claimed, ready, and jobs collected within the undo window.
func (s *Store) Queue(ctx context.Context, shopID string) (QueueSnapshot, error) {
	sh, err := s.GetShopByID(ctx, shopID)
	if err != nil {
		return QueueSnapshot{}, err
	}
	now := s.now()
	rows, err := s.pool.Query(ctx, `SELECT `+jobColumns+jobFrom+`
		WHERE j.shop_id = $1 AND (j.state IN ('queued','claimed','ready')
		   OR (j.state = 'collected' AND j.collected_at > $2))
		ORDER BY j.queued_at, j.id LIMIT 500`, shopID, now.Add(-s.policy.UndoWindow))
	if err != nil {
		return QueueSnapshot{}, err
	}
	jobs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Job, error) { return scanJob(r) })
	if err != nil {
		return QueueSnapshot{}, err
	}
	ptrs := make([]*domain.Job, len(jobs))
	for i := range jobs {
		ptrs[i] = &jobs[i]
	}
	if err := attachFiles(ctx, s.pool, ptrs); err != nil {
		return QueueSnapshot{}, err
	}
	var today int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM cd_jobs WHERE shop_id = $1 AND business_day = $2 AND state <> 'uploading'`,
		shopID, domain.BusinessDay(now, sh.Location())).Scan(&today); err != nil {
		return QueueSnapshot{}, err
	}
	wait, err := waitFor(ctx, s.pool, shopID, now)
	if err != nil {
		return QueueSnapshot{}, err
	}
	copies, err := s.copiesToDelete(ctx, shopID, now)
	if err != nil {
		return QueueSnapshot{}, err
	}
	return QueueSnapshot{Shop: sh, Jobs: jobs, CopiesToDelete: copies, TodayCount: today, Wait: wait, UndoWindow: int(s.policy.UndoWindow.Seconds()), ServerTime: now}, nil
}

type rowQuerier interface {
	querier
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// waitFor estimates the wait for a new job: queued + printing jobs ahead, shared by active counters.
func waitFor(ctx context.Context, q rowQuerier, shopID string, now time.Time) (domain.WaitEstimate, error) {
	rows, err := q.Query(ctx, `SELECT pages_total FROM cd_jobs WHERE shop_id = $1 AND state IN ('queued','claimed')`, shopID)
	if err != nil {
		return domain.WaitEstimate{}, err
	}
	pages, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		return domain.WaitEstimate{}, err
	}
	var counters int
	if err := q.QueryRow(ctx, `SELECT count(DISTINCT claimed_by) FROM cd_jobs WHERE shop_id = $1 AND claimed_at > $2 AND claimed_by <> ''`,
		shopID, now.Add(-30*time.Minute)).Scan(&counters); err != nil {
		return domain.WaitEstimate{}, err
	}
	return domain.EstimateWait(pages, 3, counters), nil
}

func (s *Store) Wait(ctx context.Context, shopID string) (domain.WaitEstimate, error) {
	return waitFor(ctx, s.pool, shopID, s.now())
}

// Position counts queued or printing jobs in the same lane that are ahead of this one.
func (s *Store) Position(ctx context.Context, j domain.Job) (int, error) {
	if j.State != domain.JobStateQueued || j.QueuedAt == nil {
		return 0, nil
	}
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM cd_jobs WHERE shop_id = $1 AND lane_id = $2
		AND (state = 'claimed' OR (state = 'queued' AND (queued_at, id) < ($3, $4)))`, j.ShopID, j.LaneID, *j.QueuedAt, j.ID).Scan(&n)
	return n, err
}

// Lookup finds today's open jobs by token (A07, a-7), or first-name prefix.
func (s *Store) Lookup(ctx context.Context, shopID, q string) ([]domain.Job, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []domain.Job{}, nil
	}
	token := ""
	up := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(q, "-", ""), " ", ""))
	if len(up) >= 2 && up[0] >= 'A' && up[0] <= 'Z' {
		var n int
		if _, err := fmt.Sscanf(up[1:], "%d", &n); err == nil && n > 0 {
			token = domain.FormatToken(up[:1], n)
		}
	}
	rows, err := s.pool.Query(ctx, `SELECT `+jobColumns+jobFrom+`
		WHERE j.shop_id = $1 AND j.state IN ('queued','claimed','ready','collected')
		  AND j.created_at > $4
		  AND (j.token = $2::text OR (length($3::text) >= 2 AND j.customer_name ILIKE $3::text || '%'))
		ORDER BY j.queued_at DESC LIMIT 10`, shopID, token, q, s.now().Add(-24*time.Hour))
	if err != nil {
		return nil, err
	}
	jobs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Job, error) { return scanJob(r) })
	if err != nil {
		return nil, err
	}
	ptrs := make([]*domain.Job, len(jobs))
	for i := range jobs {
		ptrs[i] = &jobs[i]
	}
	return jobs, attachFiles(ctx, s.pool, ptrs)
}

// FileForShop returns the object key of a live file belonging to a shop's job (for signed staff URLs).
func (s *Store) FileForShop(ctx context.Context, shopID, jobID, fileID string) (domain.JobFile, error) {
	j, err := s.GetJob(ctx, jobID)
	if err != nil {
		return domain.JobFile{}, err
	}
	if j.ShopID != shopID || j.State == domain.JobStateUploading || j.State == domain.JobStateCancelled {
		return domain.JobFile{}, ErrNotFound
	}
	for _, f := range j.Files {
		if f.ID == fileID && f.DeletedAt == nil && f.ObjectKey != "" && f.UploadStatus == domain.UploadStatusUploaded {
			return f, nil
		}
	}
	return domain.JobFile{}, ErrNotFound
}

// FileForGuest returns a draft file's key so the API can verify an upload.
func (s *Store) FileForGuest(ctx context.Context, jobID, fileID string) (domain.Job, domain.JobFile, error) {
	j, err := s.GetJob(ctx, jobID)
	if err != nil {
		return j, domain.JobFile{}, err
	}
	for _, f := range j.Files {
		if f.ID == fileID && f.DeleteStatus == domain.DeleteStatusActive {
			return j, f, nil
		}
	}
	return j, domain.JobFile{}, ErrNotFound
}

// --- what the shop did with the files ------------------------------------------------------

// RecordFileAccess notes that staff opened a file to print it or downloaded it to their device.
func (s *Store) RecordFileAccess(ctx context.Context, jobID, fileID, staffName string, download bool) (domain.Job, error) {
	now := s.now()
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		j, err := lockJob(ctx, tx, jobID)
		if err != nil {
			return err
		}
		sql := `UPDATE cd_job_files SET print_opens = print_opens + 1, printed_at = COALESCE(printed_at, $3) WHERE id = $2 AND job_id = $1`
		action := domain.Action("file_print")
		if download {
			sql = `UPDATE cd_job_files SET downloads = downloads + 1, downloaded_at = COALESCE(downloaded_at, $3),
				downloaded_by = CASE WHEN downloaded_by = '' THEN $4 ELSE downloaded_by END WHERE id = $2 AND job_id = $1`
			action = "file_download"
			// A new download means there is (again) a copy the shop must delete.
			if _, err := tx.Exec(ctx, `UPDATE cd_jobs SET copies_deleted_at = NULL, copies_deleted_by = '' WHERE id = $1`, jobID); err != nil {
				return err
			}
		}
		args := []any{jobID, fileID, now}
		if download {
			args = append(args, staffName)
		}
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return err
		}
		return logEvent(ctx, tx, j, j.State, action, domain.Actor{Type: domain.ActorStaff, Name: staffName}, fileID)
	})
	if err != nil {
		return domain.Job{}, err
	}
	return s.GetJob(ctx, jobID)
}

// RequestCopiesDeletion is the customer asking, after pickup, for their files to be deleted.
// Counter Drop's own copy is deleted right away. If the shop never downloaded anything there is
// nothing more to do; otherwise the shop is asked to delete its downloaded copies and confirm.
func (s *Store) RequestCopiesDeletion(ctx context.Context, jobID string) (domain.Job, error) {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		j, err := lockJob(ctx, tx, jobID)
		if err != nil {
			return err
		}
		if j.State != domain.JobStateCollected {
			return domain.ErrInvalidTransition
		}
		if j.CopiesDeleteRequestedAt != nil {
			return nil
		}
		now := s.now()
		if _, err := tx.Exec(ctx, `UPDATE cd_jobs SET copies_delete_requested_at = $2, updated_at = $2 WHERE id = $1`, jobID, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE cd_job_files SET delete_status = 'pending', delete_after = $2
			WHERE job_id = $1 AND deleted_at IS NULL AND (delete_after IS NULL OR delete_after > $2)`, jobID, now); err != nil {
			return err
		}
		if !j.Downloaded() && j.CopiesDeletedAt == nil {
			if _, err := tx.Exec(ctx, `UPDATE cd_jobs SET copies_deleted_at = $2, copies_deleted_by = 'no-downloads' WHERE id = $1`, jobID, now); err != nil {
				return err
			}
		}
		return logEvent(ctx, tx, j, j.State, "delete_requested", domain.Actor{Type: domain.ActorGuest, Name: "customer"}, "")
	})
	if err != nil {
		return domain.Job{}, err
	}
	return s.GetJob(ctx, jobID)
}

// MarkCopiesDeleted is the shop confirming it deleted the copies it downloaded.
func (s *Store) MarkCopiesDeleted(ctx context.Context, shopID, jobID, staffName string) (domain.Job, error) {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		j, err := lockJob(ctx, tx, jobID)
		if err != nil {
			return err
		}
		if j.ShopID != shopID || j.State == domain.JobStateUploading {
			return ErrNotFound
		}
		if j.State != domain.JobStateCollected && j.State != domain.JobStateCancelled {
			return fmt.Errorf("%w: finish the job first (collected or cancelled)", domain.ErrValidation)
		}
		if !j.Downloaded() {
			return fmt.Errorf("%w: nothing from this job was downloaded", domain.ErrValidation)
		}
		if j.CopiesDeletedAt != nil {
			return nil
		}
		now := s.now()
		if _, err := tx.Exec(ctx, `UPDATE cd_jobs SET copies_deleted_at = $2, copies_deleted_by = $3, updated_at = $2 WHERE id = $1`,
			jobID, now, staffName); err != nil {
			return err
		}
		return logEvent(ctx, tx, j, j.State, "copies_deleted", domain.Actor{Type: domain.ActorStaff, Name: staffName}, "")
	})
	if err != nil {
		return domain.Job{}, err
	}
	return s.GetJob(ctx, jobID)
}

func (s *Store) copiesToDelete(ctx context.Context, shopID string, now time.Time) ([]domain.Job, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+jobColumns+jobFrom+`
		WHERE j.shop_id = $1 AND j.state IN ('collected', 'cancelled') AND j.copies_deleted_at IS NULL
		  AND j.updated_at > $2
		  AND EXISTS (SELECT 1 FROM cd_job_files f WHERE f.job_id = j.id AND f.downloads > 0)
		ORDER BY j.copies_delete_requested_at DESC NULLS LAST, j.collected_at DESC NULLS LAST LIMIT 100`, shopID, now.Add(-30*24*time.Hour))
	if err != nil {
		return nil, err
	}
	jobs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Job, error) { return scanJob(r) })
	if err != nil {
		return nil, err
	}
	ptrs := make([]*domain.Job, len(jobs))
	for i := range jobs {
		ptrs[i] = &jobs[i]
	}
	return jobs, attachFiles(ctx, s.pool, ptrs)
}
