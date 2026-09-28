import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { AlertTriangle, FileText, Image as ImageIcon, Plus, ShieldCheck, Trash2, WifiOff, Clock, RotateCw } from 'lucide-react'
import { api, ApiError } from '@/lib/api'
import { useI18n } from '@/lib/i18n'
import { rupees, hhmm, bytes } from '@/lib/format'
import { checkPdf } from '@/lib/pdf'
import { uploadFile, mimeOf, ACCEPTED, ALLOWED_MIMES } from '@/lib/upload'
import { saveTicket, savedTickets } from '@/lib/tickets'
import type { PublicShop, Ticket, UploadTarget, FileSettings } from '@/lib/types'
import { Banner, Button, Card, Chip, LangSwitch, Logo, Segmented, Spinner, Stepper } from '@/components/ui'

const MAX_FILES = 20
const MAX_FILE_MB = 25
const MAX_JOB_MB = 50

type Status = 'checking' | 'uploading' | 'waiting' | 'done' | 'error' | 'locked'
interface LocalFile {
  clientId: string
  file: File
  mime: string
  fileId?: string
  pages: number
  status: Status
  progress: number
  error?: string
  pageRange: string
}

let seq = 0
const nextId = () => `c${Date.now().toString(36)}${(seq++).toString(36)}`

