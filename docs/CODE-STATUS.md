# Code status vs the plan

Snapshot: 28 Sep 2026, after the MVP build. Verified by `go vet`, `go test ./...` against Postgres 17, `npm run build` (strict TypeScript), and a browser walkthrough (phone viewport for the customer, desktop for the board) with no console errors.

**Summary:** the R1a walk-in pilot slice is built end to end. A customer scans the QR, uploads, sees the price, gets a token and a live ticket; staff sign in with a PIN, work the live board and mark collected; the deletion worker removes files 10 minutes after pickup and the customer's phone shows the deletion receipt.

## Present, by build-plan step

| Step | Status | Notes |
| --- | --- | --- |
| 01 Dev env | Partial | compose (Postgres; MinIO optional profile), `.env.example`, git files. No Makefile or CI yet |
| 02 Storage | Done (MVP) | `ObjectStore`: R2/S3 presign PUT (length-signed) / GET / head / delete, plus **local disk** with HMAC-signed links (default) |
| 03 OpenAPI | Stale | `contracts/openapi.yaml` predates the MVP; `api/README.md` has the current routes |
| 04 API foundation | Done | recover, request ID, path-only logging, CORS, `{data}` / `{error}` envelope, strict JSON decode, context everywhere |
| 05 Schema v2 | Done (MVP) | `0004_mvp_walkin.sql`: lanes, staff, sessions, daily token counters, job events, file settings and deletion retries; upgrades old rows |
| 06 State machine | Done (MVP) | uploading → queued → claimed → ready → collected, cancelled; release, undo, abandon; effects; full matrix test |
| 07 Auth | Partial | Staff PIN (pbkdf2, lockout), hashed sessions, shop scope, owner role; **one-time setup links** (people choose their own PIN), change PIN, owner staff management, weak-PIN rule. **No phone OTP / self-signup yet** (shops created with `cdadmin`) |
| 08 Pricing | Done | Paise, per side / per sheet, colour, copies, page ranges, minimum charge, rupee rounding, price version |
| 09 Tokens & wait | Done | Daily tokens per lane (A B/W, B colour), IST business day, wait range and ready-by |
| 10 Walk-in job API | Done | Create, add/remove files, upload confirm with size/type check, page count, quote, submit, ticket, cancel |
| 11 Shop API | Done (MVP) | Queue, claim-next (`SKIP LOCKED`), actions, lookup, state, settings, signed file links |
| 12 Outbox/audit | Partial | `cd_job_events` audit trail; no outbox |
| 13 Realtime | Done (MVP) | **SSE** hub (not WebSocket), polling fallback |
| 14 Deletion worker | Done | Retries with backoff, clears names, drops file names, abandons drafts, deletion-health endpoint |
| 15–22 Web PWA | Done (MVP) | Home, drop page, ticket, shop login, board, settings, QR poster; EN/HI/MR customer screens; installable PWA |
| 23+ | Not started | Admin console, R1b (Print nearby, UPI prepay), print agent, deploy pipeline |

## Verification (28 Sep 2026)

A 40-step browser audit on phone and desktop covered the customer flow (upload, page count, locked/oversize/wrong-type files, pricing incl. colour, both sides, copies and page ranges, tokens per lane, live ticket, cancel), the shop flow (PIN sign-in, lanes, claim/release/ready/collected/undo, staff cancel with reason, search, Online/Paused/Offline, owner price change, QR poster), file sharing (files downloaded by staff are byte-identical to what the customer sent; tampered or unsigned links refused; files stored under IDs, not names) and automatic deletion (files erased after the undo window, links stop working, receipt updates). All passed.

Fixed during the audit: a file the customer removed before sending stayed attached to the job, so staff could see and open it until the deletion worker ran. Migration 0006 now hides removed files everywhere at once; covered by `TestRemovedFileNeverReachesCounter`.

## File handling model (28 Sep 2026)

