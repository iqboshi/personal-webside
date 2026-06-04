import type { Signal, SiteData } from '../types'

export async function fetchProfile(): Promise<SiteData> {
  const response = await fetch('/api/profile')
  if (!response.ok) {
    throw new Error(`Profile API failed: ${response.status}`)
  }
  return response.json()
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

