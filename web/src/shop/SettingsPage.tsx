import { useEffect, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft, LogOut } from 'lucide-react'
import { ApiError } from '@/lib/api'
import { rupees } from '@/lib/format'
import type { PriceList, Shop } from '@/lib/types'
import { Banner, Button, Card, Spinner } from '@/components/ui'
import { useShopAuth, useShopTheme } from './auth'
import { StaffSection } from './StaffSection'

type Form = {
  name: string
  address: string
  opensAt: string
  closesAt: string
  bwOne: string
  bwBoth: string
  colourOne: string
  colourBoth: string
  minCharge: string
}

const toRs = (p: number) => (p ? String(p / 100) : '')
const toPaise = (v: string) => Math.round((Number(v.trim() || '0') || 0) * 100)

function fromShop(s: Shop): Form {
  return {
    name: s.name,
    address: s.address,
    opensAt: s.opensAt,
    closesAt: s.closesAt,
    bwOne: toRs(s.prices.bwOnePaise),
    bwBoth: toRs(s.prices.bwBothPaise),
    colourOne: toRs(s.prices.colourOnePaise),
    colourBoth: toRs(s.prices.colourBothPaise),
    minCharge: toRs(s.prices.minChargePaise),
  }
}

// Owner settings (FSD SCR-S07, R1a subset): shop profile, hours and the price list.
export default function SettingsPage() {
  useShopTheme()
  const { session, call, setShop, logout } = useShopAuth()
  const [form, setForm] = useState<Form | null>(null)
  const [msg, setMsg] = useState<{ tone: 'ready' | 'danger'; text: string } | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    call<Shop>('/shop/settings').then((s) => setForm(fromShop(s)))
  }, [call])

  if (!form || !session) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Spinner className="h-8 w-8" />
      </div>
    )
  }

  const set = (k: keyof Form) => (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [k]: e.target.value })

  const save = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setMsg(null)
    const prices: PriceList = {
      bwOnePaise: toPaise(form.bwOne),
      bwBothPaise: toPaise(form.bwBoth),
      colourOnePaise: toPaise(form.colourOne),
      colourBothPaise: toPaise(form.colourBoth),
      minChargePaise: toPaise(form.minCharge),
      version: session.shop.prices.version,
    }
    try {
      const s = await call<Shop>('/shop/settings', {
        method: 'PUT',
        body: { profile: { name: form.name.trim(), address: form.address.trim(), opensAt: form.opensAt, closesAt: form.closesAt }, prices },
      })
      setShop(s)
      setForm(fromShop(s))
      setMsg({ tone: 'ready', text: 'Saved. New jobs use these prices; jobs already sent keep their price.' })
    } catch (err) {
      setMsg({ tone: 'danger', text: err instanceof ApiError ? err.message : 'Could not save' })
    } finally {
      setBusy(false)
    }
  }

  const input = 'mt-1 h-11 w-full rounded border-[1.5px] border-line bg-surface px-3'
  const bwOne = toPaise(form.bwOne)

  return (
    <div className="mx-auto max-w-2xl space-y-4 px-4 py-6 lg:max-w-6xl lg:py-8">
      <div className="flex items-center gap-3">
        <Link to="/shop" className="rounded p-2 hover:bg-surface-tint" aria-label="Back to queue">
          <ArrowLeft className="h-5 w-5" aria-hidden />
        </Link>
        <h1 className="flex-1 text-2xl font-bold">Shop settings</h1>
        <Button variant="ghost" size="sm" onClick={logout}>
          <LogOut className="h-4 w-4" aria-hidden /> Sign out
        </Button>
      </div>
      {msg && <Banner tone={msg.tone}>{msg.text}</Banner>}

      <div className="space-y-4 lg:grid lg:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)] lg:items-start lg:gap-6 lg:space-y-0">
        <form onSubmit={save} className="space-y-4">
          <Card className="space-y-3 p-4">
            <h2 className="text-lg font-bold">Shop</h2>
            <Field label="Shop name">
              <input className={input} value={form.name} onChange={set('name')} required minLength={3} maxLength={60} />
            </Field>
            <Field label="Address (shown to customers)">
              <input className={input} value={form.address} onChange={set('address')} maxLength={160} />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field label="Opens at">
                <input type="time" className={input} value={form.opensAt} onChange={set('opensAt')} required />
              </Field>
              <Field label="Closes at">
                <input type="time" className={input} value={form.closesAt} onChange={set('closesAt')} required />
              </Field>
            </div>
            <p className="text-xs text-ink-muted">
              Link name: <span className="font-mono">{session.shop.slug}</span> (fixed — it's printed on your QR poster)
            </p>
          </Card>

          <Card className="space-y-3 p-4">
            <h2 className="text-lg font-bold">Prices (₹)</h2>
            <div className="grid grid-cols-2 gap-3">
              <Field label="B/W · per side">
                <input inputMode="decimal" className={`${input} font-mono`} value={form.bwOne} onChange={set('bwOne')} required />
              </Field>
              <Field label="B/W · both sides, per sheet" hint={`Blank = ${rupees(bwOne * 2)}`}>
                <input inputMode="decimal" className={`${input} font-mono`} value={form.bwBoth} onChange={set('bwBoth')} />
              </Field>
              <Field label="Colour · per side" hint="Blank = no colour printing">
                <input inputMode="decimal" className={`${input} font-mono`} value={form.colourOne} onChange={set('colourOne')} />
              </Field>
              <Field label="Colour · both sides, per sheet" hint="Blank = 2 × per side">
                <input inputMode="decimal" className={`${input} font-mono`} value={form.colourBoth} onChange={set('colourBoth')} />
              </Field>
              <Field label="Minimum charge per job">
                <input inputMode="decimal" className={`${input} font-mono`} value={form.minCharge} onChange={set('minCharge')} />
              </Field>
            </div>
            <p className="text-xs text-ink-muted">Totals are rounded to the nearest rupee. Customers see these prices before they send.</p>
          </Card>

          <Button type="submit" size="lg" className="w-full" disabled={busy}>
            {busy ? 'Saving…' : 'Save settings'}
          </Button>
        </form>

        <div className="lg:sticky lg:top-6">
          <StaffSection />
        </div>
      </div>
    </div>
  )
}

function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <label className="block text-sm font-semibold">
      {label}
      {children}
      {hint && <span className="mt-1 block text-xs font-normal text-ink-muted">{hint}</span>}
    </label>
  )
}
