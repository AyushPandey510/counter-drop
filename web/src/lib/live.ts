import { useEffect, useRef } from 'react'
import { API } from './api'

export interface LiveEvent {
  type: string
  data?: unknown
}

/** What the API returns from POST /jobs/{id}/live or /shop/live. */
export interface LiveTicket {
  mode: 'sse' | 'ws'
  url?: string // empty with mode "ws": this API's own /ws (local mode)
  ticket?: string
}

export interface LiveSource {
  /** Server-Sent Events path (used when the API says mode "sse"). */
  sse: string
  /** Asks the API for a live-update ticket (authenticated like any other call). */
  ticket: () => Promise<LiveTicket>
}

const HEARTBEAT_MS = 5 * 60 * 1000 // API Gateway closes WebSockets idle for 10 minutes
const MAX_BACKOFF_MS = 30_000

function wsURL(t: LiveTicket): string {
  const base = t.url || new URL(API + '/ws', window.location.href).toString().replace(/^http/, 'ws')
  return `${base}${base.includes('?') ? '&' : '?'}t=${encodeURIComponent(t.ticket ?? '')}`
}

/**
 * Live updates for a ticket page or shop board. It asks the API how to connect:
 * - "sse": a Server-Sent Events stream (the browser reconnects on its own);
 * - "ws": a WebSocket (API Gateway on AWS) with a fresh ticket on every (re)connect, backoff, and a
 *   heartbeat to stay under the idle limit. Connections also close after 2 hours on AWS; we reconnect.
 * onEvent runs for each message; onReconnect refetches a snapshot after a gap so nothing is missed.
 * A slow fallback poll covers networks that block both.
 */
export function useLive(source: LiveSource | null, onEvent: (e: LiveEvent) => void, onReconnect: () => void, pollMs = 15000) {
  const handler = useRef(onEvent)
  const resync = useRef(onReconnect)
  const src = useRef(source)
  handler.current = onEvent
  resync.current = onReconnect
  src.current = source
  const key = source?.sse ?? null

  useEffect(() => {
    if (!key) return
    let stopped = false
    let open = false
    let everOpened = false
    let attempt = 0
    let es: EventSource | null = null
    let ws: WebSocket | null = null
    let heartbeat = 0
    let retry = 0

    const deliver = (raw: string) => {
      try {
        const ev = JSON.parse(raw) as LiveEvent
        if (ev.type === 'hello') return // SSE greeting; handled by onOpen
        handler.current(ev)
      } catch {
        /* ignore malformed */
      }
    }
    const onOpen = () => {
      open = true
      attempt = 0
      if (everOpened) resync.current() // we were disconnected: catch up
      everOpened = true
    }
    const scheduleRetry = () => {
      if (stopped) return
      const delay = Math.min(MAX_BACKOFF_MS, 1000 * 2 ** attempt) * (0.75 + Math.random() * 0.5)
      attempt++
      window.clearTimeout(retry)
      retry = window.setTimeout(connect, delay)
    }

    const openSSE = () => {
      es = new EventSource(API + key)
      es.onmessage = (m) => {
        try {
          if ((JSON.parse(m.data) as LiveEvent).type === 'hello') return onOpen()
        } catch {
          /* fall through */
        }
        deliver(m.data)
      }
      es.onerror = () => {
        open = false // EventSource retries by itself
      }
    }

    const openWS = (t: LiveTicket) => {
      const sock = new WebSocket(wsURL(t))
      ws = sock
      sock.onopen = () => {
        onOpen()
        window.clearInterval(heartbeat)
        heartbeat = window.setInterval(() => {
          if (sock.readyState === WebSocket.OPEN) sock.send('{"type":"ping"}')
        }, HEARTBEAT_MS)
      }
      sock.onmessage = (m) => deliver(typeof m.data === 'string' ? m.data : '')
      sock.onclose = () => {
        open = false
        window.clearInterval(heartbeat)
        if (ws === sock) scheduleRetry()
      }
    }

    async function connect() {
      if (stopped) return
      const s = src.current
      if (!s) return
      let t: LiveTicket
      try {
        t = await s.ticket()
      } catch {
        return scheduleRetry()
      }
      if (stopped) return
      if (t.mode === 'ws' && t.ticket) openWS(t)
      else openSSE()
    }
    connect()

    const poll = window.setInterval(() => {
      if (!open) resync.current()
    }, pollMs)
    const onVisible = () => {
      if (document.visibilityState !== 'visible') return
      resync.current()
      // Phones suspend sockets in the background: reconnect straight away instead of waiting out the backoff.
      if (ws && ws.readyState !== WebSocket.OPEN && ws.readyState !== WebSocket.CONNECTING) {
        attempt = 0
        window.clearTimeout(retry)
        connect()
      }
    }
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      stopped = true
      es?.close()
      const sock = ws
      ws = null
      sock?.close()
      window.clearInterval(heartbeat)
      window.clearTimeout(retry)
      window.clearInterval(poll)
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [key, pollMs])
}

/** A short two-tone chime made with Web Audio (no sound files to download). */
export function chime(kind: 'soft' | 'ready' = 'ready') {
  try {
    const Ctx = window.AudioContext ?? (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext
    const ctx = new Ctx()
    const notes = kind === 'ready' ? [880, 1320] : [660]
    notes.forEach((f, i) => {
      const o = ctx.createOscillator()
      const g = ctx.createGain()
      o.frequency.value = f
      o.type = 'sine'
      const t = ctx.currentTime + i * 0.18
      g.gain.setValueAtTime(0.0001, t)
      g.gain.exponentialRampToValueAtTime(kind === 'ready' ? 0.35 : 0.15, t + 0.02)
      g.gain.exponentialRampToValueAtTime(0.0001, t + 0.35)
      o.connect(g).connect(ctx.destination)
      o.start(t)
      o.stop(t + 0.4)
    })
    window.setTimeout(() => ctx.close(), 1200)
  } catch {
    /* audio not available */
  }
}
