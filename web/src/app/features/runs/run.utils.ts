import type { Run, RunRequest } from './run.models'

export function terminal(state: string): boolean {
  return ['completed', 'cancelled', 'timed-out', 'blocked', 'incomplete'].includes(state)
}

export function canRerun(run: Run): boolean { return terminal(run.state) && run.results.some(r => ['failed', 'timed-out', 'flaky'].includes(r.state)) }
export function requestFromForm(value: { suite: string; environment: string; tests: string; timeout: number; retries: number }, preview = ''): RunRequest {
  return { suiteId: value.suite, environmentId: value.environment || undefined, ...(preview ? { savedSelectionPreviewId: preview } : { testIds: value.tests.split(/[\s,]+/).filter(Boolean) }), executionLocation: 'local', timeoutSeconds: value.timeout, retries: value.retries }
}

export function queryString(values: Record<string, string>): string {
  return new URLSearchParams(Object.entries(values).filter(([, value]) => value !== '')).toString()
}

export function errorMessage(error: unknown, fallback: string): string { return error instanceof Error ? error.message : fallback }
export function previewKind(contentType: string): 'text' | 'image' | 'video' | 'download' {
  const kind = contentType.split(';')[0].trim().toLowerCase()
  if (['text/plain', 'application/json'].includes(kind)) return 'text'
  if (['image/png', 'image/jpeg', 'image/webp', 'image/gif'].includes(kind)) return 'image'
  if (['video/mp4', 'video/webm'].includes(kind)) return 'video'
  return 'download'
}

export function registeredEndpoints(endpoints?: Record<string, string>): { name: string; address: string }[] { return Object.entries(endpoints ?? {}).map(([name, address]) => ({ name, address })) }
