# 22 — Shop settings: prices, profile, hours, lanes, staff and QR kit

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 2 Web PWA | R1a | M | 11, 20 | FSD SCR-S08–S12, FR-9.2–9.3, VAL-SH1–SH4, UC-S02–S06 |

## Goal

Owners manage everything themselves: price list with live preview, profile and map pin, hours and holidays, lanes, staff and devices, and a printable QR kit in three languages.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §6 SCR-S08 (price table), SCR-S10–S12 and FS details, FR-9.2–9.3, §16 VAL-SH1–SH4, and docs/BRD.md UC-S02–S06. Read API-39, 40, 44 in contracts/openapi.yaml and web/src.

Task: build owner settings under /shop/settings/* and the QR kit.

1. Prices (SCR-S08): grouped editor — B/W and Colour × paper (A4, Legal, A3) × one side (per side) / both sides (per sheet); Photo 4×6; add-ons list (name, unit per job / per item, price) with add/remove; minimum charge. ₹ inputs with 0.50 step, VAL-SH3 inline. Live preview sentence using the same pricing rules (import a TS port of the pricing function, with a shared test vector JSON generated from the Go tests so both implementations agree — add a Go test that writes api/testdata/pricing_vectors.json and a Vitest that reads it). Save bumps the version; show "Customers see new prices immediately".
2. Profile and hours (SCR-S09): name, address, landmark, station, map pin (reuse step 20), GSTIN (optional, format check), weekly hours and holiday dates (closures) with a calendar picker.
3. Lanes (SCR-S10): list with letter, name, rule (B/W jobs, Colour jobs, Manual); plan limits (Free 2, Pro 3, Plus 5) enforced with a friendly upsell message; reorder; can't delete a lane with jobs in it.
4. Staff and devices (SCR-S11): members list with role and last active; invite by phone (shows the join link/QR to scan on the staff phone); remove (confirm) → sessions revoked; devices list with revoke.
5. QR kit (SCR-S12): generate on the client with qrcode (SVG) and pdf-lib:
   - A4 counter standee: shop name, large QR for https://cd.in/s/{slug} (or the configured shop link host), short slug under it, 3 steps with icons in English, Hindi and Marathi ("Scan → Choose files → Show your token"), "Files deleted after pickup" shield, "No WhatsApp needed".
   - A3 window poster and an A4 sheet of 12 stickers.
   - Download buttons for each (PDF). High-contrast print-friendly colours.
6. Tests: price editor validation; pricing vectors agree between Go and TS; lanes limit; QR PDF contains the correct URL (decode the QR from the generated PDF page image in a test using jsQR on a rendered canvas, or assert the encoded string before rendering).

Rules: owner-only routes (redirect staff with a message). Show the price editor layout first.
```

## Acceptance

- [ ] Price changes are reflected in the next customer quote.
- [ ] Go and TypeScript pricing agree on every shared test vector.
- [ ] QR kit PDFs print cleanly and scan correctly with a phone camera.

## Verify

```bash
cd api && go test ./internal/domain/pricing/... && cd ../web && pnpm test settings
```

## Commit

`feat(shop): owner settings for prices, profile, hours, lanes, staff and printable QR kit`
