import type { HTTPCheck, JSONValue } from './catalog.models'

export interface HTTPCheckForm {
  endpoint: string; path: string; method: string; statusCodes: string; headers: string; json: string
  requiredText: string; forbiddenText: string; maxResponseMs: number; minCertificateDays: number
}
export type HTTPCheckErrors = Partial<Record<keyof HTTPCheckForm, string>>
const size = (value: string) => new TextEncoder().encode(value).length

export function httpCheckForm(check?: HTTPCheck): HTTPCheckForm {
  return {
    endpoint: check?.endpoint ?? 'api', path: check?.path ?? '/healthz', method: check?.method ?? 'GET', statusCodes: (check?.statusCodes ?? [200]).join(', '),
    headers: check?.headers ? JSON.stringify(check.headers, null, 2) : '', json: check?.json ? JSON.stringify(check.json, null, 2) : '',
    requiredText: check?.requiredText ?? '', forbiddenText: check?.forbiddenText ?? '', maxResponseMs: check?.maxResponseMs ?? 0, minCertificateDays: check?.minCertificateDays ?? 0
  }
}

function objectField(value: string): Record<string, JSONValue> | undefined {
  if (!value.trim()) return {}
  try {
    const parsed: JSONValue = JSON.parse(value)
    return parsed !== null && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : undefined
  } catch { return undefined }
}

export function parseHTTPCheck(value: HTTPCheckForm): { check?: HTTPCheck; errors: HTTPCheckErrors } {
  const errors: HTTPCheckErrors = {}
  if (!/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(value.endpoint)) errors.endpoint = 'Use a registered endpoint identifier with letters, numbers, periods, underscores, or hyphens.'
  if (!value.path.startsWith('/') || value.path.startsWith('//') || value.path.includes('#') || /[\u0000-\u001f\u007f]/.test(value.path) || /%(?![a-f\d]{2})/i.test(value.path)) errors.path = 'Use a relative path beginning with one slash, valid percent escapes, and no fragment.'
  if (value.method !== 'GET' && value.method !== 'HEAD') errors.method = 'Choose GET or HEAD.'
  const tokens = value.statusCodes.trim().split(/[\s,]+/).filter(Boolean)
  if (!tokens.length || tokens.length > 32 || tokens.some(token => !/^\d{3}$/.test(token) || Number(token) < 100 || Number(token) > 599)) errors.statusCodes = 'Enter 1 through 32 HTTP status codes from 100 through 599, separated by commas or spaces.'
  if (!Number.isInteger(value.maxResponseMs) || value.maxResponseMs < 0 || value.maxResponseMs > 3600000) errors.maxResponseMs = 'Use a whole number from 0 through 3600000 milliseconds.'
  if (!Number.isInteger(value.minCertificateDays) || value.minCertificateDays < 0 || value.minCertificateDays > 36500) errors.minCertificateDays = 'Use a whole number from 0 through 36500 days.'
  if (size(value.requiredText) > 8192) errors.requiredText = 'Required response text must fit within 8192 bytes.'
  if (size(value.forbiddenText) > 8192) errors.forbiddenText = 'Forbidden response text must fit within 8192 bytes.'
  const headers = objectField(value.headers)
  if (!headers) errors.headers = 'Enter a JSON object that maps response header names to expected string values.'
  else {
    const names = new Set<string>()
    if (Object.keys(headers).length > 32) errors.headers = 'Configure at most 32 expected response headers.'
    for (const [name, expected] of Object.entries(headers)) {
      const key = name.toLowerCase()
      if (!/^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/.test(name) || size(name) > 256 || typeof expected !== 'string' || size(expected) > 8192 || /[\r\n\u0000]/.test(expected) || names.has(key)) errors.headers = 'Use unique valid header names and single-line string values within 8192 bytes.'
      names.add(key)
    }
  }
  const json = objectField(value.json)
  if (!json) errors.json = 'Enter a JSON object that maps dot-separated field paths to expected JSON values.'
  else if (Object.keys(json).length > 32 || size(JSON.stringify(json)) > 65536 || Object.keys(json).some(path => !path || size(path) > 256 || /[\r\n\u0000]/.test(path) || path.startsWith('.') || path.endsWith('.') || path.includes('..'))) errors.json = 'Use at most 32 nonempty dot-separated paths within 256 bytes each and 65536 bytes of JSON.'
  if (value.method === 'HEAD' && (value.requiredText !== '' || value.forbiddenText !== '' || (json && Object.keys(json).length > 0))) errors.method = 'HEAD cannot assert a response body. Clear body assertions or choose GET.'
  if (Object.keys(errors).length) return { errors }
  return { errors, check: {
    endpoint: value.endpoint, path: value.path, method: value.method, statusCodes: [...new Set(tokens.map(Number))],
    headers: Object.keys(headers!).length ? headers as Record<string, string> : undefined, json: Object.keys(json!).length ? json : undefined,
    requiredText: value.requiredText || undefined, forbiddenText: value.forbiddenText || undefined,
    maxResponseMs: value.maxResponseMs, minCertificateDays: value.minCertificateDays
  } }
}
