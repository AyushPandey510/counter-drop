import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { api, ApiError, store } from '@/lib/api'
import type { Shop, Staff } from '@/lib/types'

interface Session {
  token: string
  staff: Staff
  shop: Shop
}

type Ctx = {
  session: Session | null
  loading: boolean
  login: (shop: string, name: string, pin: string) => Promise<void>
  /** Start a session from a response shaped like /shop/login (used by the setup link page). */
  adopt: (res: { token: string; staff: Staff; shop: Shop }) => void
  logout: () => Promise<void>
  call: <T>(path: string, opts?: { method?: string; body?: unknown }) => Promise<T>
  setShop: (s: Shop) => void
}

const AuthCtx = createContext<Ctx | null>(null)
const KEY = 'cd.shop.session'

export function ShopAuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const token = store.get(KEY)
    if (!token) {
      setLoading(false)
      return
    }
    api<{ staff: Staff; shop: Shop }>('/shop/me', { token })
      .then((me) => setSession({ token, ...me }))
      .catch(() => store.remove(KEY))
      .finally(() => setLoading(false))
  }, [])

  const adopt = useCallback((res: { token: string; staff: Staff; shop: Shop }) => {
    store.set(KEY, res.token)
    store.set('cd.shop.slug', res.shop.slug)
    setSession({ token: res.token, staff: res.staff, shop: res.shop })
  }, [])

  const logout = useCallback(async () => {
    if (session) await api('/shop/logout', { body: {}, token: session.token }).catch(() => {})
    store.remove(KEY)
    setSession(null)
  }, [session])

  // `call` and `setShop` keep the same identity for the whole session. Screens use them as effect
  // dependencies, so if they changed on every shop update a screen would re-fetch in an endless loop.
  const tokenRef = useRef(session?.token)
  tokenRef.current = session?.token

  const call = useCallback(async <T,>(path: string, opts: { method?: string; body?: unknown } = {}) => {
    try {
      return await api<T>(path, { ...opts, token: tokenRef.current })
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) {
        store.remove(KEY)
        setSession(null)
      }
      throw e
    }
  }, [])

  const setShop = useCallback((shop: Shop) => {
    setSession((s) => (s && JSON.stringify(s.shop) !== JSON.stringify(shop) ? { ...s, shop } : s))
  }, [])

  const value = useMemo<Ctx>(
    () => ({
      session,
      loading,
      login: async (shop, name, pin) => {
        const res = await api<{ token: string; staff: Staff; shop: Shop }>('/shop/login', { body: { shop, name, pin } })
        adopt(res)
      },
      adopt,
      logout,
      call,
      setShop,
    }),
    [session, loading, logout, adopt, call, setShop],
  )
  return <AuthCtx.Provider value={value}>{children}</AuthCtx.Provider>
}

export function useShopAuth(): Ctx {
  const ctx = useContext(AuthCtx)
  if (!ctx) throw new Error('ShopAuthProvider missing')
  return ctx
}

export function RequireStaff({ children, owner = false }: { children: ReactNode; owner?: boolean }) {
  const { session, loading } = useShopAuth()
  if (loading) return null
  if (!session) return <Navigate to="/shop/login" replace />
  if (owner && session.staff.role !== 'owner') return <Navigate to="/shop" replace />
  return <>{children}</>
}

/** Dark theme is a per-device choice for shop screens only. */
export function useShopTheme(): [boolean, (dark: boolean) => void] {
  const [dark, setDark] = useState(() => store.get('cd.shop.theme') === 'dark')
  useEffect(() => {
    document.documentElement.dataset.theme = dark ? 'dark' : 'light'
    return () => {
      document.documentElement.dataset.theme = 'light'
    }
  }, [dark])
  return [
    dark,
    (d) => {
      store.set('cd.shop.theme', d ? 'dark' : 'light')
      setDark(d)
    },
  ]
}
