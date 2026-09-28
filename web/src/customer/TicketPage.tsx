import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { BellRing, Check, Lock, ShieldCheck, Share2 } from 'lucide-react'
import { api, ApiError } from '@/lib/api'
import { useI18n, type Key } from '@/lib/i18n'
import { clock, mmss, rupees } from '@/lib/format'
import { chime, useLive } from '@/lib/live'
import { findTicket, saveTicket } from '@/lib/tickets'
import type { JobState, Ticket } from '@/lib/types'
import { Banner, Button, Card, Spinner, stateMeta } from '@/components/ui'
import { Shell } from './DropPage'

const steps: { state: JobState; key: Key }[] = [
  { state: 'queued', key: 'inLine' },
  { state: 'claimed', key: 'printing' },
  { state: 'ready', key: 'ready' },
]
const order: Record<JobState, number> = { uploading: -1, queued: 0, claimed: 1, ready: 2, collected: 3, cancelled: -1 }

export default function TicketPage() {
  const { jobId = '' } = useParams()
  const location = useLocation()
  const { t } = useI18n()
  const [secret] = useState(() => {
    // Secret comes in the URL fragment (never sent to servers) or from this phone's saved tickets.
    const frag = location.hash.replace(/^#/, '')
    return frag || findTicket(jobId)?.secret || ''
  })
  const [ticket, setTicket] = useState<Ticket | null>(null)
  const [error, setError] = useState('')
  const [soundOn, setSoundOn] = useState(false)
  const [now, setNow] = useState(Date.now())
  const lastState = useRef<JobState | null>(null)
  const skew = useRef(0)

  useEffect(() => {
    if (location.hash) window.history.replaceState(null, '', location.pathname)
  }, [location])

  const load = useCallback(async () => {
    if (!secret) return
    try {
      const tk = await api<Ticket>(`/jobs/${jobId}`, { secret })
      skew.current = new Date(tk.serverTime).getTime() - Date.now()
      setTicket(tk)
      setError('')
      const saved = findTicket(jobId)
      if (!saved) saveTicket({ jobId, secret, slug: tk.shop.slug, shopName: tk.shop.name, token: tk.job.token, createdAt: Date.now() })
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Could not load')
    }
  }, [jobId, secret])

  useEffect(() => {
    load()
  }, [load])
  useLive(secret ? `/jobs/${jobId}/events?secret=${encodeURIComponent(secret)}` : null, () => load(), load)

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(id)
  }, [])

  // The ready moment: vibrate + chime once when the job turns ready.
  useEffect(() => {
    const st = ticket?.job.state
    if (!st) return
    if (lastState.current && lastState.current !== 'ready' && st === 'ready') {
      navigator.vibrate?.([300, 150, 300])
      if (soundOn) chime('ready')
    }
    lastState.current = st
  }, [ticket?.job.state, soundOn])

  if (!secret) {
    return (
      <Shell>
        <Card className="p-5 text-center">{t('openFromPhone')}</Card>
      </Shell>
    )
  }
  if (!ticket) {
    return (
      <Shell>
        {error ? (
          <Banner tone="danger">{error}</Banner>
        ) : (
          <div className="flex justify-center py-16">
            <Spinner className="h-8 w-8" />
          </div>
        )}
      </Shell>
    )
  }

  const { job, shop, quote } = ticket
  const serverNow = now + skew.current
  const isReady = job.state === 'ready'
  const isDone = job.state === 'collected' || job.state === 'cancelled'
  const idx = order[job.state]
  const deleteAt = ticket.undoUntil ? new Date(ticket.undoUntil).getTime() : 0
  const filesCount = job.files.length

  const cancel = async () => {
    if (!window.confirm(t('cancelConfirm'))) return
    try {
      setTicket(await api<Ticket>(`/jobs/${jobId}/cancel`, { body: {}, secret }))
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Could not cancel')
    }
  }

  const share = async () => {
    const text = `Counter Drop · ${shop.name} · ${job.token}: ${t(filesCount === 1 ? 'deletedAt1' : 'deletedAt', { n: filesCount, time: clock(job.filesDeletedAt) })}`
    try {
      if (navigator.share) await navigator.share({ text })
      else await navigator.clipboard.writeText(text)
    } catch {
      /* user closed the share sheet */
    }
  }

  return (
    <Shell shop={shop}>
      {error && <Banner tone="danger">{error}</Banner>}

      <section aria-live="assertive" className={`rounded border-2 p-5 text-center ${isReady ? 'border-ready bg-ready text-white' : isDone ? 'border-line bg-surface-tint text-ink-muted' : 'border-ink bg-ink text-canvas'}`}>
        <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wide opacity-80">
          <span>{job.lane ? `Lane ${job.lane}` : ''}</span>
          <span>{t('yourToken')}</span>
        </div>
        <div className="my-2 font-mono text-7xl font-bold tracking-tight">{job.token ?? '—'}</div>
        <p className="text-sm opacity-90">{isReady ? t('readyNow', { token: job.token ?? '' }) : job.state === 'collected' ? t('doneCollected') : job.state === 'cancelled' ? t('doneCancelled') : t('showToken')}</p>
        {job.state === 'queued' && (
          <div className="mt-3 rounded bg-surface px-3 py-2 text-left text-sm text-ink">
            <div className="font-semibold">{ticket.position === 0 ? t('nextUp') : t('ahead', { n: ticket.position })}</div>
            {job.readyBy && <div className="text-ink-muted">{t('readyBy', { time: clock(job.readyBy) })}</div>}
          </div>
        )}
      </section>

      {job.state !== 'cancelled' && (
        <Card className="p-4">
          <ol className="flex items-start justify-between" aria-label="Status">
            {steps.map((s, i) => {
              const done = idx > i || job.state === 'collected'
              const active = idx === i
              const { Icon } = stateMeta[s.state]
              return (
                <li key={s.state} className="flex flex-1 flex-col items-center text-center">
                  <span className={`flex h-10 w-10 items-center justify-center rounded-full ${done ? 'bg-ready text-white' : active ? 'bg-action text-white' : 'bg-surface-tint text-ink-subtle'}`}>
                    {done ? <Check className="h-5 w-5" aria-hidden /> : <Icon className="h-5 w-5" aria-hidden />}
                  </span>
                  <span className={`mt-1 text-sm font-semibold ${active ? 'text-action' : done ? 'text-ready' : 'text-ink-subtle'}`}>{t(s.key)}</span>
                </li>
              )
            })}
          </ol>
        </Card>
      )}

      {(job.state === 'queued' || job.state === 'claimed') && (
        <button type="button" onClick={() => { setSoundOn(true); chime('soft') }} className="flex w-full items-center gap-3 rounded border-[1.5px] border-line bg-surface-tint p-3 text-left text-sm">
          <BellRing className="h-5 w-5 text-action" aria-hidden />
          <span className="flex-1">{soundOn ? t('soundReady') : t('soundOn')}</span>
        </button>
      )}

      {quote && job.state !== 'cancelled' && (
        <Card className="p-4">
          <div className="flex items-end justify-between">
            <div>
              <div className="text-sm text-ink-muted">{job.state === 'collected' ? t('total') : job.pagesToConfirm ? t('priceAtCounter') : t('due')}</div>
              <div className="font-mono text-2xl font-bold tabular">{rupees(job.priceTotalPaise || quote.totalPaise)}</div>
            </div>
            <div className="text-right text-sm text-ink-muted">
              {job.files.length} × <span className="font-mono">{job.pagesTotal || quote.pagesTotal}</span> pp
            </div>
          </div>
        </Card>
      )}

      {job.state === 'claimed' && (
        <p className="flex items-center gap-2 rounded border-[1.5px] border-line bg-surface p-3 text-sm text-ink-muted">
          <Lock className="h-4 w-4 text-attention" aria-hidden /> {t('locked2')}
        </p>
      )}
      {(job.state === 'queued' || job.state === 'uploading') && (
        <Button variant="danger" className="w-full" onClick={cancel}>
          {t('cancel')}
        </Button>
      )}

      {(job.state === 'collected' || job.state === 'cancelled') && (
        <Card className="p-4">
          <h2 className="mb-2 flex items-center gap-2 text-lg font-bold">
            <ShieldCheck className="h-6 w-6 text-ready" aria-hidden /> {t('receipt')}
          </h2>
          {job.state === 'collected' && job.collectedAt && <p className="text-sm text-ink-muted">{t('collectedAt', { time: clock(job.collectedAt) })}</p>}
          {job.state === 'cancelled' && <p className="text-sm text-ink-muted">{t('cancelledNote')}</p>}
          {job.filesDeletedAt ? (
            <p className="mt-2 font-semibold text-ready">{t(filesCount === 1 ? 'deletedAt1' : 'deletedAt', { n: filesCount, time: clock(job.filesDeletedAt) })}</p>
          ) : deleteAt ? (
            <p className="mt-2 font-semibold">{t('deleteIn', { time: mmss(deleteAt - serverNow) })}</p>
          ) : null}
          {job.filesDeletedAt && (
            <Button variant="secondary" size="sm" className="mt-3" onClick={share}>
              <Share2 className="h-4 w-4" aria-hidden /> {t('shareReceipt')}
            </Button>
          )}
        </Card>
      )}

      {(job.state === 'collected' || job.state === 'cancelled') && (
        <Link to={`/s/${shop.slug}`} className="block text-center font-semibold text-action underline">
          {t('newJob')}
        </Link>
      )}
      <p className="pb-6 text-center text-xs text-ink-muted">{t('help')}</p>
    </Shell>
  )
}
