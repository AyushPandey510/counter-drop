import type { ReactNode } from 'react'
import { Clock3, MonitorSmartphone, ShieldCheck } from 'lucide-react'
import { Logo } from '@/components/ui'

// Sign-in and setup screens. Phones: a single centred column. Wide screens: a brand panel on the left.
export function AuthLayout({ subtitle, children }: { subtitle: string; children: ReactNode }) {
  const points = [
    { Icon: MonitorSmartphone, title: 'One live queue', body: 'Every customer file arrives on this board with a token. No WhatsApp, no pen drives.' },
    { Icon: Clock3, title: 'Four taps per job', body: 'Claim, print, ready, collected. Customers see their status change on their phone.' },
    { Icon: ShieldCheck, title: 'Files delete themselves', body: '10 minutes after pickup. Your PIN is known only to you.' },
  ]
  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      <aside className="hidden flex-col justify-between bg-ink p-12 text-canvas lg:flex">
        <div className="flex items-center gap-3">
          <Logo size={48} />
          <span className="text-2xl font-bold">Counter Drop</span>
        </div>
        <div className="max-w-md space-y-8">
          <h2 className="text-4xl font-bold leading-tight">Run the print counter from one screen.</h2>
          <ul className="space-y-5">
            {points.map(({ Icon, title, body }) => (
              <li key={title} className="flex gap-4">
                <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded bg-white/10">
                  <Icon className="h-5 w-5" aria-hidden />
                </span>
                <span>
                  <span className="block font-semibold">{title}</span>
                  <span className="block text-sm opacity-80">{body}</span>
                </span>
              </li>
            ))}
          </ul>
        </div>
        <p className="text-xs opacity-60">Need help? Contact Counter Drop support.</p>
      </aside>
      <main className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center gap-4 px-4 py-8 lg:max-w-lg lg:px-10">
        <div className="flex items-center gap-3 lg:hidden">
          <Logo size={48} />
          <div>
            <h1 className="text-2xl font-bold">Counter Drop</h1>
            <p className="text-sm text-ink-muted">{subtitle}</p>
          </div>
        </div>
        <h1 className="hidden text-3xl font-bold lg:block">{subtitle}</h1>
        {children}
      </main>
    </div>
  )
}
