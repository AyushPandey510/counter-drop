import type { UploadTarget } from './types'

/**
 * PUTs a file to its presigned URL with progress. Network drops are retried with backoff
 * (1, 2, 4 … up to 15 s) and wait for the browser to come back online.
 */
export async function uploadFile(file: File, target: UploadTarget, onProgress: (fraction: number) => void, onWaiting: (waiting: boolean) => void): Promise<void> {
  let attempt = 0
  for (;;) {
    try {
      await putOnce(file, target, onProgress)
      onWaiting(false)
      return
    } catch (e) {
      const status = (e as { status?: number }).status ?? 0
      if (status >= 400 && status < 500) throw e // signature expired or mismatch: caller decides
      attempt++
      if (attempt > 8) throw e
      onWaiting(true)
      if (!navigator.onLine) await waitOnline()
      await sleep(Math.min(15000, 1000 * 2 ** (attempt - 1)))
    }
  }
}

function putOnce(file: File, target: UploadTarget, onProgress: (f: number) => void): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('PUT', target.url)
    for (const [k, v] of Object.entries(target.headers)) xhr.setRequestHeader(k, v)
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded / e.total)
    }
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        onProgress(1)
        resolve()
      } else reject(Object.assign(new Error('upload failed'), { status: xhr.status }))
    }
    xhr.onerror = () => reject(Object.assign(new Error('network'), { status: 0 }))
    xhr.send(file)
  })
}

function waitOnline(): Promise<void> {
  return new Promise((resolve) => {
    const on = () => {
      window.removeEventListener('online', on)
      resolve()
    }
    window.addEventListener('online', on)
  })
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

/** Browsers sometimes report an empty MIME type (HEIC on Android). */
export function mimeOf(file: File): string {
  if (file.type) return file.type
  const ext = file.name.split('.').pop()?.toLowerCase()
  return ({ pdf: 'application/pdf', jpg: 'image/jpeg', jpeg: 'image/jpeg', png: 'image/png', heic: 'image/heic', webp: 'image/webp' } as Record<string, string>)[ext ?? ''] ?? 'application/octet-stream'
}

export const ACCEPTED = '.pdf,.jpg,.jpeg,.png,.heic,.webp,application/pdf,image/jpeg,image/png,image/heic,image/webp'
export const ALLOWED_MIMES = new Set(['application/pdf', 'image/jpeg', 'image/png', 'image/heic', 'image/webp'])
