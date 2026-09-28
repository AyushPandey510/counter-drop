import { store } from './api'

// Tickets this phone has sent, so a customer can reopen them (FS-4.2). Secrets never leave the device
// except in the X-Ticket-Secret header.

export interface SavedTicket {
  jobId: string
  secret: string
  slug: string
  shopName: string
  token?: string
  createdAt: number
}

const KEY = 'cd.tickets'

export function savedTickets(): SavedTicket[] {
  try {
    const list = JSON.parse(store.get(KEY) ?? '[]') as SavedTicket[]
    const dayAgo = Date.now() - 24 * 3600 * 1000
    return list.filter((t) => t.createdAt > dayAgo)
  } catch {
    return []
  }
}

export function saveTicket(t: SavedTicket) {
  const rest = savedTickets().filter((x) => x.jobId !== t.jobId)
  store.set(KEY, JSON.stringify([t, ...rest].slice(0, 10)))
}

export function findTicket(jobId: string): SavedTicket | undefined {
  return savedTickets().find((t) => t.jobId === jobId)
}

export function forgetTicket(jobId: string) {
  store.set(KEY, JSON.stringify(savedTickets().filter((t) => t.jobId !== jobId)))
}
