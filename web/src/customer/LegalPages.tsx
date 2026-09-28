import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useI18n, type Lang } from '@/lib/i18n'
import { Card } from '@/components/ui'
import { Shell } from './DropPage'

// Who runs the service and how to reach them. Set at build time (see deploy/prod/.env.prod.example).
const OPERATOR = import.meta.env.VITE_OPERATOR_NAME || 'the operator of this Counter Drop service'
const CONTACT = import.meta.env.VITE_SUPPORT_EMAIL || ''
const UPDATED = '28 September 2026'

function Contact() {
  return CONTACT ? (
    <a className="font-semibold text-action underline" href={`mailto:${CONTACT}`}>
      {CONTACT}
    </a>
  ) : (
    <span className="font-semibold">the contact shown at the shop counter</span>
  )
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-2">
      <h2 className="text-lg font-bold">{title}</h2>
      <div className="space-y-2 text-sm leading-relaxed text-ink">{children}</div>
    </section>
  )
}

const summary: Record<Lang, string[]> = {
  en: [
    'No account, phone number or WhatsApp needed.',
    'Counter Drop deletes its copy of your files 10 minutes after pickup, or right away if you ask after pickup.',
    'If the shop downloads a file, your ticket shows it. You can ask the shop to delete it, and your receipt shows when the shop confirms.',
    'Cancel any time before printing starts; your files are deleted.',
  ],
  hi: [
    'कोई अकाउंट, फ़ोन नंबर या WhatsApp नहीं चाहिए।',
    'पिकअप के 10 मिनट बाद Counter Drop आपकी फ़ाइलों की अपनी कॉपी मिटा देता है; पिकअप के बाद आप कहें तो तुरंत।',
    'दुकान कोई फ़ाइल डाउनलोड करे तो आपके टिकट पर दिखता है। आप दुकान से उसे मिटाने को कह सकते हैं; दुकान की पुष्टि रसीद पर दिखती है।',
    'प्रिंट शुरू होने से पहले कभी भी रद्द करें; आपकी फ़ाइलें मिटा दी जाती हैं।',
  ],
  mr: [
    'अकाउंट, फोन नंबर किंवा WhatsApp लागत नाही.',
    'पिकअपनंतर 10 मिनिटांनी Counter Drop तुमच्या फाइल्सची स्वतःची कॉपी डिलीट करते; पिकअपनंतर तुम्ही सांगितल्यास लगेच.',
    'दुकानाने फाइल डाउनलोड केल्यास तुमच्या तिकिटावर दिसते. तुम्ही दुकानाला ती डिलीट करायला सांगू शकता; दुकानाची खात्री पावतीवर दिसते.',
    'प्रिंट सुरू होण्यापूर्वी कधीही रद्द करा; तुमच्या फाइल्स डिलीट होतात.',
  ],
}

