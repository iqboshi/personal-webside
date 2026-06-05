import type { Signal, SiteData } from '../types'
import { requestWithStaticFallback } from './request'

export async function fetchProfile(): Promise<SiteData> {
  return requestWithStaticFallback<SiteData>('/api/profile', 'data/profile.json')
}

export function subscribeSignals(onSignal: (signal: Signal) => void): () => void {
  const source = new EventSource('/api/stream')

  source.addEventListener('signal', (event) => {
    onSignal(JSON.parse((event as MessageEvent).data))
  })

  source.onerror = () => {
    source.close()
  }

  return () => source.close()
}
