import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { BellRing, Check, Download, Lock, ShieldCheck, Share2, Trash2 } from 'lucide-react'
import { api, ApiError, store } from '@/lib/api'
import { useI18n, type Key } from '@/lib/i18n'
import { clock, dayTime, mmss, rupees } from '@/lib/format'
import { chime, useLive, type LiveTicket } from '@/lib/live'
import { findTicket, saveTicket } from '@/lib/tickets'
import type { Job, JobState, Ticket } from '@/lib/types'
import { Banner, Button, Card, Spinner, stateMeta } from '@/components/ui'
import { Shell } from './DropPage'
import { InstallCard } from '@/components/InstallCard'

const steps: { state: JobState; key: Key }[] = [
  { state: 'queued', key: 'inLine' },
  { state: 'claimed', key: 'printing' },
  { state: 'ready', key: 'ready' },
]
const order: Record<JobState, number> = {
  uploading: -1,
  queued: 0,
  claimed: 1,
  ready: 2,
  collected: 3,
  cancelled: -1,
}

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
  const [soundOn, setSoundOn] = useState(() => store.get(`cd:sound:${jobId}`) === 'on')
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
      if (!saved)
        saveTicket({
          jobId,
          secret,
          slug: tk.shop.slug,
          shopName: tk.shop.name,
          token: tk.job.token,
          createdAt: Date.now(),
        })
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Could not load')
    }
  }, [jobId, secret])

  useEffect(() => {
    load()
  }, [load])
  useLive(
    secret
      ? {
          sse: `/jobs/${jobId}/events?secret=${encodeURIComponent(secret)}`,
          ticket: () => api<LiveTicket>(`/jobs/${jobId}/live`, { method: 'POST', secret }),
        }
      : null,
    (e) => {
      // A download is something the customer should notice even if the screen is idle.
      if (e.type === 'file.downloaded') navigator.vibrate?.(200)
      load()
    },
    load,
  )

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
  const isActive = job.state === 'queued' || job.state === 'claimed' || job.state === 'ready'
  const idx = order[job.state]
  const deleteAt = ticket.undoUntil ? new Date(ticket.undoUntil).getTime() : 0
  const downloaded = job.files.filter((f) => f.downloads > 0)
  const fileName = (i: number) => job.files[i]?.filename || t('rFileN', { n: i + 1 })

  const cancel = async () => {
    if (!window.confirm(t('cancelConfirm'))) return
    try {
      setTicket(await api<Ticket>(`/jobs/${jobId}/cancel`, { body: {}, secret }))
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Could not cancel')
    }
  }

  const askDelete = async () => {
    if (!window.confirm(t('askDeleteConfirm'))) return
    try {
      setTicket(await api<Ticket>(`/jobs/${jobId}/delete-request`, { body: {}, secret }))
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Could not send the request')
    }
  }

  const ourCopy = job.filesDeletedAt ? t('rOurDeleted', { time: clock(job.filesDeletedAt) }) : deleteAt ? t('rOurDeleteIn', { time: mmss(deleteAt - serverNow) }) : ''
  const shopCopies = shopCopiesText(job, t)

  const share = async () => {
    const lines = [
      `Counter Drop · ${shop.name} · ${job.token ?? ''}`,
      job.state === 'collected' ? `${t('rAmount')}: ${rupees(job.priceTotalPaise)} · ${t('rPaid')}: ${paidLabel(job, t)}` : t(job.cancelReason === 'not_collected' ? 'notCollected' : 'cancelledNote'),
      ...job.files.map((f, i) => `${fileName(i)} · ${fileStatus(f, t)}`),
      `${t('rOurCopy')}: ${ourCopy}`,
      `${t('rShopCopies')}: ${shopCopies}`,
    ]
    const text = lines.filter(Boolean).join('\n')
    try {
      if (navigator.share) await navigator.share({ text })
      else await navigator.clipboard.writeText(text)
    } catch {
      /* user closed the share sheet */
    }
  }

  return (
    <Shell shop={shop} wide>
      {error && <Banner tone="danger">{error}</Banner>}
      <div className="space-y-3 lg:grid lg:grid-cols-2 lg:items-start lg:gap-6 lg:space-y-0">
        <div className="space-y-3">
          <section
            aria-live="assertive"
            className={`rounded border-2 p-5 text-center ${isReady ? 'border-ready bg-ready text-white' : isDone ? 'border-line bg-surface-tint text-ink-muted' : 'border-ink bg-ink text-canvas'}`}
          >
            <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wide opacity-80">
              <span>{job.lane ? `Lane ${job.lane}` : ''}</span>
              <span>{t('yourToken')}</span>
            </div>
            <div className="my-2 font-mono text-7xl font-bold tracking-tight lg:my-6 lg:text-8xl">{job.token ?? '—'}</div>
            <p className="text-sm opacity-90">
              {isReady ? t('readyNow', { token: job.token ?? '' }) : job.state === 'collected' ? t('doneCollected') : job.state === 'cancelled' ? t('doneCancelled') : t('showToken')}
            </p>
            {job.state === 'queued' && (
              <div className="mt-3 rounded bg-surface px-3 py-2 text-left text-sm text-ink">
                <div className="font-semibold">{ticket.position === 0 ? t('nextUp') : t('ahead', { n: ticket.position })}</div>
                {job.readyBy && <div className="text-ink-muted">{t('readyBy', { time: clock(job.readyBy) })}</div>}
              </div>
            )}
          </section>

          {isActive && downloaded.length > 0 && (
            <div role="status" className="space-y-1 rounded border-[1.5px] border-attention bg-attention-bg p-3 text-sm text-attention">
              {downloaded.map((f) => (
                <p key={f.id} className="flex items-start gap-2">
                  <Download className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
                  {t('downloadedNotice', { shop: shop.name, file: f.filename, time: clock(f.downloadedAt) })}
                </p>
              ))}
            </div>
          )}

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
              {isActive && ticket.holdUntil && <p className="mt-3 border-t border-divider pt-2 text-center text-xs text-ink-muted">{t('collectBy', { time: dayTime(ticket.holdUntil) })}</p>}
            </Card>
          )}

          {(job.state === 'queued' || job.state === 'claimed') && (
            <button
              type="button"
              onClick={() => {
                setSoundOn(true)
                store.set(`cd:sound:${jobId}`, 'on')
                chime('soft')
              }}
              className="flex w-full items-center gap-3 rounded border-[1.5px] border-line bg-surface-tint p-3 text-left text-sm"
            >
              <BellRing className="h-5 w-5 text-action" aria-hidden />
              <span className="flex-1">{soundOn ? t('soundReady') : t('soundOn')}</span>
            </button>
          )}

          {job.state === 'claimed' && (
            <p className="flex items-center gap-2 rounded border-[1.5px] border-line bg-surface p-3 text-sm text-ink-muted">
              <Lock className="h-4 w-4 text-attention" aria-hidden /> {t('locked2')}
            </p>
          )}
        </div>
        <div className="space-y-3">
          {quote && !isDone && (
            <Card className="p-4">
              <div className="flex items-end justify-between">
                <div>
                  <div className="text-sm text-ink-muted">{job.pagesToConfirm ? t('priceAtCounter') : t('due')}</div>
                  <div className="font-mono text-2xl font-bold tabular">{rupees(job.priceTotalPaise || quote.totalPaise)}</div>
                </div>
                <div className="text-right text-sm text-ink-muted">
                  {job.files.length} × <span className="font-mono">{job.pagesTotal || quote.pagesTotal}</span> pp
                </div>
              </div>
            </Card>
          )}

          {(job.state === 'queued' || job.state === 'uploading') && (
            <Button variant="danger" className="w-full" onClick={cancel}>
              {t('cancel')}
            </Button>
          )}

          {isDone && (
            <Card className="p-4">
              <div className="mb-3 flex items-start justify-between gap-2">
                <h2 className="flex items-center gap-2 text-lg font-bold">
                  <ShieldCheck className="h-6 w-6 text-ready" aria-hidden /> {t('receipt')}
                </h2>
                <div className="text-right text-xs text-ink-muted">
                  <div className="font-semibold text-ink">{shop.name}</div>
                  <div>
                    <span className="font-mono">{job.token}</span> · {dayTime(job.collectedAt || job.cancelledAt)}
                  </div>
                </div>
              </div>

              {job.state === 'collected' ? (
                <dl className="grid grid-cols-2 gap-2 rounded bg-surface-tint p-3 text-sm">
                  <div>
                    <dt className="text-ink-muted">{t('rAmount')}</dt>
                    <dd className="font-mono text-xl font-bold tabular">{rupees(job.priceTotalPaise)}</dd>
                  </div>
                  <div>
                    <dt className="text-ink-muted">{t('rPaid')}</dt>
                    <dd className="font-semibold">{paidLabel(job, t)}</dd>
                  </div>
                </dl>
              ) : (
                <p className="rounded bg-surface-tint p-3 text-sm text-ink-muted">{t(job.cancelReason === 'not_collected' ? 'notCollected' : 'cancelledNote')}</p>
              )}

              <h3 className="mb-1 mt-4 text-sm font-semibold uppercase tracking-wide text-ink-muted">{t('rFiles')}</h3>
              <ul className="divide-y divide-divider text-sm">
                {job.files.map((f, i) => (
                  <li key={f.id} className="flex items-start justify-between gap-3 py-2">
                    <span className="min-w-0">
                      <span className="block truncate font-semibold">{fileName(i)}</span>
                      <span className="text-xs text-ink-muted">{f.pages ? `${f.pages} pp` : ''}</span>
                    </span>
                    <span className={`shrink-0 text-right text-xs ${f.downloads > 0 ? 'font-semibold text-attention' : 'text-ink-muted'}`}>{fileStatus(f, t)}</span>
                  </li>
                ))}
              </ul>

              <dl className="mt-3 space-y-2 border-t border-divider pt-3 text-sm">
                <div className="flex justify-between gap-3">
                  <dt className="text-ink-muted">{t('rOurCopy')}</dt>
                  <dd className={`text-right font-semibold ${job.filesDeletedAt ? 'text-ready' : ''}`}>{ourCopy}</dd>
                </div>
                <div className="flex justify-between gap-3">
                  <dt className="text-ink-muted">{t('rShopCopies')}</dt>
                  <dd className={`text-right font-semibold ${job.copiesDeletedAt || downloaded.length === 0 ? 'text-ready' : 'text-attention'}`}>{shopCopies}</dd>
                </div>
              </dl>

              <div className="mt-4 flex flex-wrap gap-2">
                {job.state === 'collected' && downloaded.length > 0 && !job.copiesDeleteRequestedAt && !job.copiesDeletedAt && (
                  <Button variant="danger" size="sm" onClick={askDelete}>
                    <Trash2 className="h-4 w-4" aria-hidden /> {t('askDelete')}
                  </Button>
                )}
                <Button variant="secondary" size="sm" onClick={share}>
                  <Share2 className="h-4 w-4" aria-hidden /> {t('shareReceipt')}
                </Button>
              </div>
            </Card>
          )}

          <InstallCard />
          {isDone && (
            <Link to={`/s/${shop.slug}`} className="block text-center font-semibold text-action underline">
              {t('newJob')}
            </Link>
          )}
          <p className="pb-6 text-center text-xs text-ink-muted lg:text-left">{t('help')}</p>
        </div>
      </div>
    </Shell>
  )
}

type T = (k: Key, v?: Record<string, string | number>) => string

function paidLabel(job: Job, t: T): string {
  return job.paidMethod === 'cash' ? t('rPaidCash') : job.paidMethod === 'upi' ? t('rPaidUpi') : t('rPaidCounter')
}

function fileStatus(f: Job['files'][number], t: T): string {
  if (f.downloads > 0) return t('rDownloaded', { name: f.downloadedBy || '—', time: clock(f.downloadedAt) })
  if (f.printOpens > 0) return t('rPrinted')
  return t('rNotPrinted')
}

function shopCopiesText(job: Job, t: T): string {
  if (!job.files.some((f) => f.downloads > 0)) return t('rShopNone')
  if (job.copiesDeletedAt) return t('rShopDeleted', { name: job.copiesDeletedBy || '—', time: clock(job.copiesDeletedAt) })
  if (job.copiesDeleteRequestedAt) return t('rShopRequested', { time: clock(job.copiesDeleteRequestedAt) })
  return t('rShopPending')
}
