# 17 — Customer drop flow (upload only)

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 2 Web PWA | R1a | L | 10, 16 | FSD §4 SCR-W01–W03, W06, W07, FS-4.1–4.7, §5.0 FS-5.0.1, UX-C1–C12, EX-C01–C08 |

## Goal

A customer scans the shop QR and gets a token in under 30 seconds: choose files → (optional) settings → send. Uploads start on the first tap and survive bad networks. No login, no payment.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §4 SCR-W01, W02, W03, W06, W07 (tables and FS-4.1–4.7), §5.0 FS-5.0.1 (upload-only), §16 validations and messages, docs/BRD.md §8 UC-C01 main and alternate flows, §13 EX-C01–C08 and §14 UX-C1–C12. Read contracts/openapi.yaml API-01, 02, 03, 04, 05, 06, 07, 08 and web/src from steps 15–16.

Task: build the drop flow at /s/:slug.

1. SCR-W01 Shop landing: header (shop name, area, "Open · closes 9:30 pm"), wait chip ("6 jobs ahead · about 10–15 min" / "No wait"), language switcher, primary "Choose files" (input type=file multiple accept=".pdf,.jpg,.jpeg,.png,.heic,application/pdf,image/*"), secondary "Copy of ID card" (opens W07), footer shield "Files are deleted after pickup". If a non-closed ticket for this shop exists in localStorage (try/catch), show "You have A-07 in line — View" (FS-4.2). If paused/closed → render SCR-W06 (FS-4.1).
2. Upload engine (src/customer/upload/):
   - On pick: client-side validation (type, 25 MB/file, 50 MB/job, 20 files) with MSG texts; HEIC → convert to JPEG in the browser with heic2any when the browser can't display HEIC (lazy-load the library only when needed).
   - Compute SHA-256 per file with crypto.subtle in a Web Worker (don't block the UI) and send it on create.
   - POST /shops/{slug}/jobs, store {jobId, secret} in sessionStorage + localStorage (ticket restore), then PUT each file to its uploadUrl with the returned headers, max 3 concurrent, XHR for progress events.
   - Resilience (EX-C01): on network loss show "Waiting for network — we'll continue automatically"; retry with backoff when navigator.onLine becomes true; if the presigned URL expired (403), request a fresh one (add API-03 re-sign support or re-create the file entry) and continue.
   - After each PUT: POST …/files/{id}/complete; handle 202 (poll GET /jobs/{id} until pages known), pdf_locked, pdf_corrupt, upload_mismatch with clear per-file error rows and a Remove action.
   - Move to W02 immediately after picking; uploads continue in the background.
3. SCR-W02 File settings: list of file rows (thumbnail — first PDF page via pdf.js lazy-loaded, or image thumb —, name truncated, progress, "12 pages", subtotal). Settings sheet per file with the fields from the FSD table (copies stepper 1–99, B/W|Colour segmented — hidden if shop has no colour, One side|Both sides — hidden for images, Pages All|Range with validation, Paper, Orientation, Fit, Add-ons) and "Apply to all". Smart defaults so the customer can skip this screen. Debounced PATCH (500 ms) → updated quote (FS-4.5). "Add more files", swipe/✕ to remove.
4. SCR-W03 Review and send: summary, PriceBreakdown (per file, add-ons, total, "Pay at the counter"), ready-by, optional first name (VAL-N1), "Send to counter" disabled until uploads complete and pages known or flagged (EX-C05 → "Price will be confirmed at the counter"). Submit with priceVersion; handle price_changed (MSG-C18 then resubmit), duplicate_suspected (MSG-C06 with confirm), shop_paused (W06). On success navigate to /t/:jobId (step 18 builds that page; for now show the token big).
5. SCR-W06 closed/paused: hours or pause message, retry every 30 s while visible (Page Visibility API).
6. SCR-W07 ID-card capture: getUserMedia rear camera with a card-shaped overlay; capture front then back; auto-crop with a lightweight edge detection (or manual 4-corner adjust fallback); compose both onto one A4 PDF in the browser with pdf-lib (B/W default); add it as a file to the job. Aadhaar masking is R2 — leave a hook.
7. Upload-only rule: no payment UI, no nearby links, no account prompts anywhere in this flow (FS-5.0.1). The only extra element allowed is the install banner (step 19).
8. Switch the API client to CD_LEGACY_ENVELOPE=false shape and X-Ticket-Secret header; coordinate: set CD_LEGACY_ENVELOPE=false in deploy/.env.example.
9. Analytics hooks (no-op sender for now, real endpoint in step 24): drop_page_viewed, files_picked, upload_completed, settings_changed, job_submitted with ms since view (FSD §18).
10. Tests: Vitest for the upload queue (retry, concurrency, expiry), settings reducers and validation; Playwright against MSW: happy path under 30 s budget with 2 PDFs, network drop mid-upload (use route.abort then continue), locked PDF, paused shop, Hindi locale snapshot at 360 px.

Rules: mobile-first 360 px, one primary button per screen fixed at the bottom (UX-C2), 48 px targets. Keep the /s/:slug chunk within the 200 KB budget: lazy-load pdf.js, heic2any, pdf-lib and the camera code. Show the component tree and state machine for the upload engine first.
```

## Acceptance

- [ ] Scan-to-token under 30 s on a mid-range Android over 4G (manual, with the real API and R2).
- [ ] Upload resumes after airplane mode on/off.
- [ ] No payment, nearby or login UI in the flow.
- [ ] Bundle budget still passes.

## Verify

```bash
cd web && pnpm test && pnpm exec playwright test drop
make api & (cd web && pnpm dev)   # real end-to-end on a phone via LAN IP
```

## Commit

`feat(customer): upload-only drop flow with resilient uploads, settings, review, ID-card mode`
