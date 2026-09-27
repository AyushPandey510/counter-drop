# 29 — Nearby UI and shop details

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 4 Print nearby | R1b | M | 19, 28 | FSD SCR-A03–A06, EX-C14–C15, UC-C11–C12, CO-1 |

## Goal

From the PWA Home, **Print nearby** shows a fast, honest list (and map) of shops with distance, open state, wait and prices, with filters, and a shop details page with **Order here**.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md SCR-A03, SCR-A04, SCR-A05, SCR-A06, §16 MSG-C15, docs/BRD.md UC-C11, UC-C12, CO-1 and EX-C14–C15. Read API-11 and the details endpoint from step 28 and web/src from step 19.

Task: build /nearby and /nearby/:slug behind VITE_FEATURE_NEARBY.

1. Location: ask for geolocation only when the customer opens Print nearby (never on Home). Denied or slow (>5 s) → area/station picker (searchable list of Mumbai stations) — EX-C14. Show "Near Dadar West" with a Change button. Never persist precise coordinates; remember only the chosen area name.
2. List (SCR-A03): sort segmented control (Best pick, Nearest, Shortest wait, Cheapest, Top rated), filter chips (Open now on by default, Colour, Lamination, Spiral, Wait under 15 min). Shop card: name, distance and walking minutes (distance / 80 m per min), "Open · closes 9:30 pm", wait chip, B/W and colour price per side, rating, service badges. Non-orderable shops greyed with the reason text. Wait chips refresh every 30 s while visible (FS-5.2).
3. Map view (SCR-A04): Leaflet with OSM tiles (lazy-loaded), pins coloured by wait band with icon + text in popups; tapping opens the card.
4. Empty state (EX-C15): MSG-C15 with the nearest open shop and a "Search 10 km" button.
5. Shop details (SCR-A06): photos (optional), full price list, weekly hours, services, rating summary, Directions (opens https://www.google.com/maps/dir/?api=1&destination=lat,lng), primary Order here → the order flow in step 31.
6. Performance: list interactive < 1.5 s on 4G after location; skeleton cards while loading.
7. Tests: filter and sort reducers; Playwright with MSW: denied location → station picker → list; greyed reasons; details → Order here navigation.

Rules: mobile-first; icons + text for all states. Show the component tree first.
```

## Acceptance

- [ ] Location prompt only appears inside Print nearby.
- [ ] Greyed shops always show the reason.
- [ ] Works fully without GPS via area picker.

## Commit

`feat(nearby-ui): nearby list, map, filters, empty states and shop details`
