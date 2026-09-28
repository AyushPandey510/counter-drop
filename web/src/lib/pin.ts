// Mirrors api/internal/domain/staff.go WeakPIN so people get instant feedback; the server still decides.
export function weakPin(pin: string): boolean {
  if (!/^\d{4}$/.test(pin)) return true
  if (pin.split('').every((d) => d === pin[0])) return true
  const c = pin.split('').map(Number)
  const up = c.every((d, i) => i === 0 || d === (c[i - 1] ?? NaN) + 1)
  const down = c.every((d, i) => i === 0 || d === (c[i - 1] ?? NaN) - 1)
  if (up || down) return true
  if (pin.slice(0, 2) === pin.slice(2)) return true
  return ['2580', '0852', '1004', '2000', '1122', '6969', '1010'].includes(pin)
}

export const WEAK_PIN_MSG = 'Too easy to guess. Avoid 1234, 1111, 1212 and similar.'

/** Text for sending a setup link to one person (WhatsApp, SMS, copy). */
export function setupMessage(o: { person: string; shopName: string; url: string; expiresAt: string; reset: boolean }): string {
  const exp = new Date(o.expiresAt).toLocaleString('en-IN', { day: 'numeric', month: 'short', hour: 'numeric', minute: '2-digit', hour12: true, timeZone: 'Asia/Kolkata' })
  const verb = o.reset ? 'choose a new' : 'set up your'
  return `Hi ${o.person}, open this link on the phone or computer you'll use at the counter to ${verb} Counter Drop PIN for ${o.shopName}:\n${o.url}\nIt works once and expires on ${exp}. Don't share it.`
}
