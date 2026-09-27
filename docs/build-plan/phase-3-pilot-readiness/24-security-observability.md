# 24 — Security hardening and observability

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 3 Pilot readiness | R1a | M | 11 | FSD §17 (FS-17.1–17.10, observability table), §18 analytics, NFR-1–NFR-15, NFR-21 |

## Goal

The API is safe to expose to the internet and the team can see problems before shops do: rate limits, headers, dependency scanning, metrics, traces, error tracking, alerts and privacy-safe product analytics.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §17 (security table FS-17.1–17.10 and observability table), §18 (analytics events and rules), and docs/BRD.md §15 NFR-1 to NFR-21.

Task: harden and instrument the system.

Security
1. Rate limiting middleware backed by Postgres or in-memory token buckets keyed per route group (FS-17.6): job create 10/min per IP and per device fingerprint header; nearby 60/min; staff actions 120/min per device; OTP rules already in step 07. Return 429 rate_limited with Retry-After. Trust X-Forwarded-For only from CD_TRUSTED_PROXIES.
2. Security headers on API responses: HSTS (prod), X-Content-Type-Options nosniff, Referrer-Policy no-referrer, Cross-Origin-Resource-Policy same-site. Document the web CSP headers for the static host (step 25).
3. PDF safety (FS-17.7): page counting runs with a hard timeout, memory cap and recover(); never render PDFs server-side. Add a malformed-PDF corpus in api/testdata/fuzz and a fuzz test for the page-count wrapper.
4. Secrets: fail startup if required secrets are missing in prod (CD_ENV=prod), if CD_DEV_AUTH_BYPASS is set, or if ConsoleSender is configured.
5. Dependency and code scanning in CI: govulncheck, gosec (already in lint), pnpm audit --prod (fail on high), GitHub Dependabot config, secret scanning (gitleaks action).
6. Threat-model checklist in docs/SECURITY.md: assets (files, ticket secrets, sessions, payouts), threats (IDOR on jobs, token guessing, upload abuse, admin misuse), controls implemented, and remaining risks.

Observability
7. Metrics: Prometheus endpoint /metrics on a separate port (CD_METRICS_ADDR, not public) with http_requests_total, http_request_duration_seconds (by route pattern and status), ws_connections, outbox_lag_seconds, task_lag_seconds, deletion_lag_seconds, deletion_failures_total, pagecount_duration_seconds, jobs_submitted_total by shop, claims_conflict_total.
8. Tracing: OpenTelemetry SDK with OTLP exporter (optional via CD_OTLP_ENDPOINT); spans for HTTP, pgx (otelpgx) and S3 calls; sample 10%, always sample errors.
9. Error tracking: Sentry (or GlitchTip) for Go and web, with PII scrubbing (drop request bodies, headers except request ID; scrub phone/name/filename patterns).
10. Alerts (document as rules for Grafana Cloud free tier or the chosen stack): API 5xx > 1% for 5 min; deletion lag p99 > 5 min; any object older than 25 h found by the safety check; outbox lag > 30 s; task lag > 2 min; disk > 80%.
11. Product analytics (§18): POST /api/v1/cd/events accepting a batch of allow-listed events from the web (schema-validated, IDs and numbers only, no PII), stored in cd_analytics_events (partitioned by month) with server-side job events joined for KPIs. Wire the web no-op sender from steps 17–19 to it (sendBeacon on pagehide). No third-party trackers on the drop page.
12. Health: /health (liveness) and /ready (DB ping, storage head of a sentinel object, migrations applied).
13. Structured log review: grep the codebase for any logging of phone, name, filename, secret, token, pin, otp and fix; add a test that fails on those log keys (reuse the audit denylist).

Rules: nothing in this step may change product behaviour. Show the metrics list and alert thresholds first.
```

## Acceptance

- [ ] `govulncheck`, gitleaks and pnpm audit run in CI.
- [ ] Rate limits return 429 with Retry-After.
- [ ] /metrics exposes the listed metrics; a Grafana dashboard JSON is committed in `deploy/observability/`.
- [ ] Analytics events arrive without PII (test).

## Verify

```bash
make test lint
curl -s localhost:9090/metrics | grep deletion_lag
```

## Commit

`chore(security,obs): rate limits, headers, scanning, metrics, tracing, error tracking, analytics`
