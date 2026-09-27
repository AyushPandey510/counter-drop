# Counter Drop API

Go service for the Counter Drop customer upload flow, shop queue, staff actions, storage upload preparation, and future retention/payment work.

## Run Locally

Start Postgres from the project root:

```bash
cd /home/ayush/counter-drop
docker compose -f deploy/docker-compose.yml up -d postgres
```

Run the API:

```bash
cd /home/ayush/counter-drop/api
CD_API_ADDR=:18080 \
CD_DATABASE_URL='postgres://counter_drop:counter_drop@localhost:55433/counter_drop?sslmode=disable' \
CD_STORAGE_ENDPOINT='http://localhost:9000' \
CD_STORAGE_BUCKET='counter-drop-local' \
CD_STORAGE_ACCESS_KEY='counterdrop' \
CD_STORAGE_SECRET_KEY='counterdrop123' \
go run ./cmd/api
```

Without `CD_DATABASE_URL`, the API falls back to an in-memory store for quick demos. Without complete storage config, job creation still works but responses will not include `uploadUrl`.

## First Flow

Check health and shop info:

```bash
curl http://localhost:18080/health
curl http://localhost:18080/api/v1/cd/shops/demo-print
```

Create a job:

```bash
curl -X POST http://localhost:18080/api/v1/cd/shops/demo-print/jobs \
  -H 'Content-Type: application/json' \
  -d '{"customerName":"Ayush","files":[{"filename":"notes.pdf","size":12345,"mime":"application/pdf"}],"settings":{"copies":1,"color":"bw"}}'
```

The response includes:

- `id`
- `token`
- `secret`
- file `objectKey`
- file `uploadStatus`
- file `uploadUrl` when storage config is present

After the browser uploads each file with `PUT uploadUrl`, submit the job:

```bash
curl -X POST 'http://localhost:18080/api/v1/cd/jobs/JOB_ID/submit?secret=SECRET'
```

Read the customer ticket:

```bash
curl 'http://localhost:18080/api/v1/cd/jobs/JOB_ID?secret=SECRET'
```

View the shop queue:

```bash
curl 'http://localhost:18080/api/v1/cd/shop/queue?shop=demo-print'
```

Move the job through staff actions:

```bash
curl -X POST http://localhost:18080/api/v1/cd/shop/jobs/JOB_ID/claim
curl -X POST http://localhost:18080/api/v1/cd/shop/jobs/JOB_ID/ready
curl -X POST http://localhost:18080/api/v1/cd/shop/jobs/JOB_ID/collected
```

Other supported staff actions are `cancel` and `release`.

## Environment

```bash
CD_API_ADDR=:18080
CD_DATABASE_URL=postgres://counter_drop:counter_drop@localhost:55433/counter_drop?sslmode=disable
CD_STORAGE_ENDPOINT=http://localhost:9000
CD_STORAGE_REGION=auto
CD_STORAGE_BUCKET=counter-drop-local
CD_STORAGE_ACCESS_KEY=counterdrop
CD_STORAGE_SECRET_KEY=counterdrop123
```


## File Deletion Behavior

When staff marks a job `collected`, each file is marked with `deleteStatus: pending` and a `deleteAfter` timestamp 15 minutes in the future. This gives the shop a short recovery window for accidental clicks.

When staff `cancel`s a job, each file is marked `deleteStatus: pending` with `deleteAfter` set immediately.

The next backend step is a deletion worker that will scan pending files, delete the storage objects, and set `deletedAt` / `deleteStatus: deleted`.

## Tests

Run tests from this `api/` directory:

```bash
go test ./...
```

Running `go test ./...` from the repository root does not work because the root is a `go.work` workspace, not a Go module.

## Structure

```text
api/
├── cmd/api/                  # main package
├── internal/config/          # env loading
├── internal/domain/          # job state machine, shop, file upload state
├── internal/httpapi/         # handlers, middleware, JSON envelope
├── internal/store/           # memory and Postgres stores
├── internal/storage/         # S3-compatible presigned uploads
├── internal/realtime/        # future WebSocket topics
├── internal/tasks/           # future cleanup and expiry workers
├── migrations/               # SQL migrations
├── queries/                  # future sqlc queries
└── Dockerfile
```

## Storage Caveat

The API generates presigned URLs without contacting storage, so this flow can be developed before local MinIO is fully working. Actual `PUT uploadUrl` testing still needs a working S3-compatible target such as Cloudflare R2 or a local MinIO image that pulls successfully on this machine.
