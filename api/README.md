# Counter Drop API

Go 1.27 service (`net/http`, data in one DynamoDB table behind the `store.Repository` interface — ADR-001) for the R1a walk-in flow: QR drop page, tickets, staff PIN sign-in, live queue, pricing, daily tokens per lane, and the deletion worker.

## Run

```bash
docker compose -f ../deploy/docker-compose.yml up -d dynamodb      # DynamoDB Local on :8000
CD_DYNAMODB_ENDPOINT=http://localhost:8000 go run ./cmd/api
```

With an endpoint set, the API creates the table on start. In dev the `demo-print` shop is seeded (Owner PIN `1234`, Kavita PIN `1111`). DynamoDB Local runs in memory, so data resets when it restarts. Files go to local disk (`.data/files`) unless the `CD_STORAGE_*` S3/R2 values are set. All settings: `../deploy/.env.example`.

| Variable | Default | |
| --- | --- | --- |
| `CD_DYNAMODB_TABLE` | `cd-main` | Table name |
| `CD_DYNAMODB_REGION` | `$AWS_REGION` or `ap-south-1` | |
| `CD_DYNAMODB_ENDPOINT` | — | Only for DynamoDB Local; leave empty on AWS (the Lambda/task role supplies credentials) |

`cdadmin` uses the same variables. On AWS, `cdadmin create-table` creates the table with its indexes, TTL and 7-day point-in-time recovery if CDK hasn't.

Serve the PWA from the same process (one port, no CORS):

```bash
(cd ../web && npm install && npm run build)
CD_WEB_DIR=../web/dist CD_DYNAMODB_ENDPOINT=http://localhost:8000 go run ./cmd/api     # open http://localhost:8080
```

## Onboarding shops (`cdadmin`)

Nobody hands out PINs. Creating or resetting an account prints a **one-time setup link** (valid 48 h) and a ready-to-send message; the person opens it and chooses their own PIN. Set `CD_PUBLIC_WEB_URL` (e.g. `https://counterdrop.in`) so links point at the right address.

```bash
go run ./cmd/cdadmin create-table                                 # once per AWS environment (not needed locally)
go run ./cmd/cdadmin create-shop  -slug imran-xerox -name "Imran Xerox" -address "Station Rd, Pune" -owner Imran
go run ./cmd/cdadmin add-staff    -shop imran-xerox -name Sana [-role staff|owner]
go run ./cmd/cdadmin reset-pin    -shop imran-xerox -name Sana     # old PIN stops working, signed out everywhere
go run ./cmd/cdadmin remove-staff -shop imran-xerox -name Sana
go run ./cmd/cdadmin list-staff   -shop imran-xerox
```

After the owner has set their PIN they manage staff themselves in **Settings → Staff** (add, reset PIN, remove), and everyone can change their own PIN under **My account**.

Rules: PINs are 4 digits and obvious ones (1234, 1111, 1212, 2580…) are refused; a link works once, only the newest link for a person works, and its token lives in the URL fragment so it never reaches server logs; a reset clears the old PIN and signs the person out; nobody can reset or remove themselves; a shop always keeps at least one owner.

## Layout

| Package | What |
| --- | --- |
| `internal/domain` | Pure rules: job state machine and effects, pricing and page ranges, tokens, wait estimate, shop hours |
| `internal/store` | `Repository` interface, shared types, business rules (`rules.go`), IDs/secrets/PIN hashing |
| `internal/store/ddbstore` | DynamoDB: single table, jobs with embedded files, four sparse GSIs, optimistic locking |
| `internal/backend` | Opens the DynamoDB store from the config |
| `internal/httpapi` | Routes, auth middleware, `{data}` / `{error:{code,message}}` envelope, handlers, end-to-end tests |
| `internal/realtime` | Server-Sent Events hub (job and shop topics) |
| `internal/tasks` | Deletion worker and draft abandonment |
| `internal/storage` | `ObjectStore`: R2/S3 presign, or local disk with HMAC-signed links |
| `cmd/api`, `cmd/cdadmin` | Server and admin CLI |

