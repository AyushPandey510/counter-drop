# 11 — Shop API: queue, claim, actions, lookup, state, settings

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 1 Backend core | R1a | L | 07, 10 | FSD §6, §14 API-30–44, FS-6.1–6.4, BR-Q1–Q5, EX-S01–S12 |

## Goal

Everything the counter and the owner need over HTTP: the queue snapshot, claim-next, every job action with reasons, file access via short-lived URLs, lookup by token or code, online/paused/offline, and owner settings.

## Endpoints

API-30 queue · API-31 claim-next · API-32 actions (claim, release, ready, collected, undo, cancel) · API-34 edit/price override · API-35 ask customer · API-36 file URL · API-37 lookup · API-38 shop state · API-39 profile/hours/prices/lanes · API-44 QR kit data (PDF render comes in step 22).

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §6 (all SCR-S screens and FS-6.1–6.4), §14 API-30 to API-44, §2 permission matrix, §16 VAL-SH1–SH4, docs/BRD.md BR-Q1–Q5 and EX-S01–S12. Read the code from steps 06, 07, 09, 10.

Task: implement the shop-side API. All routes require a staff or owner session; shop_id always comes from the session.

1. API-30 GET /shop/queue → QueueSnapshot: lanes (queued cards ordered by queued_at), printing (claimed), ready, collectedTray (collected within undo window, with undoUntil), remotePending (empty in R1a), shopState, wait, todayCount, and a monotonically increasing snapshotSeq (for step 13). JobCard fields per SCR-S03: token, firstName, channel badge, files × pages, settings summary string (e.g. "12 pp · B/W · 2-sided · ×2"), price, ageSeconds, ageLevel (normal <10 min, amber 10–20, red >20).
2. API-31 POST /shop/lanes/{laneId}/claim-next → uses store.ClaimNext (step 06); 404 lane_empty when nothing to claim.
3. API-32 POST /shop/jobs/{id}/{action} for claim|release|ready|collected|undo|cancel via domain.Transition with actor staff. cancel requires {reason} (preset codes: customer_request, file_problem, duplicate, other + free text ≤ 200). collected accepts {verifiedBy: token|code|name, paid?: cash|upi} (paid flag stored for reports only — FS-10.11). Execute the returned effects that exist so far (schedule deletion, clear deletion); leave others for later steps via an EffectExecutor interface with no-op implementations for Notify/Refund/Transfer.
   On a 409 from claim (someone else claimed), return {error:{code:"already_claimed", details:{claimedBy counter name}}} (MSG-S01).
4. API-34 PATCH /shop/jobs/{id}: copies, add-ons, per-file settings, and priceOverridePaise with mandatory reason (VAL-SH4: override > 3× original needs the owner role). Recompute quote; write audit with before/after.
5. API-35 POST /shop/jobs/{id}/ask {reason: file_corrupt|file_unreadable|wrong_file|clarify_settings|other, message?} → stores an "ask" event on the job (customer sees it in step 18); job keeps its token and position.
6. API-36 GET /shop/jobs/{id}/files/{fileId}/url → presigned GET (5 min) only if job is claimed or ready (or queued for preview), file not deleted; audit "file.opened" with staff and device. Also GET /shop/jobs/{id}/print.pdf that streams a merged PDF of all files in order with a cover page listing settings (use pdfcpu to merge; images placed on A4 fit-to-page) — used by "Print all" in step 21.
7. API-37 GET /shop/lookup?q= → matches token (A07, a-07, A-7 normalised), 4-digit pickup code, or first name prefix (case-insensitive) among today's non-terminal jobs; max 10 results.
8. API-38 PUT /shop/state {state: online|paused|offline, pauseMessage?}. Paused blocks new walk-in submissions (BR-Q5) but not existing jobs. Offline also sets remote intake off.
9. API-39 owner settings: GET/PUT /shop/profile (name, address, landmark, station, location lat/lng, gstin), /shop/hours (weekly + closures; VAL-SH2), /shop/prices (full price list; VAL-SH3; bumps version), /shop/lanes (up to 3 on pro, 5 on plus, 2 on free — read plan from shop; letter unique; rule bw|colour|remote|manual).
10. Auto-offline: at closing time, set online_state offline and flag uncollected jobs (EX-S05) — create the scheduled task row; executor in step 12.
11. Tests: queue ordering and card fields; claim race returns already_claimed; release returns to head of lane; ready → collected → undo within window; collected with wrong code → 409 code_mismatch; cancel requires reason; price override audit; lookup normalisation; paused shop blocks submit; staff cannot edit prices (403); file URL audited and refused after deletion.

Rules: no business logic in handlers — handlers parse, call a service in api/internal/app (create this package now for orchestration: load → domain → store → effects), and respond. Show the service interface first.
```

## Acceptance

- [ ] A staff session can run the full counter loop via curl: claim-next → print.pdf → ready → lookup → collected → undo.
- [ ] Owner-only settings return 403 for staff.
- [ ] Every action and file open is in `cd_audit_log`.

## Verify

```bash
make test-api
```

## Commit

`feat(shop): queue snapshot, claim-next, job actions, file access, lookup, state and settings API`
