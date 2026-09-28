import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import QRCode from 'qrcode'
import { ArrowLeft, Printer } from 'lucide-react'
import { Button, Logo } from '@/components/ui'
import { useShopAuth } from './auth'

// Printable A4 counter poster (FSD SCR-S08). Always printed light, whatever the board theme.
export default function QrPage() {
  const { session } = useShopAuth()
  const [svg, setSvg] = useState('')
  const url = session ? `${window.location.origin}/s/${session.shop.slug}` : ''

  useEffect(() => {
    if (!url) return
    QRCode.toString(url, { type: 'svg', errorCorrectionLevel: 'M', margin: 1, color: { dark: '#111111', light: '#ffffff' } }).then(setSvg)
  }, [url])

  if (!session) return null
  const { shop } = session

  return (
    <div className="min-h-screen bg-canvas" data-theme="light">
      <div className="mx-auto flex max-w-3xl items-center gap-3 px-4 py-4 print:hidden">
        <Link to="/shop" className="rounded p-2 hover:bg-surface-tint" aria-label="Back to queue">
          <ArrowLeft className="h-5 w-5" aria-hidden />
        </Link>
        <h1 className="flex-1 text-xl font-bold">Counter QR poster</h1>
        <Button onClick={() => window.print()}>
          <Printer className="h-4 w-4" aria-hidden /> Print
        </Button>
      </div>

      <article className="poster mx-auto flex aspect-[210/297] w-full max-w-[210mm] flex-col items-center justify-between border border-line bg-white p-[12mm] text-center text-[#111] print:border-0">
        <header className="flex flex-col items-center gap-2">
          <Logo size={56} />
          <h2 className="text-[34px] font-bold leading-tight">{shop.name}</h2>
          {shop.address && <p className="text-lg text-[#555]">{shop.address}</p>}
        </header>

        <div>
          <p className="text-[40px] font-bold leading-tight">Scan to send files for printing</p>
          <p className="mt-1 text-[28px] font-semibold">प्रिंट के लिए स्कैन करें · प्रिंटसाठी स्कॅन करा</p>
        </div>

        <div className="w-[62%]" aria-label={`QR code for ${url}`} dangerouslySetInnerHTML={{ __html: svg }} />

        <ol className="grid w-full grid-cols-3 gap-4 text-left text-base">
          <li>
            <b className="block text-lg">1 · Scan</b>No app, no WhatsApp, no phone number
            <span className="block text-sm text-[#555]">कोई ऐप नहीं, कोई WhatsApp नहीं</span>
          </li>
          <li>
            <b className="block text-lg">2 · Choose files</b>See the price before you send
            <span className="block text-sm text-[#555]">भेजने से पहले कीमत देखें</span>
          </li>
          <li>
            <b className="block text-lg">3 · Get a token</b>Pay at the counter · cash or UPI
            <span className="block text-sm text-[#555]">काउंटर पर भुगतान करें</span>
          </li>
        </ol>

        <footer className="w-full border-t border-[#ddd] pt-3 text-sm text-[#555]">
          Files are deleted automatically 10 minutes after pickup · <span className="font-mono">{url.replace(/^https?:\/\//, '')}</span>
        </footer>
      </article>
    </div>
  )
}