## API (prefix `/api/v1/cd`)

Customer — the ticket secret goes in the `X-Ticket-Secret` header (or `?secret=` for the event stream only):

| Method | Path | |
| --- | --- | --- |
| GET | `/shops/{slug}` | Public shop: status, hours, prices, wait |
| POST | `/shops/{slug}/jobs` | Create a draft with file metadata → ticket, secret, upload targets |
| GET / PATCH | `/jobs/{id}` | Ticket; update name and per-file settings |
| POST / DELETE | `/jobs/{id}/files[/{fileId}]` | Add / remove files |
| POST | `/jobs/{id}/files/{fileId}/complete` | Confirm upload (server checks size and type) with page count |
| POST | `/jobs/{id}/submit` | Send to counter (needs the quoted `priceVersion`) → token |
| POST | `/jobs/{id}/cancel` | Cancel and withdraw the files while in line |
| POST | `/jobs/{id}/delete-request` | After pickup: delete Counter Drop's copy now and ask the shop to delete any copy it downloaded |
| GET | `/jobs/{id}/events` | SSE stream |
| POST | `/jobs/{id}/live` | Live-update ticket: `{mode:"sse"}` or `{mode:"ws", url, ticket}` |

Shop — `Authorization: Bearer <session>`:

| Method | Path | |
| --- | --- | --- |
| GET | `/shop/staff-names?shop=` | Names for the sign-in tiles |
| POST | `/shop/login`, `/shop/logout`; GET `/shop/me` | PIN sign-in (5 wrong tries → 15 min lock) |
| POST | `/shop/setup/info` `{token}` | Check a setup link (who, which shop, expiry) — no auth |
| POST | `/shop/setup` `{token, pin}` | Use a setup link: set PIN and sign in — no auth |
| PUT | `/shop/me/pin` `{currentPin, newPin}` | Change own PIN; signs out your other devices |
| GET / POST | `/shop/staff` `{name, role}` | Owner: list staff; add a person → one-time link |
| POST | `/shop/staff/{id}/link` | Owner: new setup link (reset clears the old PIN) |
| DELETE | `/shop/staff/{id}` | Owner: remove a person |
| GET | `/shop/queue` | Board snapshot |
| POST | `/shop/claim-next` | Claim the oldest job in a lane (`SKIP LOCKED`) |
| POST | `/shop/jobs/{id}/{claim,release,ready,collected,undo,cancel}` | Actions |
| GET | `/shop/jobs/{id}/files/{fileId}/url?mode=print\|download` | Short-lived link: `print` opens in the browser; `download` saves to the device and the customer is told live |
| POST | `/shop/jobs/{id}/copies-deleted` | Shop confirms it deleted the copies it downloaded (shown on the customer's receipt) |
| GET | `/shop/lookup?q=` | Find by token or name |
| PUT | `/shop/state` | Online / Paused / Offline |
| GET / PUT | `/shop/settings` | Owner: profile, hours, prices |
| GET | `/shop/deletion-health` | Owner: deletion backlog and failures |
| GET | `/shop/events?token=` | SSE stream |
| POST | `/shop/live` | Live-update ticket for the board |

## Live updates

Ticket pages and shop boards get pointer events (`job.updated`, `queue.changed`, `file.downloaded`, `job.files_deleted`, `shop.state`) and refetch through the normal API. The client first calls `POST …/live`; the answer picks the transport:

| `CD_REALTIME_WS_URL` | Transport | Used for |
| --- | --- | --- |
| empty (default) | Server-Sent Events from this process | Dev, single-server fallback |
| `local` | WebSocket served by this process at `/api/v1/cd/ws` | Dev: runs the same browser code as AWS |
| `wss://ws.<domain>` | API Gateway WebSocket; pushes come from the DynamoDB stream | AWS (ADR-001) |

WebSocket connections open with a 60-second HMAC ticket (`CD_REALTIME_KEY`, 32+ characters, the same value in the API and the WebSocket Lambda); the job secret or session token never goes in a URL. The client reconnects with a fresh ticket and backoff, sends a heartbeat every 5 minutes (API Gateway drops idle sockets after 10), and refetches after any gap.

On AWS the Lambda binary `cmd/lambda` handles it (see *Running on AWS Lambda* below):

- `lambda-ws`: `$connect` checks the ticket and records the connection in `CD_WS_CONNECTIONS_TABLE` (default `cd-connections`); `$disconnect` removes it.
- `lambda-push`: reads the main table's stream (new and old images), turns each change into events (`ddbstore.ChangeEvents`) and posts them through `CD_WS_MANAGEMENT_ENDPOINT`, dropping connections that are gone.

## Running on AWS Lambda (ADR-001)

One binary, `cmd/lambda`, four handlers chosen by `CD_RUNTIME`:

| `CD_RUNTIME` | Trigger | What it does |
| --- | --- | --- |
| `lambda-api` | API Gateway HTTP API (payload 2.0), behind CloudFront | The same `httpapi` server as `cmd/api`, through `internal/lambdahttp` |
| `lambda-sweeper` | EventBridge Scheduler, every minute, reserved concurrency 1 | One deletion-worker pass: drafts, uncollected jobs, due files (Delete + Head check). Returns an error if any step failed, so failed runs show in metrics |
| `lambda-ws` | API Gateway WebSocket routes | `$connect` ticket check, `$disconnect` cleanup |
| `lambda-push` | DynamoDB stream of `cd-main` | Live-update fan-out |

Build (arm64, runtime `provided.al2023`, handler file `bootstrap`):

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -trimpath -ldflags "-s -w" -o bootstrap ./cmd/lambda
```

Differences from the long-running server, enforced at cold start (the function refuses to start otherwise):

- **Live updates** use the WebSocket API: `CD_REALTIME_WS_URL=wss://…` and `CD_REALTIME_KEY` are required; the SSE endpoints answer 404.
- **Files** go to S3 (`CD_STORAGE_BUCKET`); there is no lasting local disk.
- **Shared rate limits:** sign-in, setup links, staff names and new jobs are counted in DynamoDB (`RL#<rule>|<ip>`, one-minute windows, TTL), so every Lambda instance sees the same count. Other limits are per instance; API Gateway throttling covers the rest.
- **Client IP:** `CD_CLIENT_IP_HEADER=CloudFront-Viewer-Address` makes limits use the viewer's address, not the CloudFront edge's.
- **Origin check:** with `CD_ORIGIN_SECRET` set, every request must carry `X-Origin-Verify` (CloudFront adds it as a custom origin header), so the API Gateway URL can't be used directly to bypass CloudFront.
- **No background goroutines:** the sweeper is its own scheduled function.

## Protection

- **Rate limits** per client IP (`CD_RATE_LIMIT`, on by default): sign-in 10/min, setup links 20/min, staff names 30/min, new jobs 10/min, uploads 60/min, live-update connections 30/min, other customer actions 120/min, everything else 600/min. Over the limit: HTTP 429 with `Retry-After`. Behind a proxy set `CD_TRUST_PROXY=true` so the real client IP (last `X-Forwarded-For` entry) is used; without it a forged header is ignored. Limits live in memory, so run one API instance.
- **Security headers** on every response: Content-Security-Policy (own scripts only, no inline scripts, no framing; uploads allowed to the storage origin), `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy`, `Permissions-Policy` (camera only for the in-app scanner), and HSTS when `CD_ENV=prod`. Customer files are served without a CSP so the browser's PDF viewer works.
- **Concurrent writers** are safe: every change is a conditional write on the job's version, retried from a fresh read, so two counters (or two containers during a deploy) never act on the same job at once.

## Tests

```bash
docker compose -f ../deploy/docker-compose.yml up -d dynamodb
CD_TEST_DYNAMODB_ENDPOINT=http://localhost:8000 go test ./...
```

Each test gets its own table, dropped afterwards. The end-to-end tests are skipped when `CD_TEST_DYNAMODB_ENDPOINT` is not set.
