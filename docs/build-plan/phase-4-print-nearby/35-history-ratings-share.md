# 35 — History, reorder, ratings, share target, favourites, delete account UI

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 4 Print nearby | R1b | M | 31 | FSD SCR-A13, A14, A16, §5.0 share target, API-14, API-16, UC-C20–C25, BR-D7, AT-23 |

## Goal

Customers come back: order history with one-tap reorder (files picked again), favourite shops, a one-tap rating after pickup with receipts, and "Share → Counter Drop" from WhatsApp or Gallery on Android.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md SCR-A13, SCR-A14, SCR-A16, §5.0 (share target row), §14 API-14 and API-16, docs/BRD.md UC-C20–C25 and BR-D7. Read web/src and the step 27 /me endpoints.

Task: retention features.

1. Migration 0010_ratings_favourites.sql: cd_ratings (job_id unique, user_id, shop_id, stars 1–5, tags text[], comment ≤200, created_at, shop_reply), cd_favourites (user_id, shop_id). Update shop rating_avg/count on insert (trigger or task).
2. API-14 POST /jobs/{id}/rating (after collected, once), API-16 GET /me/jobs (metadata only: shop, date, pages, total, status — no file names), GET/PUT/DELETE /me/favourites/{shopId}, GET /jobs/{id}/receipt (items, fee, total, payment and refund refs, shop GSTIN if set; printable HTML).
3. SCR-A13 after collected: stars + tags (Fast, Good quality, Friendly, Wrong settings, Long wait) + optional comment, and the receipt link. Walk-in tickets can also rate (guest, tied to the job) — decide: allow guest rating with the ticket secret, one per job.
4. SCR-A14 Orders tab (signed-in): list and detail; Reorder opens the remote flow for the same shop with the same settings and a clear "Pick your files again — we delete files after every order" message (BR-D7).
5. Favourites: heart on shop cards and details; Favourites filter chip in Nearby.
6. Share target (Android, installed PWA): manifest share_target {action: "/share", method: "POST", enctype: "multipart/form-data", params: {files: [{name: "files", accept: ["application/pdf","image/*"]}]}}. Service worker intercepts POST /share, stores files in Cache Storage/IndexedDB temporarily, redirects to /share/start which opens the remote flow with files attached and the best-pick shop preselected (needs location or last area). Temporary copies are removed as soon as the upload starts or after 15 minutes.
7. SCR-A16 Profile: name, masked phone, language, notification toggles, favourites, Delete account (calls DELETE /me with an explanation of what is kept for tax — BR-D5).
8. Tests: rating once-only; history contains no file names; reorder prefill; Playwright for share target using a POST to /share with a multipart body against the built SW (AT-23); delete account flow.

Rules: never keep shared files longer than the flow needs. Show the share-target SW flow first.
```

## Acceptance

- [ ] AT-23: sharing a PDF from WhatsApp opens the order screen with the file attached (installed PWA, Android).
- [ ] History shows no file names; reorder asks for files again.
- [ ] Ratings update shop averages.

## Commit

`feat(retention): order history, reorder, ratings, favourites, receipts and Android share target`