export function PrivacyPage() {
  const { lang, t } = useI18n()
  return (
    <Shell>
      <h1 className="text-2xl font-bold">Privacy notice</h1>
      <p className="text-xs text-ink-muted">Last updated {UPDATED}</p>
      <Card className="p-4">
        <ul className="list-disc space-y-1 pl-5 text-sm">
          {summary[lang].map((l) => (
            <li key={l}>{l}</li>
          ))}
        </ul>
      </Card>
      <Section title="Who we are">
        <p>Counter Drop is run by {OPERATOR}. It passes the files you choose to the print shop you scanned. The shop prints them and takes your payment at the counter.</p>
      </Section>
      <Section title="What we collect">
        <ul className="list-disc space-y-1 pl-5">
          <li>The files you send and their print settings (colour, sides, copies, pages).</li>
          <li>Your first name, only if you type it, so the counter can call you.</li>
          <li>Your job record: token, pages, price, how the shop recorded your payment (cash or UPI), and times.</li>
          <li>What the shop did with each file: opened to print, or downloaded, by which staff member and when.</li>
          <li>Short-lived security logs (for example your IP address) to stop abuse. No advertising or tracking.</li>
        </ul>
        <p>We do not ask for your phone number, email address or an account.</p>
      </Section>
      <Section title="How long we keep your files">
        <ul className="list-disc space-y-1 pl-5">
          <li>Counter Drop deletes its copy 10 minutes after you collect your prints, or straight away if you ask after pickup.</li>
          <li>Cancelled jobs: deleted at once. Files you never sent: deleted after 60 minutes.</li>
          <li>Not collected: the job closes when the shop’s collection period ends (at most 7 days; your ticket shows the date) and the files are deleted.</li>
          <li>Every deletion is checked, and your receipt shows the time. After deletion we keep the job record without file names or your name.</li>
        </ul>
      </Section>
      <Section title="What the shop can do">
        <p>
          The shop can open your files to print them, or download them to its computer. <strong>Every download is shown on your ticket and receipt.</strong> Downloaded copies are the shop’s
          responsibility: our{' '}
          <Link to="/terms" className="text-action underline">
            shop terms
          </Link>{' '}
          require shops to delete them after the job and confirm it in Counter Drop. After pickup you can ask the shop to delete them, and your receipt shows when it confirms. Counter Drop cannot
          delete files from the shop’s own computers.
        </p>
      </Section>
      <Section title="On your phone">
        <p>This phone remembers the tickets you sent so you can reopen them. You can remove them from the home screen. Nothing else is stored on your phone.</p>
      </Section>
      <Section title="Who else sees your data">
        <p>
          Only the shop you chose. Our hosting and storage providers process data for us to run the service. We do not sell your data or use it for advertising. We share it only if the law requires.
        </p>
      </Section>
      <Section title="Security">
        <p>Connections are encrypted (HTTPS). Files are stored under random codes, not their names. Links to files expire within minutes. Shop staff sign in with personal PINs.</p>
      </Section>
      <Section title="Your rights and complaints">
        <p>
          You can cancel before printing starts and ask for deletion after pickup from your ticket. You can also ask us what data we hold about you, or to correct or erase it, under India’s Digital
          Personal Data Protection Act, 2023. Write to <Contact /> with your shop name and token.
        </p>
      </Section>
      <Section title="Changes">
        <p>If this notice changes, we will update this page and the date above.</p>
      </Section>
      <p className="pb-6 text-center text-sm">
        <Link to="/terms" className="font-semibold text-action underline">
          {t('termsLink')}
        </Link>
      </p>
    </Shell>
  )
}

export function TermsPage() {
  const { t } = useI18n()
  return (
    <Shell>
      <h1 className="text-2xl font-bold">Terms of use</h1>
      <p className="text-xs text-ink-muted">
        Last updated {UPDATED} · Counter Drop is run by {OPERATOR}.
      </p>
      <Section title="For customers">
        <ul className="list-disc space-y-1 pl-5">
          <li>Send only files you are allowed to print. Nothing unlawful.</li>
          <li>You see the price before you send. The shop charges its displayed prices and you pay at the counter.</li>
          <li>You can cancel until printing starts. After that the job belongs to the shop, which may still cancel it with a reason (for example, a file that won’t print).</li>
          <li>Collect by the date on your ticket. Uncollected jobs close automatically and their files are deleted.</li>
          <li>Print quality, payment and refunds are between you and the shop. Counter Drop records what happened to help resolve problems.</li>
        </ul>
      </Section>
      <Section title="For shops">
        <ul className="list-disc space-y-1 pl-5">
          <li>Use a customer’s files only to do that customer’s job. Don’t keep, share or reuse them.</li>
          <li>
            Prefer <strong>Print</strong> (opens in the browser) over <strong>Download</strong>. Customers are told about every download.
          </li>
          <li>
            If you download a file, delete every copy after the job — Downloads folder, Recycle Bin and any other device — then tap <strong>Copies deleted</strong>. Confirm only when it is true;
            customers see it on their receipt.
          </li>
          <li>When a customer asks for deletion, act on it promptly.</li>
          <li>Keep PINs personal. Remove staff who leave from Settings → Staff.</li>
          <li>Turn on your photocopier’s automatic data erase if it has one.</li>
          <li>Charge the prices shown in Counter Drop.</li>
          <li>Counter Drop records file opens, downloads and deletion confirmations and shows them to customers. Misuse can lead to suspension.</li>
        </ul>
      </Section>
      <Section title="Questions">
        <p>
          Write to <Contact />. See also the{' '}
          <Link to="/privacy" className="text-action underline">
            privacy notice
          </Link>
          .
        </p>
      </Section>
      <p className="pb-6 text-center text-sm">
        <Link to="/privacy" className="font-semibold text-action underline">
          {t('privacyLink')}
        </Link>
      </p>
    </Shell>
  )
}

/** Small footer with the two legal links, for customer and shop screens. */
export function LegalLinks({ className = '' }: { className?: string }) {
  const { t } = useI18n()
  return (
    <p className={`space-x-3 text-center text-xs text-ink-muted ${className}`}>
      <Link to="/privacy" className="underline">
        {t('privacyLink')}
      </Link>
      <Link to="/terms" className="underline">
        {t('termsLink')}
      </Link>
    </p>
  )
}
