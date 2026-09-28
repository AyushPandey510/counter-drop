const rupee = new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 2, minimumFractionDigits: 0 })

/** 2300 → "₹23", 250 → "₹2.50", Indian digit grouping. */
export function rupees(paise: number): string {
  return rupee.format(paise / 100)
}

const time = new Intl.DateTimeFormat('en-IN', { hour: 'numeric', minute: '2-digit', hour12: true, timeZone: 'Asia/Kolkata' })

/** ISO → "6:40 pm" in IST. */
export function clock(iso?: string): string {
  if (!iso) return ''
  return time.format(new Date(iso)).replace('AM', 'am').replace('PM', 'pm')
}

/** "09:00" → "9:00 am". */
export function hhmm(v: string): string {
  const [h = 0, m = 0] = v.split(':').map(Number)
  const suffix = h >= 12 ? 'pm' : 'am'
  const h12 = h % 12 === 0 ? 12 : h % 12
  return `${h12}:${String(m).padStart(2, '0')} ${suffix}`
}

export function mmss(ms: number): string {
  const s = Math.max(0, Math.floor(ms / 1000))
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
}

export function minutesSince(iso: string | undefined, now: number): number {
  if (!iso) return 0
  return Math.max(0, Math.floor((now - new Date(iso).getTime()) / 60000))
}

export function bytes(n: number): string {
  if (n < 1024 * 1024) return `${Math.max(1, Math.round(n / 1024))} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

const dayTimeFmt = new Intl.DateTimeFormat('en-IN', { weekday: 'short', day: 'numeric', month: 'short', hour: 'numeric', minute: '2-digit', hour12: true, timeZone: 'Asia/Kolkata' })

/** ISO → "Sat, 3 Oct, 6:40 pm" in IST. */
export function dayTime(iso?: string): string {
  if (!iso) return ''
  return dayTimeFmt.format(new Date(iso)).replace('AM', 'am').replace('PM', 'pm')
}
