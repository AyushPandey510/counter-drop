# 23 — Admin console (R1a)

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 3 Pilot readiness | R1a | M | 12, 14 | FSD §7, FS-7.1–7.3, FS-2.2, FS-17.10, UC-A01, A06, A07, A11, A12 |

## Goal

The internal team can approve and suspend shops, see deletion health, export audit trails and watch pilot KPIs, without ever touching customer files.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §7 (pages table and FS-7.1–7.3), FS-2.2, FS-17.10, and docs/BRD.md §10 UC-A01, A06, A07, A11, A12 and §21 metrics. Read the audit, deletion-health and task code from steps 12 and 14.

Task: build the R1a admin console (API + web at /admin).

1. Admin auth: Google sign-in (OIDC) restricted to emails in CD_ADMIN_EMAILS plus TOTP (github.com/pquerna/otp) enrolled on first login; admin sessions 8 h; roles ops, support, finance, founder stored in cd_admins. Separate cookie/session kind "admin"; never reuse shop sessions.
2. API (all under /api/v1/cd/admin, all audited with admin email and reason where the action changes state):
   - GET /admin/shops?status=&q= → table rows per FSD §7 (name, cluster, status, plan, jobs 7d, timeouts 7d placeholder, complaints placeholder).
   - POST /admin/shops/{id}/approve | /suspend {reason} | /unsuspend. Suspended shops: drop page shows "This shop isn't taking jobs on Counter Drop right now".
   - GET /admin/shops/{id}/board → read-only board with names masked (first letter + "•••") and NO file URLs (FS-7.1).
   - GET /admin/deletion-health (from step 14), POST /admin/deletion/retry/{fileId}.
   - GET /admin/audit?shop=&job=&from=&to= → CSV export streamed.
   - GET /admin/kpis?from=&to= → jobs/day, per-shop jobs, median claimed→collected minutes, peak-hour jobs (8–11, 17–20 IST), scan-to-token median (from analytics once step 24 lands; null until then), deletion SLA p99, active shops.
   - POST /admin/broadcast {message, shopIds|all} → banner on shop dashboards (store in cd_broadcasts; shown via queue snapshot).
   Enforce FS-2.2: add a test that walks every admin handler response and fails if any JSON key is objectKey, uploadUrl, url (for files) or filename after deletion.
3. Web /admin: pages Shops, Shop detail (profile, hours, lanes, members count, read-only board), Deletion health, Audit export, KPIs (simple tables plus one line chart of jobs/day using a lightweight chart lib), Broadcast. Dark theme. Desktop-first.
4. Tests: role checks (support cannot suspend? define: ops and founder can approve/suspend; support read-only in R1a), audit entries for every mutation, masking, CSV export correctness.

Rules: the admin API lives behind a separate route group with its own middleware; IP allowlist optional via CD_ADMIN_IP_ALLOWLIST. Show the admin role matrix first.
```

## Acceptance

- [ ] Ops can approve and suspend shops; each action is audited with a reason.
- [ ] No admin response contains a file URL or object key (automated test).
- [ ] KPIs page shows pilot numbers from real data.

## Verify

```bash
make test
```

## Commit

`feat(admin): R1a admin console with shop approval, deletion health, audit export and KPIs`
