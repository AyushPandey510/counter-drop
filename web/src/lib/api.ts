// Small typed client for /api/v1/cd. Responses are { data } or { error: { code, message } }.

export const API = (import.meta.env.VITE_API_BASE_URL ?? '') + '/api/v1/cd'

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message)
  }
}

export interface RequestOptions {
  method?: string
  body?: unknown
  secret?: string
  token?: string
  signal?: AbortSignal
}

export async function api<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = {}
  if (opts.body !== undefined) headers['Content-Type'] = 'application/json'
  if (opts.secret) headers['X-Ticket-Secret'] = opts.secret
  if (opts.token) headers['Authorization'] = `Bearer ${opts.token}`
  let res: Response
  try {
    res = await fetch(API + path, {
      method: opts.method ?? (opts.body !== undefined ? 'POST' : 'GET'),
      headers,
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
      signal: opts.signal,
    })
  } catch (e) {
    if ((e as Error).name === 'AbortError') throw e
    throw new ApiError(0, 'network', 'No connection. Check your internet and try again.')
  }
  let payload: { data?: T; error?: { code: string; message: string } } = {}
  try {
    payload = await res.json()
  } catch {
    /* empty body */
  }
  if (!res.ok || payload.error) {
    throw new ApiError(res.status, payload.error?.code ?? 'http_' + res.status, payload.error?.message ?? 'Something went wrong.')
  }
  return payload.data as T
}

// Safe storage: private mode or blocked storage must never break the app.
export const store = {
  get(key: string): string | null {
    try {
      return localStorage.getItem(key)
    } catch {
      return null
    }
  },
  set(key: string, value: string) {
    try {
      localStorage.setItem(key, value)
    } catch {
      /* ignore */
    }
  },
  remove(key: string) {
    try {
      localStorage.removeItem(key)
    } catch {
      /* ignore */
    }
  },
}
