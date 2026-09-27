# 04 — API foundation refactor

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 1 Backend core | R1a | M | 03 | FSD §14 conventions, §16, §17 observability |

## Goal

A clean base for every later endpoint: context everywhere, one error-to-HTTP mapping, the new envelope, request IDs, JSON logs, panic recovery, body limits and handlers split by area.

## Why now

`router.go` is one file with inline handlers, no context in the store and a different envelope from the contract. Fixing this before adding 40 endpoints avoids rewriting them later.

## Scope

**In:** router split, middleware chain, envelope v2 with a compatibility flag, `httpapi/errors.go`, request ID, slog JSON, recover, max body size, CORS for the web origin, context on the Store interface, ULID helper, clock interface.
**Out:** new endpoints, auth (step 07).

## Files

- `api/internal/httpapi/router.go` (routes only)
- `api/internal/httpapi/middleware.go` (request ID, logging, recover, CORS, body limit)
- `api/internal/httpapi/respond.go` (writeData, writeError, decodeJSON with validation)
- `api/internal/httpapi/errors.go` (sentinel → status + code)
- `api/internal/httpapi/handlers_public.go`, `handlers_shop.go`
- `api/internal/store/*.go` (add `ctx context.Context` first param; errors)
- `api/internal/platform/clock.go`, `api/internal/platform/ids.go` (new)
- `api/cmd/api/main.go` (JSON logger, wiring)

## Prompt

```text
First read docs/build-plan/00-common-context.md, then docs/FSD.md §14 (Conventions) and §16, then every file in api/internal/httpapi, api/internal/store and api/cmd/api/main.go.

Task: refactor the API foundation without changing business behaviour.

1. Create api/internal/platform:
   - clock.go: type Clock interface { Now() time.Time }; SystemClock (UTC); FixedClock for tests with Advance(d).
   - ids.go: NewID() string using github.com/oklog/ulid/v2 with a monotonic entropy source guarded by a mutex; NewSecret() (32 random bytes, base64url, no padding) and HashSecret(s) (SHA-256 hex); ConstantTimeEqual.
2. Store interface: add ctx context.Context as the first parameter on every method, in both PostgresStore and MemoryStore. Use ctx for all pgx calls (remove context.Background() inside the store).
3. Errors: move store sentinel errors to api/internal/store/errors.go. Add ErrConflict and ErrValidation (with a Fields map). Create api/internal/httpapi/errors.go with one function mapError(err) (status int, code string, message string, details map[string]any) that maps store and domain errors to the codes listed in contracts/openapi.yaml (not_found, forbidden, shop_paused, invalid_transition, validation codes). Unknown errors → 500 internal_error, and the real error is logged with the request ID but never sent to the client.
4. Envelope: implement writeData(w, status, v) → {"data": v} and writeError(w, r, err). Add config CD_LEGACY_ENVELOPE (default true for now): when true, responses also include "success": true/false and a string "error" field so current clients keep working. Step 17 will switch it off.
5. Middleware chain in middleware.go, applied in this order: recover (log stack, 500) → requestID (read X-Request-ID or generate ULID, set on response and in context) → slog request logger (method, route pattern, status, duration_ms, request_id; no query strings) → CORS (allow origins from CD_WEB_ORIGINS, comma-separated; allow headers content-type, authorization, x-ticket-secret, idempotency-key, x-request-id; credentials false) → maxBody (CD_MAX_JSON_BODY, default 1 MiB, for JSON routes only).
6. respond.go: decodeJSON(r, &dst) that rejects unknown fields, empty bodies and trailing data and returns ErrValidation with a field name.
7. Split router.go: NewRouter only registers routes; handlers live in handlers_public.go (health, shop info, create job, get job, submit) and handlers_shop.go (queue, actions). Keep exactly the same paths and behaviour.
8. main.go: slog.NewJSONHandler; log level from config; add a Server struct holding Store, ObjectStore, Clock, Logger, Config, passed to handlers (no globals).
9. Tests:
   - httpapi tests with httptest for: health, shop not found → 404 with code not_found, invalid JSON → 400 validation code, legacy envelope on/off, request ID echoed, panic → 500 without leaking the panic message.
   - platform tests for NewID ordering, secret length and hashing.
   - Existing domain test keeps passing.
10. Update contracts only if you discover a mismatch; list it in your final message.

Rules: no behaviour change visible to the current curl flow in api/README.md (run it at the end and paste results). Show the plan first.
```

## Acceptance

- [ ] The curl flow in `api/README.md` works unchanged with `CD_LEGACY_ENVELOPE=true`.
- [ ] With `CD_LEGACY_ENVELOPE=false`, responses match the contract envelope.
- [ ] Every response carries `X-Request-ID`; logs are JSON with no query strings.
- [ ] All store methods take `context.Context`.
- [ ] `make test lint` green.

## Verify

```bash
make test-api lint-api
CD_LEGACY_ENVELOPE=false make api   # then curl /api/v1/cd/shops/nope → {"error":{"code":"not_found",...}}
```

## Commit

`refactor(api): middleware chain, error mapping, envelope v2, context-aware store`
