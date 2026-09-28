import { useState } from 'react'
import { Link } from 'react-router-dom'
import { QrCode, Ticket as TicketIcon, Trash2 } from 'lucide-react'
import { useI18n } from '@/lib/i18n'
import { forgetTicket, savedTickets } from '@/lib/tickets'
import { Card } from '@/components/ui'
import { Shell } from './DropPage'

// Landing page for people who open the app without scanning a shop QR (installed PWA, shared link).
export default function Home() {
  const { t } = useI18n()
  const [tickets, setTickets] = useState(savedTickets)

  return (
    <Shell>
      <section className="rounded border-2 border-ink bg-ink p-5 text-canvas">
        <h1 className="text-2xl font-bold">{t('homeTitle')}</h1>
        <p className="mt-2 text-sm opacity-90">{t('homeBody')}</p>
      </section>

      {tickets.length > 0 && (
        <Card className="divide-y divide-line">
          {tickets.map((tk) => (
            <div key={tk.jobId} className="flex items-center gap-3 p-3">
              <TicketIcon className="h-5 w-5 text-action" aria-hidden />
              <Link to={`/t/${tk.jobId}`} className="min-w-0 flex-1">
                <div className="font-mono text-lg font-bold">{tk.token ?? '—'}</div>
                <div className="truncate text-sm text-ink-muted">{tk.shopName}</div>
              </Link>
              <button
                type="button"
                aria-label={t('remove')}
                className="rounded p-2 text-ink-muted hover:bg-surface-tint"
                onClick={() => {
                  forgetTicket(tk.jobId)
                  setTickets(savedTickets())
                }}
              >
                <Trash2 className="h-4 w-4" aria-hidden />
              </button>
            </div>
          ))}
        </Card>
      )}

      <Card className="flex items-start gap-3 p-4">
        <QrCode className="mt-0.5 h-6 w-6 shrink-0 text-action" aria-hidden />
        <p className="text-sm">{t('scanHint')}</p>
      </Card>
      <p className="text-sm text-ink-muted">{t('privacy')}</p>

      <div className="pt-6 text-center">
        <Link to="/shop/login" className="text-sm font-semibold text-action underline">
          {t('staffLogin')}
        </Link>
      </div>
    </Shell>
  )
}