export default function DropPage() {
  const { slug = '' } = useParams()
  const { t } = useI18n()
  const navigate = useNavigate()
  const [shop, setShop] = useState<PublicShop | null>(null)
  const [shopError, setShopError] = useState<string>('')
  const [files, setFiles] = useState<LocalFile[]>([])
  const [job, setJob] = useState<{ id: string; secret: string } | null>(null)
  const [ticket, setTicket] = useState<Ticket | null>(null)
  const [settings, setSettings] = useState<FileSettings>({ copies: 1, colour: false, bothSides: false })
  const [name, setName] = useState('')
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const jobRef = useRef(job)
  jobRef.current = job

  const loadShop = useCallback(() => {
    api<PublicShop>(`/shops/${slug}`)
      .then((s) => {
        setShop(s)
        setShopError('')
      })
      .catch((e: ApiError) => setShopError(e.status === 404 ? t('notFound') : e.message))
  }, [slug, t])

  useEffect(loadShop, [loadShop])
  useEffect(() => {
    if (shop?.onlineState !== 'paused' && shop?.onlineState !== 'offline') return
    const id = window.setInterval(loadShop, 30000)
    return () => window.clearInterval(id)
  }, [shop?.onlineState, loadShop])

  const openTicket = useMemo(() => savedTickets().find((x) => x.slug === slug && x.token), [slug])

  const patchFile = (clientId: string, patch: Partial<LocalFile>) => setFiles((fs) => fs.map((f) => (f.clientId === clientId ? { ...f, ...patch } : f)))

  // Upload one file to its presigned URL, then tell the API it's done (with the page count).
  const sendOne = async (lf: LocalFile, target: UploadTarget, j: { id: string; secret: string }) => {
    patchFile(lf.clientId, { fileId: target.fileId, status: 'uploading' })
    try {
      await uploadFile(
        lf.file,
        target,
        (p) => patchFile(lf.clientId, { progress: p }),
        (w) => patchFile(lf.clientId, { status: w ? 'waiting' : 'uploading' }),
      )
      const tk = await api<Ticket>(`/jobs/${j.id}/files/${target.fileId}/complete`, { body: { pages: lf.pages }, secret: j.secret })
      patchFile(lf.clientId, { status: 'done', progress: 1 })
      setTicket(tk)
    } catch (e) {
      patchFile(lf.clientId, { status: 'error', error: e instanceof ApiError ? e.message : t('uploadFailed') })
    }
  }

  const onPick = async (list: FileList | null) => {
    if (!list || !shop) return
    setError('')
    const picked = Array.from(list)
    const existing = files.filter((f) => f.status !== 'locked' && f.status !== 'error')
    if (existing.length + picked.length > MAX_FILES) {
      setError(`Up to ${MAX_FILES} files per job.`)
      return
    }
    const fresh: LocalFile[] = []
    for (const file of picked) {
      const mime = mimeOf(file)
      if (!ALLOWED_MIMES.has(mime)) {
        setError(`${file.name}: we can print PDF and photos. Save documents as PDF first.`)
        continue
      }
      if (file.size > MAX_FILE_MB * 1024 * 1024) {
        setError(`${file.name} is ${bytes(file.size)}. The limit is ${MAX_FILE_MB} MB.`)
        continue
      }
      fresh.push({ clientId: nextId(), file, mime, pages: mime.startsWith('image/') ? 1 : 0, status: 'checking', progress: 0, pageRange: '' })
    }
    const total = [...existing, ...fresh].reduce((s, f) => s + f.file.size, 0)
    if (total > MAX_JOB_MB * 1024 * 1024) {
      setError(`Files must add up to under ${MAX_JOB_MB} MB.`)
      return
    }
    if (fresh.length === 0) return
    setFiles((fs) => [...fs, ...fresh])

    // Count PDF pages on the phone; locked PDFs are stopped here.
    await Promise.all(
      fresh.map(async (lf) => {
        if (lf.mime !== 'application/pdf') return
        const r = await checkPdf(lf.file)
        lf.pages = r.pages
        if (r.locked) {
          lf.status = 'locked'
          patchFile(lf.clientId, { status: 'locked', error: t('locked') })
        } else patchFile(lf.clientId, { pages: r.pages })
      }),
    )
    const toSend = fresh.filter((f) => f.status !== 'locked')
    if (toSend.length === 0) return
    const meta = toSend.map((f) => ({ clientId: f.clientId, filename: f.file.name, size: f.file.size, mime: f.mime }))

    try {
      let j = jobRef.current
      let uploads: UploadTarget[]
      if (!j) {
        const res = await api<{ ticket: Ticket; secret: string; uploads: UploadTarget[] }>(`/shops/${slug}/jobs`, { body: { files: meta } })
        j = { id: res.ticket.job.id, secret: res.secret }
        setJob(j)
        setTicket(res.ticket)
        uploads = res.uploads
        if (settings.colour || settings.bothSides || settings.copies !== 1) {
          api<Ticket>(`/jobs/${j.id}`, { method: 'PATCH', body: { applyToAll: settings }, secret: j.secret }).then(setTicket).catch(() => {})
        }
      } else {
        const res = await api<{ ticket: Ticket; uploads: UploadTarget[] }>(`/jobs/${j.id}/files`, { body: { files: meta }, secret: j.secret })
        setTicket(res.ticket)
        uploads = res.uploads
      }
      const byClient = new Map(uploads.map((u) => [u.clientId, u]))
      // At most 3 uploads at a time.
      const queue = toSend.slice()
      const jobNow = j
      await Promise.all(
        Array.from({ length: Math.min(3, queue.length) }, async () => {
          for (let lf = queue.shift(); lf; lf = queue.shift()) {
            const target = byClient.get(lf.clientId)
            if (target) await sendOne(lf, target, jobNow)
          }
        }),
      )
    } catch (e) {
      const msg = e instanceof ApiError ? e.message : t('uploadFailed')
      setError(msg)
      toSend.forEach((f) => patchFile(f.clientId, { status: 'error', error: msg }))
      if (e instanceof ApiError && (e.code === 'shop_paused' || e.code === 'shop_offline')) loadShop()
    }
  }

  const removeFile = async (lf: LocalFile) => {
    setFiles((fs) => fs.filter((f) => f.clientId !== lf.clientId))
    if (job && lf.fileId) {
      try {
        setTicket(await api<Ticket>(`/jobs/${job.id}/files/${lf.fileId}`, { method: 'DELETE', secret: job.secret }))
      } catch {
        /* the draft expires anyway */
      }
    }
  }

  const retryFile = async (lf: LocalFile) => {
    if (!job) return
    await removeFile(lf)
    const dt = new DataTransfer()
    dt.items.add(lf.file)
    await onPick(dt.files)
  }

  const applySettings = async (next: FileSettings) => {
    setSettings(next)
    if (!job) return
    try {
      setTicket(await api<Ticket>(`/jobs/${job.id}`, { method: 'PATCH', body: { applyToAll: next }, secret: job.secret }))
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Could not update settings')
    }
  }

  const setPageRange = async (lf: LocalFile, range: string) => {
    patchFile(lf.clientId, { pageRange: range })
    if (!job || !lf.fileId) return
    try {
      setTicket(
        await api<Ticket>(`/jobs/${job.id}`, {
          method: 'PATCH',
          body: { files: [{ fileId: lf.fileId, settings: { ...settings, pageRange: range } }] },
          secret: job.secret,
        }),
      )
      patchFile(lf.clientId, { error: undefined })
    } catch (e) {
      patchFile(lf.clientId, { error: e instanceof ApiError ? e.message : 'Invalid pages' })
    }
  }

  const live = files.filter((f) => f.status !== 'locked' && f.status !== 'error')
  const allUploaded = live.length > 0 && live.every((f) => f.status === 'done')
  const quote = ticket?.quote

  const send = async () => {
    if (!job || !quote) return
    setSending(true)
    setError('')
    try {
      const tk = await api<Ticket>(`/jobs/${job.id}/submit`, { body: { priceVersion: quote.priceVersion, customerName: name.trim() }, secret: job.secret })
      saveTicket({ jobId: job.id, secret: job.secret, slug, shopName: shop?.name ?? '', token: tk.job.token, createdAt: Date.now() })
      navigate(`/t/${job.id}`, { replace: true })
    } catch (e) {
      if (e instanceof ApiError && e.code === 'price_changed') {
        const tk = await api<Ticket>(`/jobs/${job.id}`, { secret: job.secret })
        setTicket(tk)
      }
      setError(e instanceof ApiError ? e.message : 'Could not send. Try again.')
      setSending(false)
    }
  }

  if (shopError) {
    return (
      <Shell>
        <Card className="p-5 text-center">
          <AlertTriangle className="mx-auto mb-2 h-8 w-8 text-attention" aria-hidden />
          <p className="font-medium">{shopError}</p>
        </Card>
      </Shell>
    )
  }
  if (!shop) {
    return (
      <Shell>
        <div className="flex justify-center py-16 text-ink-muted">
          <Spinner className="h-8 w-8" />
        </div>
      </Shell>
    )
  }

  const blocked = shop.onlineState !== 'online'
  const p = shop.prices

  return (
    <Shell shop={shop}>
      {openTicket && (
        <Banner
          action={
            <Link className="font-semibold text-action underline" to={`/t/${openTicket.jobId}`}>
              {t('view')}
            </Link>
          }
        >
          {t('haveTicket', { token: openTicket.token ?? '' })}
        </Banner>
      )}

      {blocked ? (
        <Card className="p-5 text-center">
          <Clock className="mx-auto mb-2 h-8 w-8 text-attention" aria-hidden />
          <p className="text-lg font-semibold">{shop.onlineState === 'paused' ? t('shopPaused') : t('shopOffline')}</p>
          {shop.pauseMessage && <p className="mt-1 text-ink-muted">{shop.pauseMessage}</p>}
          <Button variant="secondary" className="mt-4" onClick={loadShop}>
            <RotateCw className="h-4 w-4" aria-hidden /> {t('checkAgain')}
          </Button>
        </Card>
      ) : (
        <>
          {!shop.isOpen && <Banner tone="attention">{t('shopClosed', { time: hhmm(shop.opensAt) })}</Banner>}

          <input ref={inputRef} type="file" multiple accept={ACCEPTED} className="hidden" onChange={(e) => onPick(e.target.files).finally(() => (e.target.value = ''))} />

          {files.length === 0 ? (
            <button type="button" onClick={() => inputRef.current?.click()} className="flex w-full flex-col items-center gap-2 rounded border-2 border-dashed border-action bg-surface px-4 py-10 text-center hover:bg-surface-tint">
              <span className="flex h-14 w-14 items-center justify-center rounded bg-surface-tint text-action">
                <Plus className="h-8 w-8" aria-hidden />
              </span>
              <span className="text-xl font-bold">{t('chooseFiles')}</span>
              <span className="text-sm text-ink-muted">{t('fileTypes', { n: MAX_FILES, mb: MAX_JOB_MB })}</span>
            </button>
          ) : (
            <div className="space-y-2">
              {files.map((lf) => (
                <FileRow key={lf.clientId} lf={lf} quote={quote?.lines.find((l) => l.fileId === lf.fileId)} onRemove={() => removeFile(lf)} onRetry={() => retryFile(lf)} onRange={(r) => setPageRange(lf, r)} />
              ))}
              <Button variant="secondary" className="w-full" onClick={() => inputRef.current?.click()} disabled={sending}>
                <Plus className="h-5 w-5" aria-hidden /> {t('addMore')}
              </Button>
            </div>
          )}

          {error && <Banner tone="danger">{error}</Banner>}

          {files.length > 0 && (
            <>
              <Card className="space-y-4 p-4">
                <div className="flex items-baseline justify-between">
                  <h2 className="text-lg font-bold">{t('settings')}</h2>
                  <span className="text-sm text-ink-muted">{t('forAllFiles')}</span>
                </div>
                <Segmented
                  label={t('colour')}
                  value={settings.colour}
                  onChange={(v) => applySettings({ ...settings, colour: v })}
                  options={[
                    { value: false, label: t('bw'), hint: t('perSide', { p: rupees(p.bwOnePaise) }) },
                    { value: true, label: t('colour'), hint: shop.colourAvailable ? t('perSide', { p: rupees(p.colourOnePaise) }) : '—', disabled: !shop.colourAvailable },
                  ]}
                />
                <Segmented
                  label={t('bothSides')}
                  value={settings.bothSides}
                  onChange={(v) => applySettings({ ...settings, bothSides: v })}
                  options={[
                    { value: false, label: t('oneSide') },
                    { value: true, label: t('bothSides'), hint: t('perSheet', { p: rupees(settings.colour ? p.colourBothPaise || p.colourOnePaise * 2 : p.bwBothPaise || p.bwOnePaise * 2) }) },
                  ]}
                />
                <div className="flex items-center justify-between">
                  <span className="font-semibold">{t('copies')}</span>
                  <Stepper label={t('copies')} value={settings.copies} onChange={(c) => applySettings({ ...settings, copies: c })} />
                </div>
              </Card>

              <Card className="p-4">
                <label htmlFor="name" className="block font-semibold">
                  {t('firstName')}
                </label>
                <input id="name" value={name} maxLength={20} autoComplete="given-name" onChange={(e) => setName(e.target.value)} className="mt-2 h-12 w-full rounded border-[1.5px] border-line bg-surface px-3 text-base focus:border-action focus:outline-none" placeholder="Priya" />
                <p className="mt-1 text-xs text-ink-muted">{t('firstNameHint')}</p>
              </Card>

              <Card className="p-4">
                <h2 className="mb-2 text-sm font-semibold uppercase tracking-wide text-ink-muted">{t('price')}</h2>
                {quote?.lines.map((l) => {
                  const f = files.find((x) => x.fileId === l.fileId)
                  if (!f) return null
                  return (
                    <div key={l.fileId} className="flex justify-between gap-3 py-1 text-sm">
                      <span className="truncate">
                        {f.file.name} · {l.unit === 'sheet' ? `${l.sheets} × ${rupees(l.unitPaise)}` : `${l.sides} × ${rupees(l.unitPaise)}`}
                        {l.copies > 1 ? ` × ${l.copies}` : ''}
                      </span>
                      <span className="font-mono tabular">{f.pages ? rupees(l.amountPaise) : '—'}</span>
                    </div>
                  )
                })}
                <div className="mt-2 flex items-end justify-between border-t border-divider pt-2">
                  <div>
                    <div className="font-mono text-2xl font-bold tabular">{quote ? rupees(quote.totalPaise) : '—'}</div>
                    <div className="text-xs text-ink-muted">{quote?.pagesToConfirm ? t('priceAtCounter') : t('payAtCounter')}</div>
                  </div>
                  <WaitChip shop={shop} />
                </div>
              </Card>
            </>
          )}

          <p className="flex gap-2 rounded bg-surface-tint p-3 text-sm text-ink-muted">
            <ShieldCheck className="h-5 w-5 shrink-0 text-ready" aria-hidden /> {t('privacy')}
          </p>

          {files.length > 0 && (
            <div className="sticky bottom-0 -mx-4 border-t border-line bg-canvas px-4 py-3">
              <Button size="lg" className="w-full" onClick={send} disabled={!allUploaded || sending || !quote}>
                {sending ? <Spinner /> : null}
                {sending ? t('sending') : !allUploaded ? t('uploading') : t('send')}
                {quote && allUploaded && !sending ? <span className="ml-auto rounded bg-white/15 px-2 font-mono text-sm">{rupees(quote.totalPaise)}</span> : null}
              </Button>
            </div>
          )}
        </>
      )}
    </Shell>
  )
}

