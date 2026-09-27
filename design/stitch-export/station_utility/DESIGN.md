---
name: Station Utility
colors:
  surface: '#f8f9ff'
  surface-dim: '#cbdbf5'
  surface-bright: '#f8f9ff'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#eff4ff'
  surface-container: '#e5eeff'
  surface-container-high: '#dce9ff'
  surface-container-highest: '#d3e4fe'
  on-surface: '#0b1c30'
  on-surface-variant: '#45464d'
  inverse-surface: '#213145'
  inverse-on-surface: '#eaf1ff'
  outline: '#76777d'
  outline-variant: '#c6c6cd'
  surface-tint: '#565e74'
  primary: '#000000'
  on-primary: '#ffffff'
  primary-container: '#131b2e'
  on-primary-container: '#7c839b'
  inverse-primary: '#bec6e0'
  secondary: '#0051d5'
  on-secondary: '#ffffff'
  secondary-container: '#316bf3'
  on-secondary-container: '#fefcff'
  tertiary: '#000000'
  on-tertiary: '#ffffff'
  tertiary-container: '#00210a'
  on-tertiary-container: '#339650'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#dae2fd'
  primary-fixed-dim: '#bec6e0'
  on-primary-fixed: '#131b2e'
  on-primary-fixed-variant: '#3f465c'
  secondary-fixed: '#dbe1ff'
  secondary-fixed-dim: '#b4c5ff'
  on-secondary-fixed: '#00174b'
  on-secondary-fixed-variant: '#003ea8'
  tertiary-fixed: '#95f8a7'
  tertiary-fixed-dim: '#79db8d'
  on-tertiary-fixed: '#00210a'
  on-tertiary-fixed-variant: '#005323'
  background: '#f8f9ff'
  on-background: '#0b1c30'
  surface-variant: '#d3e4fe'
typography:
  token-display:
    fontFamily: JetBrains Mono
    fontSize: 64px
    fontWeight: '700'
    lineHeight: 72px
    letterSpacing: -0.02em
  token-display-mobile:
    fontFamily: JetBrains Mono
    fontSize: 48px
    fontWeight: '700'
    lineHeight: 56px
    letterSpacing: -0.01em
  headline-xl:
    fontFamily: Noto Sans
    fontSize: 36px
    fontWeight: '700'
    lineHeight: 44px
    letterSpacing: -0.02em
  headline-xl-mobile:
    fontFamily: Noto Sans
    fontSize: 28px
    fontWeight: '700'
    lineHeight: 36px
    letterSpacing: -0.01em
  headline-lg:
    fontFamily: Noto Sans
    fontSize: 24px
    fontWeight: '700'
    lineHeight: 32px
  headline-md:
    fontFamily: Noto Sans
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
  headline-sm:
    fontFamily: Noto Sans
    fontSize: 18px
    fontWeight: '600'
    lineHeight: 24px
  body-lg:
    fontFamily: Noto Sans
    fontSize: 16px
    fontWeight: '500'
    lineHeight: 24px
  body-md:
    fontFamily: Noto Sans
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
  body-sm:
    fontFamily: Noto Sans
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 16px
  label-lg:
    fontFamily: Noto Sans
    fontSize: 14px
    fontWeight: '600'
    lineHeight: 20px
    letterSpacing: 0.01em
  label-md:
    fontFamily: Noto Sans
    fontSize: 12px
    fontWeight: '600'
    lineHeight: 16px
    letterSpacing: 0.02em
  code-sm:
    fontFamily: JetBrains Mono
    fontSize: 12px
    fontWeight: '500'
    lineHeight: 16px
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  gutter: 1rem
  margin: 1rem
  space-xs: 0.25rem
  space-sm: 0.5rem
  space-md: 1rem
  space-lg: 1.5rem
  space-xl: 2.5rem
---

## Brand & Style

