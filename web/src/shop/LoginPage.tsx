import { useEffect, useState } from 'react'
import { Navigate, useSearchParams } from 'react-router-dom'
import { Delete } from 'lucide-react'
import { api, ApiError, store } from '@/lib/api'
import { Banner, Button, Card, Logo, Spinner } from '@/components/ui'
import { useShopAuth, useShopTheme } from './auth'

// Staff sign-in: shop link name → name tile → 4-digit PIN (FSD SCR-S02, R1a without phone OTP).
export default function LoginPage() {
  useShopTheme()
  const { session, login } = useShopAuth()
  const [params] = useSearchParams()
  const [slug, setSlug] = useState(params.get('shop') ?? store.get('cd.shop.slug') ?? '')
  const [shop, setShop] = useState<{ name: string; slug: string } | null>(null)
  const [names, setNames] = useState<string[]>([])
  const [name, setName] = useState('')
  const [pin, setPin] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const findShop = async (s: string) => {
    setError('')
    try {
      const res = await api<{ shop: { name: string; slug: string }; names: string[] }>(`/shop/staff-names?shop=${encodeURIComponent(s.trim().toLowerCase())}`)
      setShop(res.shop)
      setNames(res.names)
    } catch (e) {
      setShop(null)
      setError(e instanceof ApiError && e.status === 404 ? 'No shop with that link name.' : 'Could not reach the server.')
    }
  }

  useEffect(() => {
    if (slug) findShop(slug)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const press = async (digit: string) => {
    if (busy) return
    const next = (pin + digit).slice(0, 4)
    setPin(next)
    if (next.length === 4 && shop) {
      setBusy(true)
      try {
        await login(shop.slug, name, next)
      } catch (e) {
        setError(e instanceof ApiError ? e.message : 'Could not sign in')
        setPin('')
      } finally {
        setBusy(false)
      }
    }
  }

  if (session) return <Navigate to="/shop" replace />

  return (
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center gap-4 px-4 py-8">
      <div className="flex items-center gap-3">
        <Logo size={48} />
        <div>
          <h1 className="text-2xl font-bold">Counter Drop</h1>
          <p className="text-sm text-ink-muted">Shop sign in</p>
        </div>
      </div>
      {error && <Banner tone="danger">{error}</Banner>}

      {!shop ? (
        <Card className="p-4">
          <form
            onSubmit={(e) => {
              e.preventDefault()
              findShop(slug)
            }}
          >
            <label htmlFor="slug" className="block font-semibold">
              Shop link name
            </label>
            <input id="slug" value={slug} onChange={(e) => setSlug(e.target.value)} autoCapitalize="none" autoCorrect="off" placeholder="imran-xerox" className="mt-2 h-12 w-full rounded border-[1.5px] border-line bg-surface px-3 font-mono" />
            <p className="mt-1 text-xs text-ink-muted">The name in your shop's QR link: …/s/<b>your-shop</b></p>
            <Button className="mt-3 w-full" type="submit" disabled={!slug.trim()}>
              Continue
            </Button>
          </form>
        </Card>
      ) : !name ? (
        <Card className="p-4">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-lg font-bold">{shop.name}</h2>
            <button className="text-sm font-semibold text-action" onClick={() => setShop(null)}>
              Change shop
            </button>
          </div>
          <p className="mb-2 text-sm text-ink-muted">Who's at the counter?</p>
          <div className="grid grid-cols-2 gap-2">
            {names.map((n) => (
              <Button key={n} variant="secondary" size="lg" onClick={() => setName(n)}>
                {n}
              </Button>
            ))}
          </div>
        </Card>
      ) : (
        <Card className="p-4">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-lg font-bold">{name}</h2>
            <button
              className="text-sm font-semibold text-action"
              onClick={() => {
                setName('')
                setPin('')
              }}
            >
              Not you?
            </button>
          </div>
          <p className="text-sm text-ink-muted">Enter your 4-digit PIN</p>
          <div className="my-4 flex justify-center gap-3" aria-live="polite" aria-label={`${pin.length} of 4 digits entered`}>
            {[0, 1, 2, 3].map((i) => (
              <span key={i} className={`h-4 w-4 rounded-full ${i < pin.length ? 'bg-ink' : 'border-2 border-line'}`} />
            ))}
          </div>
          {busy ? (
            <div className="flex justify-center py-8">
              <Spinner className="h-8 w-8" />
            </div>
          ) : (
            <div className="grid grid-cols-3 gap-2">
              {['1', '2', '3', '4', '5', '6', '7', '8', '9'].map((d) => (
                <Button key={d} variant="secondary" size="lg" className="font-mono text-2xl" onClick={() => press(d)}>
                  {d}
                </Button>
              ))}
              <span />
              <Button variant="secondary" size="lg" className="font-mono text-2xl" onClick={() => press('0')}>
                0
              </Button>
              <Button variant="ghost" size="lg" aria-label="Delete digit" onClick={() => setPin((p) => p.slice(0, -1))}>
                <Delete className="h-6 w-6" aria-hidden />
              </Button>
            </div>
          )}
        </Card>
      )}
    </div>
  )
}
