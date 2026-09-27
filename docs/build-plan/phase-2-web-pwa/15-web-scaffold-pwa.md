# 15 — Web scaffold: Vite, React, PWA, API client, design tokens

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 2 Web PWA | R1a | M | 03 | FSD §1 components, §3 screens/routes, §5.0 PWA technical spec, NFR-2, NFR-16–18, FS-17.9, FS-17.14, design/DESIGN.md |

## Goal

A production-grade React + TypeScript PWA skeleton with routing for all four surfaces (customer, shop, TV, admin), typed API client from the OpenAPI contract, mock server for parallel work, design tokens with light and dark themes, and CI.

## Stack

Vite · React 19 · TypeScript (strict) · React Router · TanStack Query · Zod · Tailwind CSS v4 with CSS variables · vite-plugin-pwa (Workbox) · openapi-fetch with generated types · MSW (mock API) · Vitest + Testing Library · Playwright · pnpm.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §1 (components), §3 (screen inventory and routes), §5.0 (PWA technical spec), §17 (localisation fonts, accessibility, CSP) and NFR-2 and NFR-16–18 in docs/BRD.md §15. Read contracts/openapi.yaml and web/README.md.

Task: scaffold the web app in web/.

1. pnpm + Vite + React + TypeScript strict (noUncheckedIndexedAccess on). ESLint (typescript-eslint strict, react-hooks, jsx-a11y) + Prettier. Path alias @/ → src/.
2. Folder structure:
   src/app (router, providers, error boundary)
   src/customer (walk-in and Print nearby screens)
   src/shop (dashboard)
   src/tv, src/admin (placeholders)
   src/components/ui (design system primitives)
   src/lib/api (generated schema.d.ts, client.ts, queries/)
   src/lib/ws (WebSocket client, step 18)
   src/lib/i18n (step 16)
   src/lib/pwa (install prompt, update toast)
   src/styles (tokens.css, globals.css)
   src/test (msw handlers, fixtures)
3. Routes (lazy-loaded per surface to keep the customer bundle small):
   /                  → customer Home (SCR-A00, placeholder)
   /s/:slug           → drop page (SCR-W01…W03, placeholder)
   /t/:jobId          → ticket (SCR-W04/W05, placeholder)
   /scan              → in-app scanner (SCR-A17, placeholder)
   /nearby            → Print nearby (R1b, placeholder behind a feature flag)
   /shop/*            → shop dashboard (login, board, settings)
   /tv/:slug          → TV (R2 placeholder)
   /admin/*           → admin (step 23)
   Budget: the /s/:slug route chunk + shared runtime ≤ 200 KB gzip (NFR-2). Add a bundle-size check script (vite build --report or rollup-plugin-visualizer + a size-limit config) and run it in CI.
4. API client: openapi-fetch typed by src/lib/api/schema.d.ts (make contract-types). A single client with base URL from VITE_API_BASE_URL, request ID header, X-Ticket-Secret injection for guest calls, Bearer token for shop calls, and unwrapping of {data} / {error:{code,message}} into a typed ApiError with code. TanStack Query defaults: retry 2 for network errors only, no retry on 4xx.
5. Mock API: MSW handlers for API-01, 02, 05, 06, 07, 08, 09 and 30 with realistic fixtures (demo-print shop, prices from migration seed) so UI steps can proceed without the backend. VITE_USE_MOCKS=true enables them in dev.
6. PWA (vite-plugin-pwa): manifest (name "Counter Drop", short_name "Counter Drop", display standalone, start_url "/?src=pwa", scope "/", theme and background colors from tokens, icons 192/512/maskable — generate placeholder icons), share_target stub (step 35 fills it), handle_links "preferred". Workbox: precache the app shell; runtime network-first for /api (no caching of POST, never cache signed URLs or uploads); an "update available" toast.
7. Design tokens from design/DESIGN.md (read it fully, including §8 content rules): CSS variables on :root (light, default for every surface) and [data-theme="dark"] (shop dashboard and shop mode only, owner setting), mapped into Tailwind with the theme snippet in DESIGN.md §9. Status colours follow DESIGN.md §2 (In line ink, Printing action blue, Ready green, Waiting amber, Cancelled red), always with icon + text. 4 px radius, 1.5 px borders, no soft shadows (modal uses the hard offset shadow).
   Components (match design/screens/*/screen.png): Button (primary 52 px, ready, secondary, destructive), Card, Chip (4 px), SegmentedTiles with price hints, Stepper, TokenCard (dark ink panel, JetBrains Mono 72 px), StatusStepper (3 steps), FileRow (with progress and error states), PriceBreakdown, LanguageSwitcher (EN · हिं · मरा), InstallBanner, DeletionReceipt, BottomNav (Print nearby · Scan · My jobs), BoardCard, CountdownBanner, Toast, Sheet, EmptyState. Each with a Vitest + Testing Library test and jsx-a11y clean. Build a /dev/ui page that renders every component in both themes for visual review.
8. Fonts: self-host Noto Sans + Noto Sans Devanagari subsets (woff2) and JetBrains Mono for shop numerals; font-display swap.
9. Security: a CSP meta for dev and a documented header set for prod (step 25): default-src 'self'; connect-src API and R2 origins; img-src self blob: data:; frame-src Razorpay only on /nearby pay routes (R1b).
10. Makefile targets: web-install, web-dev, web-test, web-build, web-lint; add them to make test and make lint; enable the web job in CI (pnpm cache, lint, typecheck, test, build, bundle size check).

Rules: no UI beyond placeholders and primitives in this step. Show the folder tree and dependency list first.
```

## Acceptance

- [ ] `pnpm dev` with `VITE_USE_MOCKS=true` renders every route placeholder.
- [ ] Lighthouse PWA installable check passes on a production build.
- [ ] Bundle budget check passes in CI.
- [ ] Primitives pass tests and jsx-a11y.

## Verify

```bash
make web-install web-lint web-test web-build
cd web && pnpm preview   # open, check "Install" is offered in Chrome
```

## Commit

`feat(web): Vite React TS PWA scaffold, typed API client, mocks, design tokens and CI`
