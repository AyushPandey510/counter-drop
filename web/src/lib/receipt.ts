// Receipt as a PNG the customer can save. Drawn on a canvas with the app's own fonts, so Hindi and
// Marathi render the same on every phone; no PDF library to download.

export interface ReceiptData {
  title: string // "Receipt"
  shopName: string
  shopAddress?: string
  token: string
  when: string // collected date and time
  files: { name: string; detail: string }[]
  rows: { label: string; value: string }[] // amount lines before the total
  total: { label: string; value: string }
  paid: { label: string; value: string }
  notes: { label: string; value: string }[] // deletion proof
  footer: string
}

const SANS = '"Noto Sans", "Noto Sans Devanagari", system-ui, sans-serif'
const MONO = '"JetBrains Mono", ui-monospace, monospace'
const W = 600 // CSS pixels; drawn at 2× for sharp text
const PAD = 32

export async function receiptBlob(d: ReceiptData): Promise<Blob> {
  try {
    await document.fonts?.ready
  } catch {
    /* fonts API not available: system fonts are fine */
  }
  // Two passes: measure the height, then draw.
  const height = draw(null, d)
  const canvas = document.createElement('canvas')
  canvas.width = W * 2
  canvas.height = height * 2
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('canvas not available')
  ctx.scale(2, 2)
  draw(ctx, d)
  return new Promise((resolve, reject) => canvas.toBlob((b) => (b ? resolve(b) : reject(new Error('could not create image'))), 'image/png'))
}

export async function downloadReceipt(d: ReceiptData, filename: string) {
  const blob = await receiptBlob(d)
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  window.setTimeout(() => URL.revokeObjectURL(url), 10_000)
}

/** Draws the receipt (or only measures it when ctx is null) and returns its height. */
function draw(ctx: CanvasRenderingContext2D | null, d: ReceiptData): number {
  const m = ctx ?? measureCtx()
  const ink = '#0F172A'
  const muted = '#475569'
  const line = '#CBD5E1'
  let y = PAD

  const text = (s: string, x: number, size: number, opts: { bold?: boolean; mono?: boolean; colour?: string; align?: CanvasTextAlign; max?: number } = {}) => {
    m.font = `${opts.bold ? 700 : 400} ${size}px ${opts.mono ? MONO : SANS}`
    m.textAlign = opts.align ?? 'left'
    m.fillStyle = opts.colour ?? ink
    const v = opts.max ? fit(m, s, opts.max) : s
    if (ctx) ctx.fillText(v, x, y)
  }
  const rule = () => {
    if (ctx) {
      ctx.strokeStyle = line
      ctx.lineWidth = 1
      ctx.beginPath()
      ctx.moveTo(PAD, y)
      ctx.lineTo(W - PAD, y)
      ctx.stroke()
    }
  }
  const pair = (label: string, value: string, size = 16, bold = false) => {
    y += size + 10
    text(label, PAD, size, { colour: muted, max: W - 2 * PAD - 200 })
    text(value, W - PAD, size, { bold, align: 'right', max: 260 })
  }

  if (ctx) {
    ctx.fillStyle = '#FFFFFF'
    ctx.fillRect(0, 0, W, 10_000)
  }
  y += 22
  text('Counter Drop', PAD, 18, { bold: true })
  text(d.title, W - PAD, 18, { bold: true, align: 'right', colour: muted })
  y += 34
  text(d.shopName, PAD, 24, { bold: true, max: W - 2 * PAD })
  if (d.shopAddress) {
    y += 22
    text(d.shopAddress, PAD, 14, { colour: muted, max: W - 2 * PAD })
  }
  y += 50
  text(d.token, PAD, 44, { bold: true, mono: true })
  text(d.when, W - PAD, 16, { align: 'right', colour: muted })
  y += 20
  rule()

  for (const f of d.files) {
    y += 28
    text(f.name, PAD, 16, { bold: true, max: W - 2 * PAD - 190 })
    text(f.detail, W - PAD, 14, { align: 'right', colour: muted, max: 180 })
  }
  y += 18
  rule()
  for (const r of d.rows) pair(r.label, r.value)
  pair(d.total.label, d.total.value, 22, true)
  pair(d.paid.label, d.paid.value)
  y += 18
  rule()
  // Deletion proof: label above, value wrapped below (these sentences can be long).
  for (const n of d.notes) {
    y += 26
    text(n.label, PAD, 13, { colour: muted })
    m.font = `400 15px ${SANS}`
    for (const l of wrap(m, n.value, W - 2 * PAD)) {
      y += 21
      text(l, PAD, 15)
    }
  }
  y += 34
  text(d.footer, W / 2, 12, { align: 'center', colour: muted, max: W - 2 * PAD })
  return y + PAD
}

let measurer: CanvasRenderingContext2D | null = null
function measureCtx(): CanvasRenderingContext2D {
  if (!measurer) measurer = document.createElement('canvas').getContext('2d')!
  return measurer
}

/** Shortens text with an ellipsis to fit a width. */
function fit(ctx: CanvasRenderingContext2D, s: string, max: number): string {
  if (ctx.measureText(s).width <= max) return s
  const chars = Array.from(s)
  while (chars.length > 1 && ctx.measureText(chars.join('') + '…').width > max) chars.pop()
  return chars.join('') + '…'
}

/** Splits text into lines that fit a width (breaks on spaces; very long words are cut). */
function wrap(ctx: CanvasRenderingContext2D, s: string, max: number): string[] {
  const out: string[] = []
  let cur = ''
  for (const w of s.split(/\s+/)) {
    const next = cur ? `${cur} ${w}` : w
    if (ctx.measureText(next).width <= max || !cur) cur = next
    else {
      out.push(cur)
      cur = w
    }
  }
  if (cur) out.push(cur)
  return out.map((l) => fit(ctx, l, max))
}
