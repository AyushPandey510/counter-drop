# 10 — Walk-in job API

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 1 Backend core | R1a | L | 02, 06, 08, 09 | FSD §4, §14 API-01–10, §16 VAL-F/S/N/J, FS-5.0.1–5.0.2, EX-C01–C07 |

## Goal

The complete guest (upload-only) job flow over HTTP: create draft → presigned uploads → confirm each file (verify object, count pages) → edit settings and quote → submit (token) → view ticket → cancel.

## Endpoints

| ID | Route | Auth |
| --- | --- | --- |
| API-01 | `GET /shops/{slug}` | none |
| API-02 | `POST /shops/{slug}/jobs` | none |
| API-03 | `POST /jobs/{id}/files` | ticket |
| API-04 | `DELETE /jobs/{id}/files/{fileId}` | ticket |
| API-05 | `POST /jobs/{id}/files/{fileId}/complete` | ticket |
| API-06 | `PATCH /jobs/{id}` | ticket |
| API-07 | `GET /jobs/{id}/quote` | ticket |
| API-08 | `POST /jobs/{id}/submit` | ticket |
| API-09 | `GET /jobs/{id}` | ticket |
| API-10 | `POST /jobs/{id}/cancel` | ticket |

## Prompt

```text
First read docs/build-plan/00-common-context.md, then docs/FSD.md §4 (all walk-in screens and FS-4.1–4.9), §5.0 FS-5.0.1–5.0.2 (scan is upload-only), §14 API-01 to API-10 and the example for API-08, §16 validations VAL-F1–F4, VAL-S1–S3, VAL-N1, VAL-J1, and docs/BRD.md §13 EX-C01–C07. Read contracts/openapi.yaml for these operations and the code from steps 02, 06, 08, 09.

Task: implement API-01 to API-10 for walk-in (guest) jobs.

1. API-01 GET /shops/{slug}: public shop info (name, area, station, isOpen, opensAt, closesAt, onlineState, pauseMessage, wait range, lanes, price list summary for the settings UI: available modes/papers/sides/add-ons with prices). Only shops with status live. Cache-Control: max-age=15.
2. API-02 POST /shops/{slug}/jobs {files:[{clientId, filename, size, mime, sha256?}], customerName?}:
   - Validate VAL-F1 (mime whitelist: application/pdf, image/jpeg, image/png, image/heic — HEIC allowed but see note), VAL-F2 (≤ CD_MAX_FILE_BYTES 25 MiB each, ≤ CD_MAX_JOB_BYTES 50 MiB total), VAL-F3 (≤ 20 files).
   - Shop must be live and online (409 shop_paused / shop_closed with opensAt).
   - Create job in state uploading, channel walkin, default settings per file (bw, one side, 1 copy, all pages, A4, auto, fit), a new ticket secret (return raw secret ONCE; store hash), object keys via storage.FileKey, presigned PUT with headers.
   - Response: {job (ticket view), secret, files[{fileId, clientId, uploadUrl, uploadHeaders}]}.
   - customerName is now optional (FSD UX-C11); remove the old "customerName is required" check.
3. API-03 add file / API-04 remove file: only while uploading or queued-and-unclaimed; recalc totals.
4. API-05 complete: HEAD the object; size must equal declared size and content type must match; else 422 upload_mismatch and delete the object. Then count pages:
   - PDF: use github.com/pdfcpu/pdfcpu (api.PageCount on a streamed download with a 5 s timeout and a 50 MiB cap). Encrypted → pdf_locked; parse error → pdf_corrupt (file stays but job cannot submit until removed). > 500 pages → pdf_too_long.
   - Images: 1 page. HEIC: 1 page; set a flag heicNeedsConversion (the web converts HEIC to JPEG before upload where possible — step 17).
   - Unknown count (timeouts) → pages_status unknown → quote PagesToConfirm (walk-in allowed).
   - Mark upload_status uploaded, pages_status counted|unknown|failed.
   - Do the page count in a bounded worker pool (CD_PAGECOUNT_WORKERS default 4) so the HTTP call returns within 6 s; if still running, return 202 and let the client poll GET /jobs/{id}.
5. API-06 PATCH /jobs/{id}: {customerName?, files:[{fileId, settings}], applyToAll?: fileId}. Validate VAL-S1–S3 and VAL-N1. Allowed in uploading, or in queued before claim (edit transition). Returns the new quote.
6. API-07 GET quote: the pricing engine output for the job (walk-in fee always 0).
7. API-08 submit: body {priceVersion, customerName?}. Guards (VAL-J1): all files uploaded (409 uploads_incomplete), no failed/locked files, priceVersion current (409 price_changed with the new quote in details), shop online and not paused. Duplicate check FS-4.7: if any file sha256 matches a job at this shop submitted in the last 2 minutes and the request lacks confirmDuplicate=true → 409 duplicate_suspected with the other token. Transition submit → queued; execute effects: issue token (step 09), set lane (default lane by rule: colour jobs → lane with rule colour, else first bw lane), queued_at, ready_by. Response as in the FSD API-08 example.
8. API-09 GET /jobs/{id}: ticket view (token, lane, state, position, readyBy, quote, files with pages and settings but no object keys, timeline from cd_job_events, filesDeletedAt, deletion countdown if collected). Accept X-Ticket-Secret (and legacy ?secret= while CD_LEGACY_ENVELOPE=true).
9. API-10 cancel: customer cancel allowed in uploading and queued-before-claim; effects DeleteFilesNow (just mark delete_after=now; the worker in step 14 deletes).
10. Upload-only rule: walk-in jobs never expose payment fields, and POST /jobs/{id}/pay (stub now) returns 409 pay_not_allowed for channel walkin (FS-5.0.2).
11. Abandoned drafts: schedule T18 (abandon after 60 min of inactivity) via the effect list; the executor lands in step 12 — for now write the cd_scheduled_tasks row.
12. Rate limits (simple in-memory token bucket per IP + per shop until step 24): 10 job creations per minute per IP.
13. Tests (httptest + FakeStore + test DB): full happy path (create → PUT to fake → complete → patch → quote → submit → get) — AT-01 at API level; price correctness AT-02; locked PDF AT-08 (use a fixture encrypted PDF in api/testdata); mismatch size; duplicate submit; paused shop; cancel then get shows cancelled; secret wrong → 403; edit after claim → 409.

Rules: never return object keys or signed GET URLs to guests; never log file names. Keep the old create-job route shape working until step 17 switches the web (map old body to new). Show the plan first, including how the page-count pool and 202 fallback work.
```

## Acceptance

- [ ] API-level AT-01, AT-02, AT-08 pass.
- [ ] Wrong size upload is rejected and the object deleted.
- [ ] `POST /jobs/{id}/pay` on a walk-in job → 409 `pay_not_allowed`.
- [ ] No file names or object keys in logs or guest responses.

## Verify

```bash
make test-api
# manual with R2: run the curl flow in api/README.md (update it in this step)
```

## Commit

`feat(jobs): walk-in upload-only job API with verification, page count, quotes and submit`
