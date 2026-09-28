import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ArrowLeft, KeyRound, LogOut, Settings } from 'lucide-react'
import { ApiError } from '@/lib/api'
import { weakPin, WEAK_PIN_MSG } from '@/lib/pin'
import { Banner, Button, Card, Chip } from '@/components/ui'
import { useShopAuth, useShopTheme } from './auth'

// Everyone's own account: change PIN, sign out (FSD SCR-S02).
export default function AccountPage() {
  useShopTheme()
  const { session, call, logout } = useShopAuth()
  const navigate = useNavigate()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [again, setAgain] = useState('')
  const [msg, setMsg] = useState<{ tone: 'ready' | 'danger'; text: string } | null>(null)
  const [busy, setBusy] = useState(false)
  if (!session) return null
  const { staff, shop } = session

  const digits = (set: (v: string) => void) => (e: React.ChangeEvent<HTMLInputElement>) => set(e.target.value.replace(/\D/g, '').slice(0, 4))

  const save = async (e: React.FormEvent) => {
    e.preventDefault()
    setMsg(null)
    if (next !== again) return setMsg({ tone: 'danger', text: "The new PINs don't match." })
    if (weakPin(next)) return setMsg({ tone: 'danger', text: WEAK_PIN_MSG })
    setBusy(true)
    try {
      await call('/shop/me/pin', { method: 'PUT', body: { currentPin: current, newPin: next } })
      setCurrent('')
      setNext('')
      setAgain('')
      setMsg({ tone: 'ready', text: 'PIN changed. Other devices signed in as you have been signed out.' })
    } catch (err) {
      setMsg({ tone: 'danger', text: err instanceof ApiError ? err.message : 'Could not change PIN' })
    } finally {
      setBusy(false)
    }
  }

  const input = 'mt-1 h-12 w-full rounded border-[1.5px] border-line bg-surface px-3 font-mono text-xl tracking-[0.5em]'

  return (
    <div className="mx-auto max-w-md space-y-4 px-4 py-6 lg:max-w-4xl lg:py-8">
      <div className="flex items-center gap-3">
        <Link to="/shop" className="rounded p-2 hover:bg-surface-tint" aria-label="Back to queue">
          <ArrowLeft className="h-5 w-5" aria-hidden />
        </Link>
        <h1 className="flex-1 text-2xl font-bold">My account</h1>
      </div>

      <div className="space-y-4 lg:grid lg:grid-cols-2 lg:items-start lg:gap-6 lg:space-y-0">
        <div className="space-y-4">
          <Card className="flex items-center gap-3 p-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-ink text-xl font-bold text-canvas">{staff.name.slice(0, 1).toUpperCase()}</div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <span className="text-lg font-bold">{staff.name}</span>
                <Chip tone={staff.role === 'owner' ? 'action' : 'neutral'}>{staff.role === 'owner' ? 'Owner' : 'Staff'}</Chip>
              </div>
              <div className="truncate text-sm text-ink-muted">
                {shop.name} · <span className="font-mono">{shop.slug}</span>
              </div>
            </div>
          </Card>

          <div className="flex gap-2 lg:flex-col">
            {staff.role === 'owner' && (
              <Button variant="secondary" className="flex-1" onClick={() => navigate('/shop/settings')}>
                <Settings className="h-4 w-4" aria-hidden /> Shop & staff
              </Button>
            )}
            <Button variant="secondary" className="flex-1" onClick={logout}>
              <LogOut className="h-4 w-4" aria-hidden /> Sign out
            </Button>
          </div>
        </div>
        <Card className="p-4">
          <h2 className="mb-3 flex items-center gap-2 text-lg font-bold">
            <KeyRound className="h-5 w-5" aria-hidden /> Change PIN
          </h2>
          {msg && (
            <div className="mb-3">
              <Banner tone={msg.tone}>{msg.text}</Banner>
            </div>
          )}
          <form onSubmit={save} className="space-y-3">
            <label className="block text-sm font-semibold">
              Current PIN
              <input type="password" inputMode="numeric" autoComplete="current-password" className={input} value={current} onChange={digits(setCurrent)} required minLength={4} maxLength={4} />
            </label>
            <label className="block text-sm font-semibold">
              New PIN
              <input type="password" inputMode="numeric" autoComplete="new-password" className={input} value={next} onChange={digits(setNext)} required minLength={4} maxLength={4} />
            </label>
            <label className="block text-sm font-semibold">
              New PIN again
              <input type="password" inputMode="numeric" autoComplete="new-password" className={input} value={again} onChange={digits(setAgain)} required minLength={4} maxLength={4} />
            </label>
            <Button type="submit" className="w-full" disabled={busy || current.length < 4 || next.length < 4 || again.length < 4}>
              {busy ? 'Saving…' : 'Change PIN'}
            </Button>
          </form>
          <p className="mt-3 text-xs text-ink-muted">Forgot your current PIN? Ask the shop owner to send you a new PIN link.</p>
        </Card>
      </div>
    </div>
  )
}
