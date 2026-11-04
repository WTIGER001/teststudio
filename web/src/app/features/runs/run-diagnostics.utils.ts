import type { RecordValue } from '../../core/api.service'
import type { Attempt, Run, TestStep } from './run.models'

export function runTestLabels(run: Run | null): Record<string, string> {
  return Object.fromEntries((run?.tests ?? []).map(test => [test.id, test.title || test.id]))
}

export function diagnosticTests(run: Run | null): { id: string; label: string }[] {
  const states = new Map((run?.results ?? []).map(result => [result.testId, result.state]))
  return (run?.tests ?? []).map(test => ({ id: test.id, label: `${test.title || test.id} · ${test.id}${states.has(test.id) ? ' · ' + states.get(test.id) : ''}` }))
}

export function diagnosticAttempts(run: Run | null, test: string, rows: RecordValue<Attempt>[]): number[] {
  if (!test) return []
  const observed = rows.filter(row => row.data.testId === test).map(row => row.data.attempt)
  const final = run?.results.find(result => result.testId === test)?.attempt ?? 0
  const total = Math.min(3, Math.max((run?.request.retries ?? 0) + 1, final, ...observed))
  return Array.from({ length: total }, (_, index) => index + 1)
}

export function failedSteps(steps: TestStep[]): number { return steps.filter(step => !!step.error).length }

export function runActionFeedback(action: string, status: number, message: string): string {
  const name = action === 'rerun-failures' ? 'rerun this run' : 'cancel this run'
  if (status === 403) return `Your current permissions do not allow you to ${name}. The saved run and its results remain unchanged. Refresh access or contact the project administrator.`
  if (action === 'rerun-failures' && (status === 0 || status >= 500)) return 'The rerun response was not received. Retry Rerun failures on this page to reuse the same request and avoid a duplicate run. The original results remain available.'
  return `${message || 'The run action failed.'} The saved run and its results remain available.`
}

export function resultSummaries(run: Run | null): Record<string, string> {
  return Object.fromEntries((run?.results ?? []).map(result => { const first = (result.error || result.failureKind || '').split('\n').find(line => line.trim())?.trim() ?? ''; return [result.testId, first.length > 220 ? first.slice(0, 217) + '…' : first] }))
}
