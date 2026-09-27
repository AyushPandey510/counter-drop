# 01 — Dev environment, Makefile and CI

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 0 Foundation | R1a | S | — | FSD §0, NFR-21 |

## Goal

One command starts everything locally, one command runs all tests and linters, and every push runs the same checks in CI.

## Why now

Every later step relies on `make test` and `make lint` as its definition of done. Without CI, regressions in the state machine or pricing slip in silently.

## Scope

**In:** Makefile, golangci-lint config, GitHub Actions workflow, `.env` loading for local runs, test database in docker-compose, `.editorconfig`, pre-commit hook (optional), README quick start.
**Out:** web tooling (step 15 adds its own targets), deployment (step 25).

## Files

- `Makefile` (new)
- `.golangci.yml` (new)
- `.github/workflows/ci.yml` (new)
- `deploy/docker-compose.yml` (add `postgres-test` service on port 55434, tmpfs)
- `deploy/.env.example` (document every `CD_*` variable)
- `.editorconfig`, `.gitignore`, `.gitattributes` (already added in repo prep on 27 Sep 2026 — review and extend only)
- `README.md` (quick start section)

## Prompt

```text
You are working in the Counter Drop monorepo at the repo root.

First read:
- docs/build-plan/00-common-context.md
- README.md, api/README.md, deploy/docker-compose.yml, deploy/.env.example, api/go.mod

Task: set up the developer environment and CI so every later step can rely on `make test` and `make lint`.

Do this:

1. Makefile at repo root with these targets (use .PHONY, keep each target short, print what it does):
   - `db-up`: start postgres and postgres-test via docker compose (deploy/docker-compose.yml).
   - `db-down`: stop them.
   - `db-reset`: drop and recreate the dev database volume (ask for confirmation with a `CONFIRM=1` variable).
   - `api`: run the API from api/ with env loaded from deploy/.env if it exists (use `set -a; . ../deploy/.env; set +a`).
   - `test-api`: `cd api && go test -race -count=1 ./...` with `CD_TEST_DATABASE_URL` pointing at postgres-test.
   - `lint-api`: `cd api && golangci-lint run ./...` and `go vet ./...`.
   - `fmt`: gofmt + goimports on api/ and agent/.
   - `test`, `lint`: aggregate targets (web targets will be added in step 15; leave a placeholder comment).
   - `dev`: db-up then api.
2. docker-compose: add a `postgres-test` service (postgres:17-alpine, port 55434, db/user/password `counter_drop_test`, data on tmpfs so it is always fresh). Keep the existing `postgres` service unchanged. Leave the minio service but add a comment that local dev uses Cloudflare R2 (step 02) because the MinIO image pull failed on this machine.
3. deploy/.env.example already documents the current CD_* variables; add `CD_TEST_DATABASE_URL=postgres://counter_drop_test:counter_drop_test@localhost:55434/counter_drop_test?sslmode=disable`. .gitignore already ignores deploy/.env — don't duplicate it.
4. .golangci.yml: enable govet, staticcheck, errcheck, ineffassign, unused, gosec (exclude G104 duplicates of errcheck), revive (exported rule off), bodyclose, sqlclosecheck, rowserrcheck, contextcheck, misspell (locale US), gofmt, goimports. Timeout 5m. Exclude _test.go from gosec.
5. GitHub Actions `.github/workflows/ci.yml`:
   - Trigger on push and pull_request.
   - Job `api`: ubuntu-latest, Postgres 17 service container on 5432 with the test credentials, setup-go using the version in api/go.mod, cache modules, run golangci-lint (official action), then `go test -race -count=1 ./...` inside api/ with CD_TEST_DATABASE_URL set.
   - Leave a commented-out `web` job skeleton for step 15.
6. `go mod tidy` in api/ so direct dependencies are no longer marked `// indirect` (aws-sdk-go-v2, pgx). Do not upgrade versions.
7. `.editorconfig` exists; leave it unless something is missing. Delete `.gitkeep` files in folders that now contain code. Read docs/CODE-STATUS.md and fix issue 13 (go mod tidy) and 15 (Go version consistency) here.
8. README.md: replace the quick start with `make db-up`, `cp deploy/.env.example deploy/.env`, `make api`, `make test`. Keep the rest of the README.

Rules:
- Do not change any Go behaviour in this step.
- If golangci-lint reports existing issues, fix only trivial ones (unchecked errors, formatting). List anything larger in a TODO section of your final message instead of fixing it.
- Show me the plan first, then implement.

When done, run `make db-up`, `make test-api`, `make lint-api` and paste the output summary.
```

## Acceptance

- [ ] `make db-up && make test-api` passes on a clean clone.
- [ ] `make lint-api` passes (or remaining issues are listed as TODOs).
- [ ] CI runs on a pushed branch and is green.
- [ ] `deploy/.env` is ignored by git; `.env.example` documents every variable.
- [ ] `api/go.mod` lists aws-sdk-go-v2 and pgx as direct requirements.

## Verify

```bash
make db-up
cp deploy/.env.example deploy/.env
make test-api
make lint-api
git status   # deploy/.env must not appear
```

## Commit

`chore: add Makefile, golangci-lint, CI workflow and test database`