function WaitChip({ shop }: { shop: PublicShop }) {
  const { t } = useI18n()
  if (shop.wait.jobsAhead === 0) return <Chip tone="ready">{t('noWait')}</Chip>
  return <Chip>{t(shop.wait.jobsAhead === 1 ? 'jobAhead1' : 'jobsAhead', { n: shop.wait.jobsAhead, low: shop.wait.lowMinutes, high: shop.wait.highMinutes })}</Chip>
}

function FileRow({ lf, quote, onRemove, onRetry, onRange }: { lf: LocalFile; quote?: { amountPaise: number }; onRemove: () => void; onRetry: () => void; onRange: (r: string) => void }) {
  const { t } = useI18n()
  const [editRange, setEditRange] = useState(false)
  const [range, setRange] = useState(lf.pageRange)
  const Icon = lf.mime.startsWith('image/') ? ImageIcon : FileText
  const bad = lf.status === 'error' || lf.status === 'locked'
  return (
    <div className={`rounded border-[1.5px] bg-surface p-3 ${bad ? 'border-danger' : 'border-line'}`}>
      <div className="flex items-center gap-3">
        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded bg-surface-tint text-action">
          <Icon className="h-5 w-5" aria-hidden />
        </span>
        <div className="min-w-0 flex-1">
          <div className="truncate font-semibold">{lf.file.name}</div>
          <div className="flex flex-wrap items-center gap-1 text-xs text-ink-muted">
            {lf.status === 'checking' ? (
              t('counting')
            ) : lf.pages > 0 ? (
              <Chip>{lf.pages === 1 ? t('page') : t('pages', { n: lf.pages })}</Chip>
            ) : lf.status !== 'locked' ? (
              <Chip tone="attention">{t('pagesUnknown')}</Chip>
            ) : null}
            {lf.pageRange && <Chip tone="action">{lf.pageRange}</Chip>}
            <span>{bytes(lf.file.size)}</span>
          </div>
        </div>
        <div className="text-right">
          {lf.status === 'done' && quote && lf.pages > 0 && <div className="font-mono font-bold tabular">{rupees(quote.amountPaise)}</div>}
          <button type="button" onClick={onRemove} className="p-2 text-ink-muted hover:text-danger" aria-label={`${t('remove')} ${lf.file.name}`}>
            <Trash2 className="h-5 w-5" aria-hidden />
          </button>
        </div>
      </div>
      {(lf.status === 'uploading' || lf.status === 'waiting') && (
        <div className="mt-2">
          <div className="h-1 overflow-hidden rounded bg-divider">
            <div className="h-full bg-action transition-all" style={{ width: `${Math.round(lf.progress * 100)}%` }} />
          </div>
          {lf.status === 'waiting' && (
            <p className="mt-1 flex items-center gap-1 text-xs text-attention">
              <WifiOff className="h-3 w-3" aria-hidden /> {t('waitingNetwork')}
            </p>
          )}
        </div>
      )}
      {bad && (
        <p className="mt-2 flex items-center justify-between gap-2 text-sm text-danger">
          <span>{lf.error}</span>
          {lf.status === 'error' && (
            <Button size="sm" variant="secondary" onClick={onRetry}>
              {t('retry')}
            </Button>
          )}
        </p>
      )}
      {lf.status === 'done' && lf.mime === 'application/pdf' && lf.pages > 1 && (
        <div className="mt-2 text-sm">
          {editRange ? (
            <form
              className="flex gap-2"
              onSubmit={(e) => {
                e.preventDefault()
                onRange(range.trim())
                setEditRange(false)
              }}
            >
              <input value={range} onChange={(e) => setRange(e.target.value)} inputMode="numeric" placeholder={t('pageRangeHint')} className="h-10 flex-1 rounded border-[1.5px] border-line bg-surface px-2" aria-label={t('pageRange')} />
              <Button size="sm" type="submit">
                OK
              </Button>
            </form>
          ) : (
            <button type="button" className="font-semibold text-action" onClick={() => setEditRange(true)}>
              {t('pageRange')}: {lf.pageRange || t('allPages')}
            </button>
          )}
          {lf.error && !bad && <p className="text-danger">{lf.error}</p>}
        </div>
      )}
    </div>
  )
}

