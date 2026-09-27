# 02 — Object storage on Cloudflare R2 and upload verification

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 0 Foundation | R1a | S | 01 | FSD §1, §12, FS-17.3, FR-1.4 |

## Goal

Real uploads work end to end against a Cloudflare R2 bucket: presigned PUT with size and type bound, server-side verification that the object exists, presigned GET for staff, delete, and a lifecycle backstop.

## Why now

The README notes that `PUT uploadUrl` has never been tested because MinIO would not pull. Every customer flow depends on this, so unblock it first.

## Manual setup (you, before running the prompt)

1. Cloudflare dashboard → R2 → create buckets `counter-drop-dev` and `counter-drop-prod` (location hint: Asia-Pacific).
2. R2 → Manage API tokens → create a token with **Object Read & Write** scoped to both buckets. Note Access Key ID, Secret, and the S3 endpoint `https://<account_id>.r2.cloudflarestorage.com`.
3. Bucket settings → CORS for `counter-drop-dev`: allow origins `http://localhost:5173`, methods `PUT, GET, HEAD`, headers `content-type, content-length`, max age 3600.
4. Bucket settings → Object lifecycle rule: prefix `cd/`, delete objects after **1 day** (R2 lifecycle works in days; this is the backstop for FS-12.1).
5. Put the values in `deploy/.env` (never commit them).

## Scope

**In:** storage client wrapper with PresignPut (content-type + content-length), PresignGet (5 min), Head, Delete, object key scheme, config for durations, a CLI smoke test, unit tests with a fake.
**Out:** page counting (step 10), deletion worker (step 14).

## Files

- `api/internal/storage/s3.go` → split into `client.go` (interface + S3 impl) and `fake.go` (in-memory fake for tests)
- `api/internal/storage/keys.go` (object key builder)
- `api/internal/config/config.go` (new durations)
- `api/cmd/storagecheck/main.go` (smoke test CLI)
- `api/internal/httpapi/router.go` (use new interface name only; no behaviour change)
- `deploy/.env.example`

## Prompt

```text
First read docs/build-plan/00-common-context.md, then FSD sections 1, 12 and FS-17.3 in docs/FSD.md, then api/internal/storage/s3.go, api/internal/config/config.go, api/cmd/api/main.go and api/internal/httpapi/router.go.

Task: turn the storage package into a complete, tested S3-compatible client for Cloudflare R2.

1. Define in api/internal/storage/client.go:
   type ObjectStore interface {
       PresignPut(ctx context.Context, key, contentType string, contentLength int64) (url string, headers map[string]string, err error)
       PresignGet(ctx context.Context, key string, filename string) (url string, err error)
       Head(ctx context.Context, key string) (ObjectInfo, error)   // returns ErrObjectNotFound if missing
       Delete(ctx context.Context, key string) error               // missing object is NOT an error
   }
   type ObjectInfo struct { Size int64; ContentType string; ETag string; LastModified time.Time }
   var ErrObjectNotFound = errors.New("object not found")
   Implement S3Store with aws-sdk-go-v2 (keep UsePathStyle and BaseEndpoint logic). PresignPut must sign ContentType and ContentLength so the client cannot upload a different size or type; return the headers the browser must send. PresignGet sets ResponseContentDisposition to `inline; filename="<sanitised filename>"`. Durations come from config: CD_STORAGE_PUT_TTL (default 15m), CD_STORAGE_GET_TTL (default 5m).
2. api/internal/storage/keys.go: func FileKey(shopID, jobID, fileID string) string returning "cd/{shopID}/{jobID}/{fileID}". No file names or customer data in keys. Validate the IDs are non-empty and contain only [A-Za-z0-9_-].
3. api/internal/storage/fake.go: FakeStore implementing ObjectStore in memory (map key→ObjectInfo), with a Put(key, size, contentType) helper for tests. Presign URLs return "fake://put/{key}" style strings.
4. Keep the existing httpapi.UploadPresigner usage compiling: adapt router.go and main.go to the new interface (PresignPut with the declared file size), without changing HTTP responses except that each file now also includes `uploadHeaders`.
5. api/cmd/storagecheck/main.go: a small CLI that loads config, puts a 1 KB test object through a real presigned PUT (net/http), Heads it, presigns a GET and downloads it, compares bytes, deletes it, Heads again expecting ErrObjectNotFound, and prints PASS/FAIL per step. Exit code non-zero on failure.
6. Tests: unit tests for FileKey validation and FakeStore; an integration test for S3Store that runs only when CD_STORAGE_BUCKET and credentials are set (t.Skip otherwise) doing the same round trip as the CLI.
7. Update deploy/.env.example with CD_STORAGE_ENDPOINT (R2 format), CD_STORAGE_BUCKET, CD_STORAGE_ACCESS_KEY, CD_STORAGE_SECRET_KEY, CD_STORAGE_PUT_TTL, CD_STORAGE_GET_TTL, and a comment describing the R2 CORS and lifecycle settings from docs/build-plan/phase-0-foundation/02-storage-r2.md.
8. Update api/README.md "Storage Caveat" section: R2 is now the dev target; show `go run ./cmd/storagecheck`.

Rules: no logging of keys with customer data (keys contain only IDs, which is fine). Show the plan first.
```

## Acceptance

- [ ] `go run ./cmd/storagecheck` prints PASS for put, head, get, delete, head-after-delete against R2.
- [ ] A PUT with a different Content-Length than signed is rejected by R2 (manual curl check).
- [ ] Unit tests pass without R2 credentials; the integration test passes with them.
- [ ] Create-job response still works and now includes `uploadHeaders` per file.

## Verify

```bash
cd api
set -a; . ../deploy/.env; set +a
go run ./cmd/storagecheck
go test ./internal/storage/...
```

## Commit

`feat(storage): R2-ready object store with presigned PUT/GET, head, delete and smoke test`
