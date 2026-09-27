# 00 — Common context for every step

Every step prompt tells the agent to read this file first. Keep it current when a decision changes.

## Product in one paragraph

Counter Drop replaces "send it on my WhatsApp" at Indian print and xerox shops. A customer scans the shop's QR, uploads files from the phone browser, sees the price and wait, gets a token (e.g. `A-07`) and collects when it's ready; files are deleted automatically after pickup. Later (R1b), customers use **Print nearby** in the same PWA to find an open shop, upload, prepay by UPI and collect with a pickup code. Shops run every job from one live queue on a PC or phone.

## Decisions already made

- **Release plan:** R1a = walk-in pilot (10 shops), then R1b = Print nearby + payments. Build R1a first.
- **One React PWA** for customer, shop dashboard, TV and admin. No native app. No Flutter.
- **A scan is upload-only.** Scanning a shop QR (camera or in-app Scan) opens that shop's upload screen. No payment, no nearby, no history there. Walk-in customers pay the shop at the counter.
- **UPI prepay exists only in Print nearby** (R1b), via Razorpay, with Route settlement to the shop.
- **Files live at most 24 hours**; deleted 10 minutes after collection (undo window), at shop closing, or immediately on cancel.
- Counter Drop never stores file content in the database, never shows files to admins and never holds funds.
- **Design:** light theme by default (Station Utility palette), optional dark theme for shop screens; follow the content rules in `design/DESIGN.md` §8 (no names on TV, no customer location, no compliance badges, no racks/printer telemetry until R2).
- **Spoken announcements / PA audio are a later release** (backlog 37h). R1a has a simple chime only.

## Source documents

- `docs/BRD.md` — business requirements (IDs: BO, UC-C, UC-S, FR, BR, EX, UX, NFR, CT).
- `docs/FSD.md` — functional spec (IDs: SCR, FS, ST, T, API, VAL, MSG, AT). **This is the spec to implement.**
- `contracts/openapi.yaml` — API contract; keep it in sync with code.
- `design/DESIGN.md` — design system (colours, type, components, content rules). `design/screens/` holds corrected reference screens; `design/stitch-export/` is the raw Stitch export — don't build from it.

## Repository layout

```text
counter-drop/
├── go.work                 # Go workspace: api, agent, pkg/cdclient
├── api/                    # Go HTTP API (module counter-drop/api, Go 1.27)
│   ├── cmd/api/main.go
│   ├── internal/config     # env loading (CD_* variables)
│   ├── internal/domain     # job state machine, shop, pricing, tokens (pure Go, no I/O)
│   ├── internal/httpapi    # handlers, middleware, JSON envelope
│   ├── internal/store      # Postgres (pgx v5) + in-memory store
│   ├── internal/storage    # S3-compatible presigned URLs (aws-sdk-go-v2)
│   ├── internal/realtime   # WebSocket hub (step 13)
│   ├── internal/tasks      # workers: deletion, timers, reconciliation
│   └── migrations/         # plain SQL, applied in order by ApplyMigrations
├── web/                    # React + Vite + TypeScript PWA (step 15)
├── agent/                  # Windows print agent (R2)
├── contracts/openapi.yaml
├── deploy/                 # docker-compose (Postgres), .env.example
└── docs/                   # BRD.md, FSD.md, ARCHITECTURE.md, ROADMAP.md, build-plan/
```

## Current code facts (as of 27 Sep 2026)

- Router: `api/internal/httpapi/router.go`, Go 1.22+ `http.ServeMux` patterns; envelope `{success, data, error}`.
- Routes today: `GET /health`, `GET /api/v1/cd/shops/{slug}`, `POST /api/v1/cd/shops/{slug}/jobs`, `GET /api/v1/cd/jobs/{id}?secret=`, `POST /api/v1/cd/jobs/{id}/submit?secret=`, `GET /api/v1/cd/shop/queue?shop=`, `POST /api/v1/cd/shop/jobs/{id}/{action}`.
- Job states in `domain/job.go`: `new, claimed, ready, collected, cancelled`. Actions: claim, ready, collected, cancel, release.
- Store interface methods have no `context.Context`; Postgres store uses pgx v5 transactions and `FOR UPDATE`.
- Migrations 0001–0003 exist: `cd_shops`, `cd_token_counters`, `cd_jobs`, `cd_job_files` (+ `object_key`, `upload_status`, `delete_status`, `delete_after`, `deleted_at`).
- Collected sets `delete_after` = +15 min (spec says 10). No deletion worker yet. No auth on shop routes.
- Local Postgres: `docker compose -f deploy/docker-compose.yml up -d postgres` on port 55433. MinIO image pull failed on this machine; use Cloudflare R2 (step 02).
- Tests run from `api/`: `cd api && go test ./...` (root is a `go.work` workspace, not a module).
- Repo prep done 27 Sep 2026: `.gitignore`, `.gitattributes`, `.editorconfig` added; Dockerfile now ships `migrations/`; `docs/CODE-STATUS.md` lists known issues (numbered) and which step fixes each — check it at the start of every step.

## Engineering rules for every step

1. **Read before writing.** Read the FSD sections named in the step and the files you will change. Propose a short plan (files, functions, tests) and wait for approval before large changes.
2. **Stay in scope.** Do only what the step says. Note anything else as a TODO in the PR description, not in code.
3. **Domain stays pure.** `internal/domain` has no database, HTTP or time.Now() calls; pass `now time.Time` in.
4. **Every state change is one DB transaction** using `UPDATE … WHERE id = $1 AND state = $from RETURNING …`; zero rows → `ErrInvalidTransition` (HTTP 409).
5. **Money is integer paise** (`int64`). Times are `time.Time` in UTC in Go and `timestamptz` in SQL; convert to IST only for display and business-day math.
6. **IDs are ULIDs** (`github.com/oklog/ulid/v2`) as text, except existing rows.
7. **Errors:** typed sentinel errors in the store/domain, mapped to HTTP in one place (`httpapi/errors.go`) with stable `code` strings from FSD §16.
8. **Logging:** `log/slog` JSON; include `request_id`, `shop_id`, `job_id`; never log secrets, OTPs, PINs, phone numbers, customer names or file names.
9. **Tests:** table-driven unit tests for domain rules; integration tests against a real Postgres (use `CD_TEST_DATABASE_URL`, skip if unset) for store and HTTP.
10. **API contract:** update `contracts/openapi.yaml` in the same PR as any endpoint change.
11. **Backwards compatibility:** keep existing routes working until the web app is switched; mark old ones `deprecated: true` in OpenAPI.
12. **Config:** every tunable in FSD marked *(config)* is a `CD_*` env var with a default in `internal/config`.
13. **No new infrastructure** (queues, Redis, Kafka) unless the step says so. Postgres + goroutines are enough for year one.
14. **Frontend:** TypeScript strict, no `any`; server state with TanStack Query; all strings through i18n keys; mobile-first at 360 px; 48 px touch targets.

## Commands

```bash
# Postgres
docker compose -f deploy/docker-compose.yml up -d postgres

# API (from api/)
go run ./cmd/api
go test ./...

# Web (from web/, after step 15)
pnpm install
pnpm dev
pnpm test
pnpm build

# Everything (after step 01)
make dev
make test
make lint
```

## Glossary

Token (`A-07`, walk-in) · Pickup code (4 digits + QR, remote) · Lane (sub-queue, e.g. B/W or colour) · Claim (a counter takes a job) · Rush mode (simplified board) · Pause intake (stop new jobs) · Business day (shop opening to next opening, IST) · Ticket secret (128-bit value giving guest access to one job).
