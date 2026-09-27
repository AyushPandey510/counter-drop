# 08 — Pricing engine and quotes

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 1 Backend core | R1a (fee for R1b) | M | 05 | FSD §9.1 (FS-9.1–9.6), BR-P1–P5, VAL-S1–S3, AT-02 |

## Goal

A pure, deterministic pricing function that turns a shop's price list and a job's file settings into a quote with line items, total in paise and a `price_version` hash.

## Why now

The drop page, the dashboard, remote payments and disputes all show the same numbers. One function, tested hard, prevents price arguments at the counter.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §9.1 (formulas, FS-9.1–9.6 and the worked example), §16 VAL-S1–S3, and docs/BRD.md BR-P1–P5. Read migration 0005 (cd_price_lists) and the seed price list.

Task: implement the pricing engine in api/internal/domain/pricing.

1. Types:
   - PriceList {Version int; Rates map[RateKey]int64 (paise); AddOns []AddOn{ID, Name, Unit: per_job|per_item, PricePaise}; MinChargePaise int64; ColourAvailable, DoubleSidedAvailable derived}.
   - RateKey {Mode: bw|colour; Paper: A4|Legal|A3|Photo4x6; Sides: one|both}. "one" is per side; "both" is per sheet.
   - FileSettings {Copies int; Mode; Sides; PageRange string; Paper; Orientation; Fit; AddOns []AddOnSel{ID, Qty}}.
   - FileInput {FileID; Kind: pdf|image; Pages int (0 = unknown); Settings}.
   - Quote {Lines []Line{FileID, SelectedPages, Sides, Sheets, Copies, RateKey, UnitPaise, AmountPaise}; AddOns []AddOnLine; PrintTotalPaise; FeePaise; TotalPaise; PagesToConfirm bool; PriceVersion string}.
2. ParsePageRange(expr string, pages int) ([]int, error): accepts "" (all), "1-3,5", spaces tolerated; errors page_range for descending ranges, zero, out of bounds, duplicates are merged. Unit tests with 20+ cases.
3. Quote(pl PriceList, files []FileInput, channel walkin|remote, feePolicy FeePolicy) (Quote, error):
   - sides_f = |P|; sheets_f = ceil(|P|/2).
   - one side: copies × sides × rate(mode,paper,one); both sides: copies × sheets × rate(mode,paper,both); if the shop has no "both" rate, charge 2 × one-side rate per sheet (FS-9.1).
   - images: one side each, sides forced to one; Photo4x6 uses the photo rate (FS-9.2).
   - unknown pages (Pages==0): the file's line amount is 0 and PagesToConfirm=true (FS-9.6); remote channel returns ErrPagesUnknown.
   - add-ons: per_job once, per_item × Qty.
   - total = max(min charge, sum); round only the final total to the nearest rupee, half up (FS-9.3) — i.e. round paise to a multiple of 100.
   - remote fee from FeePolicy (tiers: ≤5000 → 200, ≤20000 → 300, else 500; FirstOrderFree flag) — FS-9.4. Walk-in fee always 0.
   - Validation: copies 1–99, option not in price list → ErrOptionUnavailable with the field.
4. PriceVersion: hex SHA-256 (first 12 chars) of the canonical JSON of {price list version, rates, add-ons, min charge, normalised settings of every file, pages}. Same inputs → same version; any change → different version.
5. Tests: the FSD worked example (12-page PDF, pages 1–10, both sides, 2 copies, ₹3/sheet + lamination ₹20 → ₹50; remote → ₹52 with fee ₹2) — AT-02; min charge; missing double-sided rate; images; unknown pages walk-in vs remote; rounding (e.g. 4950 → 5000, 4949 → 4900); fee tiers; version stability and sensitivity. Add a fuzz test for ParsePageRange.
6. Store helper: LoadPriceList(ctx, shopID) returning the current version.

Rules: pure Go, no floats anywhere (use integer math). Show the type definitions first.
```

## Acceptance

- [ ] FSD worked example returns exactly ₹50 (walk-in) and ₹52 (remote).
- [ ] No `float` in the pricing package (`grep -r float api/internal/domain/pricing` is empty).
- [ ] Page range fuzz test runs 30 s without panics.

## Verify

```bash
cd api && go test ./internal/domain/pricing/... -v
go test ./internal/domain/pricing/ -fuzz=FuzzParsePageRange -fuzztime=30s
```

## Commit

`feat(pricing): deterministic quote engine with page ranges, add-ons, fees and price versions`
