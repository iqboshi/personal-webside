import { staticAssetPath } from '../url'

const rawApiBaseUrl = (import.meta.env.VITE_API_BASE_URL || '').trim().replace(/\/$/, '')

export class HttpError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'HttpError'
    this.status = status
  }
}

export function apiURL(path: string): string {
  if (!path.startsWith('/api/')) return path
  return rawApiBaseUrl ? `${rawApiBaseUrl}${path}` : path
}

export async function requestJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const requestUrl = apiURL(url)
  const response = await fetch(requestUrl, {
    ...init,
    credentials: url.startsWith('/api/') ? 'include' : init?.credentials,
  })
  if (!response.ok) {
    throw new HttpError(`${requestUrl} failed: ${response.status}`, response.status)
  }
  return response.json()
}

export async function requestWithStaticFallback<T>(apiPath: string, staticPath: string): Promise<T> {
  try {
    return await requestJSON<T>(apiPath)
  } catch (apiError) {
    try {
      return await requestJSON<T>(staticAssetPath(staticPath))
    } catch {
      throw apiError
    }
  }
}
