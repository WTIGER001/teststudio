export type StatusState = 'passed' | 'failed' | 'timed-out' | 'flaky' | 'skipped' | 'unknown' | string
export interface StatusCounts { passed: number; failed: number; timedOut: number; flaky: number; skipped: number; unknown: number }
export interface StatusTotals {
  counts: StatusCounts
  upSeconds: number
  downSeconds: number
  unknownSeconds: number
  durationSumMs: number
  durationSampleCount: number
  censoredCount: number
  durationUnavailableCount: number
}
export interface StatusAvailability { percent: number | null; coveragePercent: number; upSeconds: number; downSeconds: number; unknownSeconds: number; since: string }
export interface StatusBucket {
  since: string
  until: string
  state: StatusState
  partial: boolean
  totals: StatusTotals
  passRate: number | null
  availability: StatusAvailability | null
}
export interface StatusRow {
  title: string
  description: string
  state: StatusState
  metric?: 'availability' | 'outcomes' | string
  history: { at: string; state: StatusState }[]
  buckets?: StatusBucket[]
  totals?: StatusTotals
  passRate?: number | null
  availability: StatusAvailability
  contextCount?: number
}
export interface StatusReport {
	logo?: string
	logoAlt?: string
  title: string
  background: string
  window: string
  maxAgeSeconds: number
  asOf: string
  probes: StatusRow[]
}
export interface StatusSelection { row: StatusRow; bucket: StatusBucket }
export interface StatusSource { url: string; refreshSeconds?: number }
export interface StatusLoadError { status: number; message: string; withdrawn: boolean }
export const statusPeriods = [{ id: '24h', label: '24 hours' }, { id: '7d', label: '7 days' }, { id: '1m', label: '1 month (30 days)' }, { id: '1y', label: '1 year (365 days)' }, { id: 'all', label: 'All time' }]

export function statusLabel(state: StatusState, metric = 'availability'): string {
  if (metric === 'outcomes') return ({ passed: 'Passed', failed: 'Failed', 'timed-out': 'Timed out', flaky: 'Flaky', skipped: 'Skipped', unknown: 'Unknown or incomplete' } as Record<string, string>)[state] || 'Unknown'
  return ({ passed: 'Available', failed: 'Unavailable', 'timed-out': 'Timed out', flaky: 'Some downtime', skipped: 'Skipped', unknown: 'Unknown or stale' } as Record<string, string>)[state] || 'Unknown'
}

export function monitoredLabel(value: number): string { return value > 0 && value < .1 ? '<0.1%' : value.toFixed(1) + '%' }

export interface StatusCheck { at: string; state: StatusState; durationMs: number | null; censored: boolean; runId?: string }
export interface StatusCheckPage { items: StatusCheck[]; cursor?: string; partial: boolean }
export interface StatusCheckRange { since: string; until: string }
export interface StatusCheckQuery extends StatusCheckRange { row: number; cursor?: string; limit: number }
export type StatusCheckLoader = (query: StatusCheckQuery) => Promise<StatusCheckPage>
