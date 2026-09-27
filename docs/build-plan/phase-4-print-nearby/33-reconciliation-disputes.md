# 33 — Reconciliation, disputes and admin money pages

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 4 Print nearby | R1b | M | 32 | FSD §7 (Orders, Disputes, Reconciliation pages, FS-7.2), FS-10.8, SCR-A15, API-15, API-61–62, BR-P9, UC-A03–A05, UC-C18, UC-S25 |

## Goal

Finance can prove every rupee each day, support can find any order and refund within policy, and customers can report a problem that gets decided within 48 hours.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §7 (Orders, Disputes, Reconciliation pages and FS-7.1–7.3), FS-10.8, SCR-A15, §14 API-15, API-61, API-62, docs/BRD.md BR-P9, UC-A03–A05, UC-C18, UC-S25. Read the payments and settlement code from steps 30–32 and the admin console from step 23.

Task: reconciliation, disputes and admin money tools.

1. Daily reconciliation task at 02:00 IST (FS-10.8): fetch the previous day's payments, refunds, transfers and settlements from Razorpay (paginated); compare with cd_payments, cd_refunds, cd_transfers and cd_ledger; write cd_recon_runs and cd_recon_items (type, razorpay_id, job_id, expected, actual, status matched|missing_local|missing_remote|amount_mismatch). Mismatches alert finance. Re-run for a date from admin.
2. Admin Orders page: search by order/job ID, token + shop, pickup code, Razorpay payment or refund ID; timeline from cd_job_events + ledger; manual refund (full/partial with reason, within BR-P6–P9); refunds above ₹500 need a second admin's approval (FS-7.2) — cd_refund_approvals.
3. Disputes (UC-C18, SCR-A15): customer POST /jobs/{id}/disputes {reason: wrong_settings|missing_pages|poor_quality|charged_wrongly|shop_closed|other, comment, photo?} within 48 h of collection; optional photo uploaded to R2 under disputes/ with 30-day deletion. Shop owner sees it (UC-S25) and can respond: reprint / approve refund / disagree with note. Admin Disputes page with 48 h SLA timer; decision reprint | refund (amount) | reject; both sides notified.
4. Admin Reconciliation page: runs by date, counts by status, drill-down, mark resolved with note.
5. Tests: reconciliation with seeded mismatches of each type; four-eyes refund approval; dispute lifecycle and SLA flag; photo deletion scheduling.

Rules: admins still never see customer print files (dispute photos are separate, customer-provided evidence and are deleted after 30 days). Show the recon tables and statuses first.
```

## Acceptance

- [ ] Reconciliation shows zero unexplained mismatches for 2 consecutive weeks on staging test data.
- [ ] Refunds over ₹500 require two admins.
- [ ] Disputes are decided within the 48 h SLA timer.

## Commit

`feat(finance): daily reconciliation, disputes with SLA and admin order tools`