- **Counter Drop's copy** is deleted automatically 10 minutes after pickup (or when the job is cancelled, or right away when the customer asks). Every deletion is checked: the file must really be gone before it is marked deleted.
- **The shop chooses per file:** **Print** (opens in the browser, nothing saved) or **Download** (saved to the shop's device). Every download is recorded and the customer is told live on their ticket.
- **The customer:** can cancel and withdraw their files while the job is in line; once printing starts the job is locked to the shop. After pickup they can ask the shop to delete downloaded copies.
- **The shop confirms** with "Copies deleted"; the board keeps a to-do list of downloaded copies until then. Deleting the shop's own copies is the shop's responsibility — Counter Drop records it.
- **Receipt:** amount, payment method, each file's status (printed / downloaded by whom and when), when Counter Drop's copy was deleted, and the shop's confirmation.
- **Active jobs keep their files** (no deletion at closing time). Jobs not collected within the owner's setting (1–7 days, default 7) close as "not collected" and their files are deleted.

## Pilot readiness (28 Sep 2026, evening)

Built on top of the AWS/CI/Docker work added the same afternoon (Dockerfile, `deploy/aws/*`, CI, Makefile, `docs/LAUNCH.md`, OpenAPI):

- **PWA:** Install button (Android/desktop), iPhone "Add to Home Screen" hint, in-app **Scan shop QR** (native detector or a small decoder on iPhone; only this app's shop codes accepted), manifest id/shortcuts/screenshots, first-visit cache cut from 1.6 MB to 0.9 MB.
- **Privacy and terms pages** (`/privacy`, `/terms`), linked from every customer screen, sign-in, setup and the QR poster. Operator name and contact come from `VITE_OPERATOR_NAME` / `VITE_SUPPORT_EMAIL`. Needs a lawyer's review before a public launch.
- **Rate limits and security headers** (see `api/README.md` → Protection).
- **Deploy:** Caddy for automatic HTTPS on the VM path (`--profile https`), app bound to localhost, build-time legal contact, `CD_TRUST_PROXY`, S3 public-access block + lifecycle backstop + versioning-off guidance, "run exactly one task" on ECS, CI now runs the database tests against Postgres.

Fixed: the shop board re-fetched the queue in an endless loop (≈220 requests/minute per open board) — now 2 requests in 30 s idle; two containers starting together could crash creating the migrations table; the security policy initially blocked the embedded fonts.

Verified: all Go tests (incl. rate limits, headers, concurrent migrations); 46-step browser audit twice (0 errors, 0 policy violations); production-mode run over real HTTPS through Caddy (cdadmin shop → owner/staff setup links → in-app camera scan → upload → print/download → live updates → receipt → delete request → confirmation; rate limits; headers). Not verifiable here: the Docker image build itself (this sandbox can't reach container registries) — each of its steps was run natively instead.

## Decisions taken while building (update BRD/FSD when convenient)

| Decision | Why |
| --- | --- |
| Server-Sent Events instead of WebSocket | One-way updates are all R1a needs; no extra dependency; works through proxies. Single API instance for now (hub is in memory) |
| Page count on the phone with pdf.js; server falls back to "pages confirmed at counter" | No PDF parsing on the server in R1a |
| Local-disk storage as the default | Pilot runs on one server without an R2 account; R2 switches on with env vars |
| Postgres required; in-memory store removed | One code path to test |
| Business day = local midnight (IST) | Tokens restart at A-01 each day |
| PIN-only staff sign-in, shops created with `cdadmin` | OTP and self-signup are the next auth step |
| One-time setup links instead of handing out PINs | Whoever creates an account never knows its PIN; works without SMS; becomes the "invite staff" step of the OTP signup later |
| Audio/voice features deferred | As agreed |

## Next

1. Deploy a pilot: one small VM (API + built web via `CD_WEB_DIR`) + managed Postgres, HTTPS, `CD_STORAGE_SIGNING_KEY` set; or R2.
2. Phone OTP for owners and shop self-signup (step 07 — setup links, change PIN and staff management are done) + admin console (step 23).
3. Refresh `contracts/openapi.yaml`; add Makefile and CI (step 01).
4. R1b: Print nearby and UPI prepay.
