package ddbstore

import (
	"sort"
	"strings"
	"time"

	"counter-drop/api/internal/domain"
)

// --- shops -----------------------------------------------------------------------------------

type shopRec struct {
	PK, SK     string
	GSI3PK     string `dynamodbav:",omitempty"`
	GSI3SK     string `dynamodbav:",omitempty"`
	Type       string
	Ver        int64
	Shop       domain.Shop
	OwnerCount int
	CreatedAt  time.Time
}

func newShopRec(sh domain.Shop, now time.Time) shopRec {
	return shopRec{PK: "S#" + sh.ID, SK: "PROFILE", GSI3PK: "SHOPS", GSI3SK: sh.ID, Type: "shop", Shop: sh, CreatedAt: now}
}

type slugRec struct {
	PK, SK string
	Type   string
	ShopID string
}

// --- staff -----------------------------------------------------------------------------------

type staffRec struct {
	PK, SK         string
	Type           string
	ID             string
	ShopID         string
	Name           string
	Role           string
	PinHash        string
	PinSetAt       *time.Time
	FailedAttempts int
	LockedUntil    *time.Time
	Active         bool
	CreatedAt      time.Time
	RemovedAt      *time.Time
	// Bumping an epoch invalidates every session (SessEpoch) or unused setup link (LinkEpoch) issued before.
	SessEpoch int64
	LinkEpoch int64
}

func staffSK(id string) string { return "STAFF#" + id }

func nameSK(name string) string { return "NAME#" + strings.ToLower(strings.TrimSpace(name)) }

type nameRec struct {
	PK, SK  string
	Type    string
	StaffID string
}

type sessionRec struct {
	PK, SK    string
	Type      string
	StaffID   string
	ShopID    string
	Epoch     int64
	ExpiresAt time.Time
	RevokedAt *time.Time
	TTL       int64 `dynamodbav:"ttl"`
}

type linkRec struct {
	PK, SK    string
	Type      string
	StaffID   string
	ShopID    string
	Purpose   string
	CreatedBy string
	Epoch     int64
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
	Used      bool
	TTL       int64 `dynamodbav:"ttl"`
}

type counterRec struct {
	LastNo int
}

// --- jobs ------------------------------------------------------------------------------------

// fileRec is a job file plus the bookkeeping the worker needs (not part of the API shape).
type fileRec struct {
	F               domain.JobFile
	Seq             int
	CreatedAt       time.Time
	RemovedAt       *time.Time
	DeleteAttempts  int
	NextAttemptAt   *time.Time
	LastDeleteError string
}

type jobRec struct {
	PK, SK string
	GSI1PK string `dynamodbav:",omitempty"`
	GSI1SK string `dynamodbav:",omitempty"`
	GSI2PK string `dynamodbav:",omitempty"`
	GSI2SK string `dynamodbav:",omitempty"`
	GSI3PK string `dynamodbav:",omitempty"`
	GSI3SK string `dynamodbav:",omitempty"`
	GSI4PK string `dynamodbav:",omitempty"`
	GSI4SK string `dynamodbav:",omitempty"`
	Type   string
	Ver    int64

	SecretHash string
	Job        domain.Job // Files is always empty here; they live in Files below
	Files      []fileRec
}

func jobPK(id string) string { return "J#" + id }

// domainJob returns the job as the API sees it: removed files hidden, names dropped after deletion.
func (r *jobRec) domainJob() domain.Job {
	j := r.Job
	j.Files = []domain.JobFile{}
	files := append([]fileRec(nil), r.Files...)
	sort.SliceStable(files, func(a, b int) bool { return files[a].Seq < files[b].Seq })
	for _, f := range files {
		if f.RemovedAt != nil {
			continue
		}
		jf := f.F
		if jf.DeletedAt != nil {
			jf.Filename = "" // names are dropped once files are gone (FS-12.3)
			jf.Settings.Note = ""
		}
		jf.Settings = domain.NormaliseSettings(jf.Settings, domain.FileKind(jf.Mime))
		j.Files = append(j.Files, jf)
	}
	return j
}

// allFiles returns every file including removed ones (for pricing rules that need them).
func (r *jobRec) file(id string) *fileRec {
	for i := range r.Files {
		if r.Files[i].F.ID == id {
			return &r.Files[i]
		}
	}
	return nil
}

// reindex sets the sparse GSI keys from the job's current state. It is the single place that
// decides which lists a job appears on.
func (r *jobRec) reindex() {
	j := r.Job
	r.GSI1PK, r.GSI1SK, r.GSI2PK, r.GSI2SK, r.GSI3PK, r.GSI3SK, r.GSI4PK, r.GSI4SK = "", "", "", "", "", "", "", ""

	if j.QueuedAt != nil {
		sk := ts(*j.QueuedAt) + "#" + j.ID
		switch j.State {
		case domain.JobStateQueued, domain.JobStateClaimed, domain.JobStateReady:
			r.GSI1PK, r.GSI1SK = "S#"+j.ShopID+"#OPEN", sk
		}
		if j.BusinessDay != "" {
			r.GSI2PK, r.GSI2SK = "S#"+j.ShopID+"#D#"+j.BusinessDay, sk
		}
	}

	switch {
	case j.State == domain.JobStateUploading:
		r.GSI3PK, r.GSI3SK = "DRAFT", ts(j.UpdatedAt)+"#"+j.ID
	case j.State.IsTerminal() && j.CopiesDeletedAt == nil && r.downloaded():
		r.GSI3PK, r.GSI3SK = "COPIES#"+j.ShopID, ts(j.UpdatedAt)+"#"+j.ID
	}

	if due, ok := r.nextDue(); ok {
		r.GSI4PK, r.GSI4SK = "DUE", ts(due)+"#"+j.ID
	}
}

func (r *jobRec) downloaded() bool {
	for _, f := range r.Files {
		if f.F.Downloads > 0 {
			return true
		}
	}
	return false
}

// nextDue is the earliest time the worker should look at this job: the soonest file whose deletion
// is due (or whose retry is due).
func (r *jobRec) nextDue() (time.Time, bool) {
	var best time.Time
	found := false
	for _, f := range r.Files {
		if f.F.DeletedAt != nil || f.F.DeleteAfter == nil {
			continue
		}
		at := *f.F.DeleteAfter
		if f.NextAttemptAt != nil && f.NextAttemptAt.After(at) {
			at = *f.NextAttemptAt
		}
		if !found || at.Before(best) {
			best, found = at, true
		}
	}
	return best, found
}

type eventRec struct {
	PK, SK    string
	Type      string
	JobID     string
	ShopID    string
	FromState string
	ToState   string
	Action    string
	ActorType string
	ActorName string
	Reason    string
	At        time.Time
	TTL       int64 `dynamodbav:"ttl"`
}

// Audit entries are kept for one year (BR-D8, proposal).
const eventRetention = 365 * 24 * time.Hour

type statRec struct {
	PK, SK  string
	Deleted int
	TTL     int64 `dynamodbav:"ttl"`
}