This design system is built for mission-critical, high-stress transit environments where seconds matter. In bustling railway hubs—such as Dadar, CSMT, or Thane—commuters juggle ticketing queues, packed platforms, and imminent departures. The digital interface must function with the immediate clarity of physical wayfinding signage: unmistakable, high-contrast, robust, and zero-latency.

The visual ethos blends **Utilitarian Functionalism** with **Civic Wayfinding**:
- **Absolute Solids:** Strictly no gradients, no decorative blurs, and no frosted glass surfaces. Every element relies on solid color fills and crisp geometric edges.
- **High-Velocity Scanning:** Information hierarchy prioritizes token numbers, pickup rack codes, and print readiness over brand embellishments.
- **Multilingual Clarity:** Seamless legibility across English, Hindi (Devanagari), and Marathi, accommodating dense glyph clusters and variable vowel mark heights without clipping or line-height jitter.
- **Tactile Trust:** Heavy touch affordances with distinct borders, bold states, and immediate feedback designed for single-handed mobile operation in direct sunlight or crowded transit corridors.

## Colors

The palette is engineered for maximum perceptual contrast in harsh sunlight and dim subterranean concourses, exceeding WCAG 2.1 AA and targeting AAA compliance across core interactive pathways.

- **Primary Canvas & Text (`#0F172A` / `#334155`):** Deep navy-charcoal acts as the primary structural anchor. It provides crisp terminal contrast on text and high-priority counters.
- **Action Cobalt (`#2563EB`):** A vibrant, unadulterated blue dedicated strictly to primary calls-to-action, active selection outlines, and file upload progress states.
- **Verification Emerald (`#15803D`):** Dedicated to the "Ready for Pickup" status, payment success confirmations, and machine dispense triggers. Paired with a solid light tint (`#DCFCE7`) for high-visibility badges.
- **Caution Amber (`#B45309`):** Reserved for paper jams, low ink warnings, missing print attributes, and token queue delays. Paired with `#FEF3C7` backgrounds.
- **Urgent Crimson (`#DC2626`):** Dedicated to file corruption alerts, failed transactions, and critical machine-offline notices.
- **Structural Neutrals:** Surface layers use crisp paper whites (`#FFFFFF`) framed against an ultra-clean base (`#F8FAFC`). Borders use calibrated technical greys (`#E2E8F0` and `#CBD5E1`) for hard structural division without visual noise.

## Typography

The type system prioritizes universal multilingual rendering across English, Hindi, and Marathi via **Noto Sans**. It ensures parity in x-height, stroke weight, and optical density across script boundaries, preventing truncation in layout headers when toggling language modes.

- **Token Display (`JetBrains Mono`):** Pickup codes, kiosk bay numbers, and numeric pins use a monospaced tabular font with slashed zeros and distinct glyphs to eradicate confusion between `0` and `O`, or `1` and `I`.
- **Vertical Metrics & Devanagari:** Line heights across body text are set generously (minimum 1.45×) to ensure diacritic marks (matras, anusvara, and nuktas) render without clipping against parent container borders.
- **Numeric Clarity:** Prices, page counts, and queue countdown timers are rendered in semi-bold and bold weights with tabular lining figures for immediate column alignment.

## Layout & Spacing

The layout is anchored on a rigid 8px baseline grid designed to streamline mobile checkouts and kiosk screens.

- **Mobile Viewport (up to 640px):** Single-column stack with `1rem` outer canvas padding. Action areas are locked to the lower 35% of the viewport (thumb-reach zone) to allow quick, one-handed operation on commuter platforms.
- **Tablet / Counter Kiosk (641px to 1024px):** 6-column grid with `1rem` gutters. Splits document configuration (pages, color modes, binding) on the left from the live pickup summary and token status on the right.
- **Station Admin / Large Screens (1025px+):** 12-column grid capped at 1280px maximum container width with `1.5rem` gutters.
- **Touch Target Integrity:** Every clickable element maintains a strict minimum bounding box of 48×48px, with interactive row items separated by at least `0.5rem` to prevent mis-taps during transit movement.

