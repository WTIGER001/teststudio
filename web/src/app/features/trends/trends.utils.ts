import type { TrendFilters, TrendRanking } from './trends.models'

export function defaultTrendFilters(now: number): TrendFilters {
  return { from: new Date(now - 29 * 86400000).toISOString().slice(0, 10), to: new Date(now).toISOString().slice(0, 10), suite: '', environment: '', category: '', test: '', location: '' }
}

export function readTrendFilters(query: Record<string, unknown>, now: number): TrendFilters {
  const result = defaultTrendFilters(now)
  for (const key of Object.keys(result) as (keyof TrendFilters)[]) if (typeof query[key] === 'string') result[key] = query[key] as string
  return result
}

function validDate(value: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false
  const date = new Date(`${value}T00:00:00.000Z`)
  return Number.isFinite(date.getTime()) && date.toISOString().slice(0, 10) === value
}

export function trendFilterIssue(filters: TrendFilters): string {
  if (!validDate(filters.from) || !validDate(filters.to)) return 'Choose valid start and end dates.'
  const span = Date.parse(`${filters.to}T23:59:59.999Z`) - Date.parse(`${filters.from}T00:00:00.000Z`)
  if (span < 0 || span > 366 * 86400000) return 'Choose an ordered period of no more than 366 days.'
  if (filters.test && !filters.suite) return 'Choose a suite before filtering by a permanent test identifier.'
  return ''
}

export function trendRequest(filters: TrendFilters, cursor = ''): string {
  return new URLSearchParams({ since: `${filters.from}T00:00:00.000Z`, until: `${filters.to}T23:59:59.999Z`, suite: filters.suite, environment: filters.environment, category: filters.category, test: filters.test, location: filters.location, cursor, limit: '200' }).toString()
}

export function trendTitle(filters: TrendFilters): string {
  if (filters.test) return `Test ${filters.suite} / ${filters.test}`
  if (filters.category) return `Category: ${filters.category}`
  return 'Overall result states'
}

export function rankingQuery(filters: TrendFilters, rank: TrendRanking, category = false): Record<string, string> {
  return { ...filters, suite: rank.suiteId, test: category ? '' : rank.testId, category: category ? rank.category : '' }
}

export function rankingRows(filters: TrendFilters, ranks: TrendRanking[]) {
  return ranks.map(rank => ({ key: `${rank.suiteId}/${rank.testId}`, rank, testQuery: rankingQuery(filters, rank), categoryQuery: rankingQuery(filters, rank, true) }))
}
