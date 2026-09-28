import { Download, Share, X } from 'lucide-react'
import { useI18n } from '@/lib/i18n'
import { useInstall } from '@/lib/install'
import { Button } from './ui'

/** Offers to install the PWA. Never blocks anything: upload always works without installing. */
export function InstallCard({ compact = false }: { compact?: boolean }) {
  const { t } = useI18n()
  const { mode, install, dismiss } = useInstall()
  if (mode === 'none') return null

  if (compact) {
    return (
      <div className="flex items-center gap-2 rounded border-[1.5px] border-line bg-surface p-2 pl-3 text-sm">
        <Download className="h-4 w-4 shrink-0 text-action" aria-hidden />
        <span className="flex-1">{mode === 'ios' ? t('installIOS') : t('installShort')}</span>
        {mode === 'prompt' && (
          <Button size="sm" onClick={install}>
            {t('installBtn')}
          </Button>
        )}
        <button type="button" onClick={dismiss} className="rounded p-1 text-ink-muted hover:bg-surface-tint" aria-label={t('notNow')}>
          <X className="h-4 w-4" aria-hidden />
        </button>
      </div>
    )
  }

  return (
    <section className="rounded border-[1.5px] border-action bg-surface-tint p-4" aria-label={t('installTitle')}>
      <div className="flex items-start gap-3">
        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded bg-action text-white">
          <Download className="h-5 w-5" aria-hidden />
        </span>
        <div className="min-w-0 flex-1">
          <h2 className="font-bold">{t('installTitle')}</h2>
          <p className="mt-1 text-sm text-ink-muted">{t('installBody')}</p>
          {mode === 'ios' && (
            <p className="mt-2 flex items-center gap-2 text-sm font-semibold">
              <Share className="h-4 w-4 text-action" aria-hidden /> {t('installIOS')}
            </p>
          )}
          <div className="mt-3 flex gap-2">
            {mode === 'prompt' && <Button onClick={install}>{t('installBtn')}</Button>}
            <Button variant="ghost" onClick={dismiss}>
              {t('notNow')}
            </Button>
          </div>
        </div>
      </div>
    </section>
  )
}
