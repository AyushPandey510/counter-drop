import { useCallback, useEffect, useState } from 'react'
import { Copy, Link2, MessageCircle, Share2, UserMinus, UserPlus, X } from 'lucide-react'
import { ApiError } from '@/lib/api'
import { setupMessage } from '@/lib/pin'
import { clock } from '@/lib/format'
import type { SetupLinkOut, StaffMember } from '@/lib/types'
import { Banner, Button, Card, Chip, Segmented, Spinner } from '@/components/ui'
import { useShopAuth } from './auth'

type Shown = { person: string; link: SetupLinkOut }

// Owner's staff list: add people, send one-time PIN links, remove people. Nobody ever sees another person's PIN.
export function StaffSection() {
  const { session, call } = useShopAuth()
  const [list, setList] = useState<StaffMember[] | null>(null)
  const [name, setName] = useState('')
  const [role, setRole] = useState<'staff' | 'owner'>('staff')
  const [shown, setShown] = useState<Shown | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = useCallback(async () => {
    const res = await call<{ staff: StaffMember[] }>('/shop/staff')
    setList(res.staff)
  }, [call])
  useEffect(() => {
    load().catch(() => setError('Could not load staff'))
  }, [load])

  const run = async (fn: () => Promise<void>) => {
    setError('')
    setBusy(true)
    try {
      await fn()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Something went wrong')
    } finally {
      setBusy(false)
    }
  }

  const add = (e: React.FormEvent) => {
    e.preventDefault()
    run(async () => {
      const res = await call<{ staff: StaffMember; link: SetupLinkOut }>('/shop/staff', { body: { name: name.trim(), role } })
      setShown({ person: res.staff.name, link: res.link })
      setName('')
      setRole('staff')
      await load()
    })
  }

  const newLink = (m: StaffMember) => {
    if (!m.pending && !window.confirm(`Send ${m.name} a new PIN link?\n\nTheir current PIN stops working now and they'll be signed out until they use the link.`)) return
    run(async () => {
      const res = await call<{ link: SetupLinkOut }>(`/shop/staff/${m.id}/link`, { body: {} })
      setShown({ person: m.name, link: res.link })
      await load()
    })
  }

  const remove = (m: StaffMember) => {
    if (!window.confirm(`Remove ${m.name}?\n\nThey'll be signed out and can't sign in again. You can add them back later.`)) return
    run(async () => {
      await call(`/shop/staff/${m.id}`, { method: 'DELETE' })
      if (shown?.person === m.name) setShown(null)
      await load()
    })
  }

  return (
    <Card className="space-y-3 p-4">
      <div>
        <h2 className="text-lg font-bold">Staff</h2>
        <p className="text-sm text-ink-muted">Each person gets a one-time link and chooses their own PIN. You never see anyone's PIN.</p>
      </div>
      {error && <Banner tone="danger">{error}</Banner>}
      {shown && session && <LinkCard shown={shown} shopName={session.shop.name} onClose={() => setShown(null)} />}

      {!list ? (
        <div className="flex justify-center py-4">
          <Spinner />
        </div>
      ) : (
        <ul className="divide-y divide-line rounded border-[1.5px] border-line">
          {list.map((m) => {
            const me = m.id === session?.staff.id
            return (
              <li key={m.id} className="flex flex-wrap items-center gap-2 p-3">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-semibold">{m.name}</span>
                    <Chip tone={m.role === 'owner' ? 'action' : 'neutral'}>{m.role === 'owner' ? 'Owner' : 'Staff'}</Chip>
                    {me && <Chip tone="dark">You</Chip>}
                    {m.pending && <Chip tone="attention">Waiting for setup</Chip>}
                  </div>
                  {!m.pending && m.pinSetAt && (
                    <div className="text-xs text-ink-muted">PIN set {new Date(m.pinSetAt).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', timeZone: 'Asia/Kolkata' })}</div>
                  )}
                </div>
                {!me && (
                  <div className="flex gap-2">
                    <Button size="sm" variant="secondary" disabled={busy} onClick={() => newLink(m)}>
                      <Link2 className="h-4 w-4" aria-hidden /> {m.pending ? 'New link' : 'Reset PIN'}
                    </Button>
                    <Button size="sm" variant="danger" disabled={busy} onClick={() => remove(m)} aria-label={`Remove ${m.name}`}>
                      <UserMinus className="h-4 w-4" aria-hidden />
                    </Button>
                  </div>
                )}
              </li>
            )
          })}
        </ul>
      )}

      <form onSubmit={add} className="space-y-2 rounded border-[1.5px] border-dashed border-line p-3">
        <label className="block text-sm font-semibold">
          Add a person
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="First name, e.g. Sana"
            maxLength={30}
            className="mt-1 h-11 w-full rounded border-[1.5px] border-line bg-surface px-3"
          />
        </label>
        <Segmented
          label="Role"
          value={role}
          onChange={setRole}
          options={[
            { value: 'staff', label: 'Staff', hint: 'Runs the counter' },
            { value: 'owner', label: 'Owner', hint: 'Also prices, hours, staff' },
          ]}
        />
        <Button type="submit" className="w-full" disabled={busy || name.trim().length < 2}>
          <UserPlus className="h-4 w-4" aria-hidden /> Create PIN link
        </Button>
      </form>
    </Card>
  )
}

function LinkCard({ shown, shopName, onClose }: { shown: Shown; shopName: string; onClose: () => void }) {
  const url = window.location.origin + shown.link.setupPath
  const text = setupMessage({ person: shown.person, shopName, url, expiresAt: shown.link.expiresAt, reset: shown.link.purpose === 'reset' })
  const [copied, setCopied] = useState(false)

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 2000)
    } catch {
      window.prompt('Copy this message', text)
    }
  }
  const share = async () => {
    try {
      await navigator.share({ text })
    } catch {
      /* closed */
    }
  }

  return (
    <div className="space-y-2 rounded border-2 border-action bg-surface-tint p-3" role="region" aria-label={`PIN link for ${shown.person}`}>
      <div className="flex items-start justify-between gap-2">
        <div>
          <div className="font-bold">Send this link to {shown.person} only</div>
          <div className="text-xs text-ink-muted">
            Works once · expires {clock(shown.link.expiresAt)}, {new Date(shown.link.expiresAt).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', timeZone: 'Asia/Kolkata' })}. Until it's
            used, anyone with it can set {shown.person}'s PIN.
          </div>
        </div>
        <button type="button" onClick={onClose} className="rounded p-1 hover:bg-surface" aria-label="Close">
          <X className="h-4 w-4" aria-hidden />
        </button>
      </div>
      <input readOnly value={url} onFocus={(e) => e.target.select()} className="h-10 w-full rounded border-[1.5px] border-line bg-surface px-2 font-mono text-xs" aria-label="Setup link" />
      <div className="flex flex-wrap gap-2">
        <Button size="sm" onClick={copy}>
          <Copy className="h-4 w-4" aria-hidden /> {copied ? 'Copied' : 'Copy message'}
        </Button>
        <Button size="sm" variant="secondary" onClick={() => window.open(`https://wa.me/?text=${encodeURIComponent(text)}`, '_blank', 'noopener')}>
          <MessageCircle className="h-4 w-4" aria-hidden /> WhatsApp
        </Button>
        {typeof navigator.share === 'function' && (
          <Button size="sm" variant="secondary" onClick={share}>
            <Share2 className="h-4 w-4" aria-hidden /> Share
          </Button>
        )}
      </div>
    </div>
  )
}