## Elevation & Depth

This design system avoids soft, atmospheric drop shadows, which become invisible under direct outdoor sunlight and wash out on low-spec counter displays. Visual hierarchy is established via **High-Contrast Structural Borders and Solid Tonal Stacking**:

- **Level 0 (Base Canvas):** Solid `#F8FAFC`.
- **Level 1 (Cards, Modules, Input Fields):** Solid `#FFFFFF` enclosed in a crisp `1.5px solid #CBD5E1` border.
- **Level 2 (Active Selections, Popovers):** Solid `#FFFFFF` with a `2px solid #2563EB` border.
- **Level 3 (Emergency Drawers & Overlays):** Solid `#FFFFFF` backed by an opaque modal scrim of `rgba(15, 23, 42, 0.72)`. Modals use a `2px solid #0F172A` outline and a hard, unblurred drop-offset: `box-shadow: 4px 4px 0px 0px #0F172A`.

## Shapes

Shapes communicate industrial sturdiness and reliability. Corner geometry is kept deliberately compact to maintain structured, rectangular alignment across dense tabular receipts and token slips:

- **Compact Softness:** Standard components utilize a controlled `0.25rem` (4px) radius. This softens edges just enough to differentiate software containers from physical hardware without wasting valuable layout space.
- **Badges and Pills:** Secondary status pills (e.g., "A4 B&W", "Double-Sided") use a strict `0.25rem` or zero-radius frame; rounded pill silhouettes are avoided to keep badges looking technical rather than playful.
- **Framing Lines:** Stroke widths for dividers and structural borders are locked to `1px` or `1.5px`, maintaining razor-sharp rendering on standard DPI kiosk monitors.

## Components

### Buttons
- **Primary Action (Execute Print / Pay):** Minimum height 52px. Solid background `#2563EB`, text `#FFFFFF`, font weight 600. Active states drop background to `#1D4ED8`. Active keyboard/touch focus triggers a `2px solid #0F172A` ring offset by 2px.
- **Station Ready Action:** Used for "Verify & Dispense" actions. Solid background `#15803D`, text `#FFFFFF`.
- **Secondary Utility:** Solid white `#FFFFFF` surface with a `1.5px solid #0F172A` outline and `#0F172A` text.
- **Destructive Action:** Solid white `#FFFFFF` surface with a `1.5px solid #DC2626` outline and `#DC2626` text.

### Token & Status Display
- **Token Counter Card:** Heavy-duty verification container. High-contrast white box framed in a `2px solid #0F172A` border. Displays the terminal pickup token (e.g., `DDR-0492`) rendered in `JetBrains Mono` at `64px` weight 700 with a persistent solid status strip (e.g., `#15803D` for "READY ON COUNTER 2").
- **Language Switcher:** Persistent 48px segmented controller locked to the top bar. High-contrast toggle between `ENG`, `हिं`, and `मरा` with instant zero-reload UI string updates.

### Input Fields & Selectors
- **File & Specification Inputs:** Height 48px, background `#FFFFFF`, border `1.5px solid #CBD5E1`, text `#0F172A`. 
- **Focus State:** Border color switches to `#2563EB` with an immediate `2px solid #2563EB` outline ring.
- **Error State:** Border switches to `#DC2626` with red validation helper text displayed directly below in `12px` font weight 600.

### Checkboxes & Segmented Radio Options
- **Page Layout & Copies Selectors:** Sized at 24×24px with high-contrast check marks. For print option tiles (e.g., "Color vs B&W", "Single vs Duplex"), utilize segmented card selectors that turn `#F8FAFC` into a solid `#2563EB` border with a subtle check icon indicator.

### Lists & Queue Summaries
- **Print Queue Rows:** Clean alternating or cleanly bordered item strips separated by `1px solid #E2E8F0`. Page numbers, copies, and costs are right-aligned using tabular numbers for rapid cognitive scanning before tap-to-pay.