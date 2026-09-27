# 26 — R1a acceptance tests and pilot runbook

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 3 Pilot readiness | R1a | M | 21, 22, 25 | FSD §19 AT-01–AT-14, AT-24, BRD §21 R1a acceptance criteria, §18 launch sequence |

## Goal

Every R1a acceptance test is automated and green on staging, and there is a written, rehearsed plan to onboard 10 pilot shops and measure counter time before and after.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §19 (AT-01 to AT-14 and AT-24) and docs/BRD.md §21 (metrics, R1a acceptance criteria) and §18 (launch sequence, onboarding checklist, support model).

Task: build the R1a acceptance suite and the pilot runbook.

1. e2e/ at the repo root: a Playwright project that runs against a deployed environment (BASE_URL, API_URL) with seeded data via an admin-only seeding endpoint enabled only on staging (CD_ENV=staging): create shop, owner, staff, price list; reset between tests.
2. Implement AT-01 to AT-14 and AT-24 as named tests (test title starts with the AT ID). Mobile emulation (Pixel 7, iPhone 14) for customer tests, desktop Chrome for shop tests. AT-04 (concurrency) runs as a Go load test in api/cmd/loadclaim that creates 500 jobs and claims with 3 workers against staging, printing duplicates=0.
3. Performance checks: Lighthouse CI on /s/demo-print (mobile, 4G throttling) asserting interactive < 2 s and PWA installable; k6 script for 20 shops × 30 jobs/hour to confirm p95 API latency < 300 ms and ws delivery < 1 s.
4. CI: a nightly workflow running e2e + Lighthouse against staging, posting a summary.
5. docs/pilot/runbook.md:
   - Shop selection criteria (footfall, owner openness, has a PC or smartphone), 10 shops across Dadar, Andheri, Thane.
   - Before go-live: shadow one peak hour per shop and record counter time per job (template CSV docs/pilot/counter-time.csv with columns shop, date, job#, arrive_counter, leave_counter, channel).
   - Go-live day checklist: signup with the owner, prices, QR kit placed at counter and window, staff logins, one test job, "No WhatsApp files" sign, owner's WhatsApp broadcast to regulars.
   - Weekly: usage review per shop, NPS, issues log, WhatsApp support group.
   - Week 4 and week 8: re-shadow peak hour; compute BO-1 and BO-2.
   - Exit criteria = Gate 1 from BRD §6.
6. docs/pilot/support-playbook.md: top 15 expected issues (QR won't scan, upload stuck, wrong price, printer settings, staff forgot PIN, shop offline) with the fix and what to tell the shop.
7. docs/pilot/feedback-form.md: 8 questions for owners and 5 for customers (Hindi and Marathi versions).

Rules: tests must be independent and idempotent; no reliance on test order. Show the test list mapping (AT ID → file → assertion) first.
```

## Acceptance

- [ ] AT-01–AT-14 and AT-24 green on staging; nightly run scheduled.
- [ ] Lighthouse: interactive < 2 s on mobile 4G; installable.
- [ ] Load test meets p95 < 300 ms.
- [ ] Pilot runbook reviewed and the first shop scheduled.

## Verify

```bash
cd e2e && BASE_URL=https://app.staging.<domain> API_URL=https://api.staging.<domain> pnpm exec playwright test
```

## Commit

`test(e2e): R1a acceptance suite, performance checks and pilot runbook`
