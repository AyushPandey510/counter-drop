/** Tokens from design/DESIGN.md §2 and §9. */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        ink: { DEFAULT: 'var(--ink)', muted: 'var(--ink-muted)', subtle: 'var(--ink-subtle)' },
        canvas: 'var(--canvas)',
        surface: { DEFAULT: 'var(--surface)', tint: 'var(--surface-tint)' },
        line: 'var(--border)',
        divider: 'var(--divider)',
        action: { DEFAULT: 'var(--action)', pressed: 'var(--action-pressed)' },
        ready: { DEFAULT: 'var(--ready)', bg: 'var(--ready-bg)' },
        attention: { DEFAULT: 'var(--attention)', bg: 'var(--attention-bg)' },
        danger: { DEFAULT: 'var(--danger)', bg: 'var(--danger-bg)' },
      },
      fontFamily: {
        sans: ['"Noto Sans"', '"Noto Sans Devanagari"', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'monospace'],
      },
      borderRadius: { DEFAULT: '4px', sm: '2px', md: '6px', lg: '8px' },
      boxShadow: { modal: '4px 4px 0 0 #0F172A' },
    },
  },
  plugins: [],
}
