import { useEffect } from 'react'
import { Delete } from 'lucide-react'
import { Button, Spinner } from './ui'

/** 4-dot PIN display plus a 0–9 keypad. Physical keyboards work too (digits, Backspace). */
export function PinPad({ value, onDigit, onDelete, busy = false, label = 'PIN' }: { value: string; onDigit: (d: string) => void; onDelete: () => void; busy?: boolean; label?: string }) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (busy || e.ctrlKey || e.metaKey || e.altKey) return
      if (/^\d$/.test(e.key)) onDigit(e.key)
      else if (e.key === 'Backspace') onDelete()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [busy, onDigit, onDelete])

  return (
    <div>
      <div className="my-4 flex justify-center gap-3" role="status" aria-live="polite" aria-label={`${label}: ${value.length} of 4 digits entered`}>
        {[0, 1, 2, 3].map((i) => (
          <span key={i} className={`h-4 w-4 rounded-full ${i < value.length ? 'bg-ink' : 'border-2 border-line'}`} />
        ))}
      </div>
      {busy ? (
        <div className="flex justify-center py-8">
          <Spinner className="h-8 w-8" />
        </div>
      ) : (
        <div className="grid grid-cols-3 gap-2">
          {['1', '2', '3', '4', '5', '6', '7', '8', '9'].map((d) => (
            <Button key={d} variant="secondary" size="lg" className="font-mono text-2xl" onClick={() => onDigit(d)}>
              {d}
            </Button>
          ))}
          <span />
          <Button variant="secondary" size="lg" className="font-mono text-2xl" onClick={() => onDigit('0')}>
            0
          </Button>
          <Button variant="ghost" size="lg" aria-label="Delete digit" onClick={onDelete}>
            <Delete className="h-6 w-6" aria-hidden />
          </Button>
        </div>
      )}
    </div>
  )
}
