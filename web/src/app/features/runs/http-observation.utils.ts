import type { HTTPObservation } from './run.models'

export function httpTimingRows(http: HTTPObservation): { label: string; milliseconds: number | undefined }[] {
  return [
    { key: 'dns', label: 'DNS lookup' }, { key: 'connect', label: 'Connection' },
    { key: 'tls', label: 'TLS handshake' }, { key: 'firstByte', label: 'Time to first byte' }
  ].map(phase => {
    const value = http.timingsMs?.[phase.key]
    return { label: phase.label, milliseconds: value != null && Number.isFinite(value) && value >= 0 ? value : undefined }
  })
}

export function certificateObservation(http: HTTPObservation, observedAt: string): { expiresAt?: string; remainingDays?: number } {
  if (!http.certificateExpires) return {}
  const expiry = Date.parse(http.certificateExpires), observation = Date.parse(observedAt)
  if (!Number.isFinite(expiry)) return {}
  return { expiresAt: http.certificateExpires, remainingDays: Number.isFinite(observation) ? (expiry - observation) / 86400000 : undefined }
}
