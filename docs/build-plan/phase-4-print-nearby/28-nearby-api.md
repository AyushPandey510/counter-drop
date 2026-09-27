# 28 — Geo search and nearby API

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 4 Print nearby | R1b | M | 09 | FSD SCR-A03, FS-5.1–5.2, FS-9.14, §14 API-11, FR-3.1–3.5, BR-R2, BR-R4–R5, AT-22 |

## Goal

`GET /nearby` returns orderable and non-orderable shops within 10 km in under 500 ms p95, with live wait, prices, open state and a "best pick" ranking.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md SCR-A03, FS-5.1, FS-5.2, FS-9.14 (best pick formula), §14 API-11, and docs/BRD.md FR-3.1–3.5, BR-R2, BR-R4, BR-R5. Read migration 0004 (location, GIST index) and the wait engine from step 09.

Task: implement nearby search.

1. GET /api/v1/cd/nearby?lat&lng&radius (m, default 2000, max 10000)&sort=best|distance|wait|price|rating&filters=open,colour,lamination,spiral,wait15&area= (alternative to lat/lng: area or station slug resolved from cd_clusters/stations table).
2. Query: ST_DWithin(location, ST_MakePoint(lng,lat)::geography, radius) with ST_Distance, only status=live shops; join hours, price list summary, cached wait. Expand radius 2 → 5 → 10 km until ≥ 5 orderable shops (FS-5.1).
3. Orderable = live AND online AND not paused AND remote_enabled AND payout_status active AND open ≥ 30 min after the estimated ready-by (BR-R2) AND not hidden by a timeout penalty (BR-R4: hidden_until > now). Non-orderable shops are returned with reason: closed | paused | not_online_orders | busy_penalty.
4. Best pick (FS-9.14) with weights from config (CD_RANK_W_WAIT 0.35, DIST 0.25, PRICE 0.15, ACCEPT 0.15, RATING 0.10); each factor normalised 0–1 across the result set (lower wait/distance/price is better); shops without enough data get neutral 0.5 for acceptance and rating.
5. GET /nearby/waits?ids= → lightweight wait refresh for up to 30 shops (FS-5.2); Cache-Control max-age 15.
6. GET /shops/{slug}/details → SCR-A06 data (full price list, hours per day, services, rating summary, last 5 comments once step 35 lands).
7. Response includes remoteOrdering=false and a banner code when Razorpay is marked unavailable (FS-10.10; a simple feature flag in cd_settings for now).
8. Data: seed a staging dataset of 30 shops around Dadar with varied hours, prices and waits (cmd/seednearby).
9. Tests: radius expansion; orderable rules for each reason; ranking order matches a hand-computed example; performance test with 5,000 shops across Mumbai asserting p95 < 500 ms locally (AT-22).

Rules: never store the customer's location (BR-D6); don't log lat/lng with more than 2 decimals. Show the SQL first.
```

## Acceptance

- [ ] AT-22 passes (p95 < 500 ms, correct greyed reasons, ranking matches formula).
- [ ] Customer coordinates are not persisted or logged precisely.

## Commit

`feat(nearby): PostGIS nearby search with orderability rules and best-pick ranking`
