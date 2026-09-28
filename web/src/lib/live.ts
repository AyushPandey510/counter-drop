import { useEffect, useRef } from 'react'
import { API } from './api'

export interface LiveEvent {
  type: string
  data?: unknown
}

/**
 * Subscribes to a Server-Sent Events stream and calls onEvent for each message. The browser
 * reconnects on its own; onReconnect lets the caller refetch a snapshot so no update is missed.
 * A slow fallback poll covers networks that block streaming.
 */
export function useLive(path: string | null, onEvent: (e: LiveEvent) => void, onReconnect: () => void, pollMs = 15000) {
  const handler = useRef(onEvent)
  const resync = useRef(onReconnect)
  handler.current = onEvent
  resync.current = onReconnect

  useEffect(() => {
    if (!path) return
    let opened = false
    const es = new EventSource(API + path)
    es.onmessage = (m) => {
      try {
        const ev = JSON.parse(m.data) as LiveEvent
        if (ev.type === 'hello') {
          if (opened) resync.current()
          opened = true
          return
        }
        handler.current(ev)
      } catch {
        /* ignore malformed */
      }
    }
    const poll = window.setInterval(() => {
      if (es.readyState !== EventSource.OPEN) resync.current()
    }, pollMs)
    const onVisible = () => {
      if (document.visibilityState === 'visible') resync.current()
    }
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      es.close()
      window.clearInterval(poll)
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [path, pollMs])
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
