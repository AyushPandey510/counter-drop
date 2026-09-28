import { useCallback, useEffect, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { ShieldCheck } from 'lucide-react'
import { api, ApiError } from '@/lib/api'
import { weakPin, WEAK_PIN_MSG } from '@/lib/pin'
import { clock } from '@/lib/format'
import type { SetupInfo, Shop, Staff } from '@/lib/types'
import { Banner, Card, Chip, Logo, Spinner } from '@/components/ui'
import { PinPad } from '@/components/PinPad'
import { useShopAuth, useShopTheme } from './auth'

const HOLD = 'cd.setup.token' // survives a reload of this tab only; cleared once used

// One-time setup link (/shop/setup#token): the person chooses their own PIN and is signed in on this device.
export default function SetupPage() {
  useShopTheme()
  const location = useLocation()
  const navigate = useNavigate()
  const { session, adopt } = useShopAuth()
  const [token] = useState(() => {
    const frag = location.hash.replace(/^#/, '')
    try {
      if (frag) sessionStorage.setItem(HOLD, frag)
      return frag || sessionStorage.getItem(HOLD) || ''
    } catch {
      return frag
    }
  })
  const [info, setInfo] = useState<SetupInfo | null>(null)
  const [invalid, setInvalid] = useState('')
  const [step, setStep] = useState<'choose' | 'confirm'>('choose')
  const [first, setFirst] = useState('')
  const [pin, setPin] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  // Keep the token out of browser history and screenshots of the address bar.
  useEffect(() => {
    if (location.hash) window.history.replaceState(null, '', location.pathname)
  }, [location])

  useEffect(() => {
    if (!token) {
      setInvalid('This page needs the full link you were sent. Open it again from the message.')
      return
    }
    api<SetupInfo>('/shop/setup/info', { body: { token } })
      .then(setInfo)
      .catch((e) => setInvalid(e instanceof ApiError ? e.message : 'Could not check this link. Check your internet and try again.'))
  }, [token])

  const submit = useCallback(
    async (chosen: string) => {
      setBusy(true)
      try {
        const res = await api<{ token: string; staff: Staff; shop: Shop }>('/shop/setup', { body: { token, pin: chosen } })
        try {
          sessionStorage.removeItem(HOLD)
        } catch {
          /* ignore */
        }
        adopt(res)
        navigate('/shop', { replace: true })
      } catch (e) {
        if (e instanceof ApiError && e.code === 'link_invalid') setInvalid(e.message)
        else setError(e instanceof ApiError ? e.message : 'Could not save. Try again.')
        setStep('choose')
        setFirst('')
        setPin('')
      } finally {
        setBusy(false)
      }
    },
    [token, adopt, navigate],
  )

  const press = useCallback(
    (d: string) => {
      if (busy) return
      const next = (pin + d).slice(0, 4)
      setPin(next)
      if (next.length < 4) return
      if (step === 'choose') {
        if (weakPin(next)) {
          setError(WEAK_PIN_MSG)
          setPin('')
          return
        }
        setError('')
        setFirst(next)
        setPin('')
        setStep('confirm')
      } else if (next !== first) {
        setError("The two PINs didn't match. Choose your PIN again.")
        setFirst('')
        setPin('')
        setStep('choose')
      } else {
        submit(next)
      }
    },
    [busy, pin, step, first, submit],
  )
  const del = useCallback(() => setPin((p) => p.slice(0, -1)), [])

  return (
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center gap-4 px-4 py-8">
      <div className="flex items-center gap-3">
        <Logo size={48} />
        <div>
          <h1 className="text-2xl font-bold">Counter Drop</h1>
          <p className="text-sm text-ink-muted">Set your PIN</p>
        </div>
      </div>

      {invalid ? (
        <Card className="space-y-3 p-5">
          <Banner tone="danger">{invalid}</Banner>
          <p className="text-sm text-ink-muted">Setup links work once and expire after 48 hours. Ask the shop owner (or Counter Drop support) to send you a new one.</p>
          <Link to="/shop/login" className="block text-sm font-semibold text-action underline">
            I already have a PIN — sign in
          </Link>
        </Card>
      ) : !info ? (
        <div className="flex justify-center py-16">
          <Spinner className="h-8 w-8" />
        </div>
      ) : (
        <Card className="p-4">
          <div className="mb-1 flex items-center gap-2">
            <h2 className="text-xl font-bold">{info.purpose === 'reset' ? `New PIN for ${info.staffName}` : `Welcome, ${info.staffName}`}</h2>
            <Chip tone={info.role === 'owner' ? 'action' : 'neutral'}>{info.role === 'owner' ? 'Owner' : 'Staff'}</Chip>
          </div>
          <p className="text-sm text-ink-muted">
            {info.shopName} · link valid until {clock(info.expiresAt)}, {new Date(info.expiresAt).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', timeZone: 'Asia/Kolkata' })}
          </p>
          {session && session.staff.name !== info.staffName && (
            <div className="mt-3">
              <Banner tone="attention">
                This device is signed in as {session.staff.name}. Finishing here signs it in as {info.staffName} instead.
              </Banner>
            </div>
          )}
          {error && (
            <div className="mt-3">
              <Banner tone="danger">{error}</Banner>
            </div>
          )}
          <p className="mt-4 font-semibold">{step === 'choose' ? 'Choose a 4-digit PIN' : 'Enter the same PIN again'}</p>
          <p className="text-xs text-ink-muted">You'll use it every day to sign in at the counter. Don't use 1234, 1111 or your birth year.</p>
          <PinPad value={pin} onDigit={press} onDelete={del} busy={busy} label={step === 'choose' ? 'New PIN' : 'Confirm PIN'} />
          <p className="mt-4 flex items-start gap-2 text-xs text-ink-muted">
            <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-ready" aria-hidden />
            Only you will know this PIN — not the shop owner, not Counter Drop. If you forget it, ask for a new link.
          </p>
        </Card>
      )}
    </div>
  )
}
