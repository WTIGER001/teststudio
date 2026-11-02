import type { TimingEstimate, TimingPolicy, TimingPrediction } from './timing.models'
import type { Run } from '../runs/run.models'

export function defaultTimingPolicy(): TimingPolicy {
  return { mode: 'fail', slowdownFactor: 2, minimumIncreaseMs: 100, minimumSamples: 5, window: 50, maxAgeDays: 30, timeoutMode: 'adaptive', timeoutFactor: 3, timeoutMarginMs: 250, minimumTimeoutMs: 1000 }
}
export function predictionRows(estimate: TimingEstimate | undefined): TimingPrediction[] {
  return Object.values(estimate?.tests ?? {}).sort((a, b) => a.context.testId.localeCompare(b.context.testId))
}
export function timingPolicyIssue(policy: TimingPolicy): string {
  if (policy.window < policy.minimumSamples) return 'The sample window must contain at least the minimum number of observations.'
  return ''
}
export function liveTiming(run: Run | null, now: number): { elapsedMs: number | null; remainingMs: number | null; overdue: boolean } {
  if (!run || !run.startedAt) return { elapsedMs: null, remainingMs: null, overdue: false }
  const end = run.finishedAt ? Date.parse(run.finishedAt) : now
  const elapsedMs = Math.max(0, end - Date.parse(run.startedAt))
  if (run.finishedAt) return { elapsedMs, remainingMs: null, overdue: false }
  const completed = new Set(run.results.map(result => result.testId))
  const remaining = run.tests.filter(test => !completed.has(test.id))
  let remainingMs = 0
  for (const test of remaining) {
    const prediction = run.timing?.tests[test.id]
    if (prediction?.expectedMs == null) return { elapsedMs, remainingMs: null, overdue: false }
    remainingMs += prediction.expectedMs
  }
  const current = run.currentTestId && !completed.has(run.currentTestId) ? run.timing?.tests[run.currentTestId] : undefined
  const currentElapsed = current && run.currentTestStartedAt ? Math.max(0, now - Date.parse(run.currentTestStartedAt)) : 0
  const overdue = current?.expectedMs != null && currentElapsed > current.expectedMs
  if (overdue) return { elapsedMs, remainingMs: null, overdue: true }
  return { elapsedMs, remainingMs: Math.max(0, remainingMs - currentElapsed), overdue: false }
}
