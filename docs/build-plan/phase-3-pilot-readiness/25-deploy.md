# 25 — Deploy: staging and production

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 3 Pilot readiness | R1a | M | 24 | BRD BO-8 (cost under ₹2,000/month), NFR-1, NFR-10, FSD §1 components |

## Goal

Staging and production environments that cost under ₹2,000 a month, deploy on merge, back up nightly and can be restored.

## Recommended setup (confirm before running the prompt)

| Piece | Choice | Why | Approx. cost |
| --- | --- | --- | --- |
| API + workers | 1 × AWS Lightsail instance (2 GB, Mumbai ap-south-1) running Docker Compose + Caddy (auto TLS) | You know AWS; flat price; low latency to Mumbai | ~$12/month |
| Postgres | Same instance, Postgres 17 + PostGIS container, nightly `pg_dump` to R2, WAL archiving later | Cheapest reliable start for 10–50 shops | included |
| Files | Cloudflare R2 (`counter-drop-prod`) | No egress fees, lifecycle rule | ~free at pilot volume |
| Web PWA | Cloudflare Pages (static), custom domain | Free, global CDN, headers file for CSP | free |
| DNS | Cloudflare | Same account | free |
| Staging | Second smaller Lightsail (1 GB) or same box with separate compose project | Real HTTPS for phone testing | ~$5–7/month |

Alternatives to note in the PR if you prefer: AWS App Runner + RDS (simpler ops, higher cost), Fly.io + Neon (free tiers, less control).

## Prompt

```text
First read docs/build-plan/00-common-context.md and the "Recommended setup" table in docs/build-plan/phase-3-pilot-readiness/25-deploy.md. Read deploy/, api/Dockerfile, web/ build config and the Makefile.

Task: make the system deployable to staging and production with the recommended setup.

1. api/Dockerfile: multi-stage (golang:1.27 builder with -trimpath -ldflags "-s -w -X main.version=$GIT_SHA", distroless/static nonroot runtime), copies migrations, HEALTHCHECK hitting /health.
2. deploy/prod/docker-compose.yml: services api (image from GHCR), postgres (postgis/postgis:17-3.5 with a named volume, not exposed publicly), caddy (reverse proxy for api.<domain> with automatic TLS, WebSocket support, gzip, security headers), backup (a small container running pg_dump nightly at 02:30 IST → gzip → upload to R2 bucket counter-drop-backups with 30-day retention via lifecycle).
3. deploy/prod/Caddyfile and deploy/prod/.env.example (all prod variables, no secrets committed). Document secret handling: secrets live only in the server's .env with 600 permissions (or AWS SSM Parameter Store if you prefer; implement a tiny loader that reads SSM when CD_SSM_PREFIX is set).
4. Web: Cloudflare Pages config — build command, output dir, _headers file with CSP (from step 15), cache headers (immutable for hashed assets, no-cache for index.html and the service worker), _redirects for SPA routes. Environment variables per env (VITE_API_BASE_URL, VITE_SHOP_LINK_HOSTS, VITE_FEATURE_NEARBY=false).
5. GitHub Actions:
   - build-and-push: on main, build the API image, tag with sha, push to GHCR.
   - deploy-staging: on main after build, SSH (appleboy/ssh-action) to staging, docker compose pull && up -d, run a smoke test (curl /ready, create a job against demo shop, storagecheck).
   - deploy-prod: manual workflow_dispatch with an environment approval, same steps, plus a DB backup before migrate.
6. Runbooks in docs/runbooks/: deploy.md (normal deploy, rollback to previous image tag), restore.md (restore pg_dump from R2 into a fresh instance — test it once on staging and record the time taken), incident.md (who to call, how to pause remote orders, how to put shops in read-only mode, status page update).
7. Domain setup doc: api.counterdrop.in, app.counterdrop.in (web), and the short link host for QR codes (cd.in is illustrative — replace with the domain you actually own; update VITE_SHOP_LINK_HOSTS and the QR kit).
8. Cost sheet in docs/runbooks/costs.md with the monthly estimate and what triggers an upgrade (e.g. > 50 shops, DB > 5 GB, CPU > 60%).

Rules: no manual steps that aren't written in a runbook. Show the compose file and workflow outline first.
```

## Acceptance

- [ ] Merge to main deploys to staging automatically and the smoke test passes.
- [ ] Production deploy needs a manual approval and takes a backup first.
- [ ] A restore from backup has been rehearsed on staging and timed.
- [ ] Monthly cost estimate is under ₹2,000.

## Verify

```bash
curl -s https://api.staging.<domain>/ready
```

## Commit

`chore(deploy): Docker, Caddy, Lightsail compose, Cloudflare Pages, CI/CD, backups and runbooks`
