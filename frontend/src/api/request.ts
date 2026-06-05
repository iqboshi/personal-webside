import { staticAssetPath } from '../url'

export class HttpError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'HttpError'
    this.status = status
  }
}

export async function requestJSON<T>(url: string): Promise<T> {
  const response = await fetch(url)
  if (!response.ok) {
    throw new HttpError(`${url} failed: ${response.status}`, response.status)
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