export function Shell({ shop, children }: { shop?: PublicShop; children: React.ReactNode }) {
  const { t } = useI18n()
  return (
    <div className="mx-auto flex min-h-screen max-w-lg flex-col">
      <header className="sticky top-0 z-10 border-b border-line bg-surface px-4 py-3">
        <div className="flex items-center gap-3">
          <Logo size={40} />
          <div className="min-w-0 flex-1">
            <div className="truncate text-lg font-bold leading-tight">{shop?.name ?? 'Counter Drop'}</div>
            <div className="truncate text-xs text-ink-muted">{shop ? shop.address || t('tagline') : t('tagline')}</div>
          </div>
          <LangSwitch />
        </div>
        {shop && (
          <div className="mt-2 flex flex-wrap gap-2">
            {shop.onlineState === 'online' ? (
              <Chip tone={shop.isOpen ? 'ready' : 'attention'}>{shop.isOpen ? t('open', { time: hhmm(shop.closesAt) }) : t('closed', { time: hhmm(shop.opensAt) })}</Chip>
            ) : (
              <Chip tone="attention">{t('paused')}</Chip>
            )}
            {shop.onlineState === 'online' && <WaitChip shop={shop} />}
          </div>
        )}
      </header>
      <main className="flex-1 space-y-3 px-4 py-4">{children}</main>
    </div>
  )
}
