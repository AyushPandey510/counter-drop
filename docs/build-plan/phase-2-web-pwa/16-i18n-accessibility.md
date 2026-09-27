# 16 — i18n (English, Hindi, Marathi) and accessibility baseline

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 2 Web PWA | R1a | S | 15 | FSD FS-17.11–17.17, §16 MSG catalogue, UX-C5–C9, CO-9, AT-24 |

## Goal

Every string comes from a translation catalogue in three languages with correct plurals and Indian number formatting, and the app meets WCAG 2.1 AA from the start.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md FS-17.11–17.17 and §16 (message catalogue MSG-C01…C19, MSG-S01…S05), and docs/BRD.md UX-C5–C9. Read web/src from step 15.

Task: add i18n and an accessibility baseline to the web app.

1. i18next + react-i18next + i18next-icu (ICU MessageFormat). Catalogues in web/src/lib/i18n/locales/{en,hi,mr}/{common,customer,shop,errors}.json. English is the master; Hindi and Marathi files contain the same keys (machine-drafted is OK for now; mark each with "_review": true at the top so a native speaker reviews before the pilot — BRD UX-C7).
2. Keys for everything in FSD §4 screens, §6 shop screens and every MSG-* text, plus error code → message mapping (errors.json keyed by API error code from the contract). ApiError from step 15 renders through t(`errors.${code}`) with a generic fallback.
3. Language detection order (FS-17.12): saved choice (localStorage, try/catch) → navigator.language (hi*, mr* → hi/mr) → en. A LanguageSwitcher component (EN / हिं / मरा) usable in the header.
4. Formatting helpers in src/lib/i18n/format.ts: formatRupees(paise) → "₹1,00,000" / "₹52" using Intl.NumberFormat('en-IN') with Western digits in all languages; formatTime (12-hour, IST, "6:40 pm"); formatRelativeMinutes ("about 12 min"); formatPages with ICU plurals.
5. A script `pnpm i18n:check` that fails CI if any key is missing in hi/mr, if a key is unused, or if a component contains a hard-coded user-visible string literal (use eslint-plugin-i18next with an allowlist for aria-hidden icons and test files).
6. Accessibility baseline:
   - Global focus-visible styles; skip-to-content link.
   - An aria-live="polite" region provider for status updates and an assertive one for the Ready screen (FS-17.16).
   - Status components always render icon + text (never colour alone).
   - Text scales to 200% without horizontal scroll at 360 px (FS-17.17): add a Playwright test at 360×800 with root font-size 200% on the /s/:slug placeholder.
   - @axe-core/playwright check on every route placeholder in CI.
7. Fonts: make sure Devanagari renders with Noto Sans Devanagari (from step 15) and line-height is comfortable (1.5).

Rules: no screen implementation here beyond wiring existing placeholders to t(). Show the key naming convention first (e.g. customer.drop.chooseFiles).
```

## Acceptance

- [ ] Switching language changes all placeholder text; choice persists.
- [ ] `pnpm i18n:check` and axe checks pass in CI.
- [ ] ₹ amounts use Indian grouping; times show "6:40 pm".

## Verify

```bash
cd web && pnpm i18n:check && pnpm test && pnpm exec playwright test a11y
```

## Commit

`feat(web): i18n for en/hi/mr with ICU, formatting helpers and accessibility baseline`
