export interface TrendCounts { total: number; passed: number; failed: number; timedOut: number; flaky: number; skipped: number; unknown: number }
export type TrendState = Exclude<keyof TrendCounts, 'total'>
export interface TrendPoint { runId: string; at: string; suiteId: string; environmentId?: string; executionLocation: string; state: string; outcome: string; counts: TrendCounts; gap?: string }
export interface TrendRanking { suiteId: string; testId: string; title: string; category: string; counts: TrendCounts; comparisons: number; changes: number; volatility?: number }
export interface TrendFilters { from: string; to: string; suite: string; environment: string; category: string; test: string; location: string }
export interface TrendReport {
  projectId: string; filter: { since: string; until: string; suite?: string; environment?: string; category?: string; test?: string; location?: string }
  points: TrendPoint[]; totals: TrendCounts; sampleCount: number; contextRuns: number; cursor?: string; partial: boolean
  failureRankings: TrendRanking[]; volatilityRankings: TrendRanking[]
  options: { suites: { id: string; name: string }[]; environments: { id: string; name: string }[]; categories: string[]; locations: string[]; partial: boolean }
}
