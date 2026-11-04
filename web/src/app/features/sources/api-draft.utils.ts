import type { PlaywrightDraft } from '../../core/generated/access.types'

export interface APIStarterOptions {
  title: string; id: string; method: string; path: string; endpoint: string; prefix: string
  status: number; readOnly: boolean; body: string; maxBodyBytes: number; maxRequests: number; maxResponseBytes: number
}

export function apiBodyMethod(method: string): boolean { return ['POST', 'PUT', 'PATCH', 'DELETE'].includes(method) }
export function apiReadOnlyMethod(method: string): boolean { return ['GET', 'HEAD', 'OPTIONS', 'POST'].includes(method) }
export function apiPathPrefix(value: string): string { const path = value.split('?')[0]; return path.length > 1 ? path.replace(/\/$/, '') : path }

function canonicalAPIPath(value: string, trailing = false): boolean {
  if (!value.startsWith('/') || value.startsWith('//') || /[\\%?#\u0000-\u0020\u007f]/.test(value) || value.includes('//') || value.length > 2048) return false
  if (value !== '/' && value.endsWith('/') && !trailing) return false
  return !value.split('/').some(part => part === '.' || part === '..')
}

export function apiStarterProblem(value: APIStarterOptions, step?: number): { field: keyof APIStarterOptions; message: string } | undefined {
  const issue = (field: keyof APIStarterOptions, message: string) => ({ field, message })
  if (step !== 1 && (!value.title.trim() || /[\u0000-\u001f\u007f]/.test(value.title) || value.title.trim().length > 120)) return issue('title', 'Enter an API test title of 1 through 120 characters.')
  if (step !== 1 && !/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(value.id)) return issue('id', 'Enter a valid permanent identifier under Advanced details.')
  if (step === 0) return undefined
  if (!['GET', 'HEAD', 'OPTIONS', 'POST', 'PUT', 'PATCH', 'DELETE'].includes(value.method)) return issue('method', 'Choose a supported HTTP request method.')
  const query = value.path.indexOf('?'), pathname = query < 0 ? value.path : value.path.slice(0, query)
  if (!canonicalAPIPath(pathname, true) || /[#\u0000-\u0020\u007f]/.test(value.path) || value.path.length - pathname.length > 4097) return issue('path', 'Enter a relative path without encoded paths, dot segments, fragments, repeated slashes, or spaces.')
  if (!canonicalAPIPath(value.prefix) || !(pathname === value.prefix || value.prefix === '/' || pathname.startsWith(value.prefix + '/'))) return issue('prefix', 'The allowed path prefix must be canonical and include the request path.')
  if (!/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(value.endpoint)) return issue('endpoint', 'Enter a valid registered endpoint name under Advanced request policy.')
  if (!Number.isInteger(value.status) || value.status < 100 || value.status > 599) return issue('status', 'Enter an expected response status from 100 through 599.')
  if (!Number.isInteger(value.maxRequests) || value.maxRequests < 1 || value.maxRequests > 1000) return issue('maxRequests', 'Set a request budget from 1 through 1000.')
  if (!Number.isInteger(value.maxResponseBytes) || value.maxResponseBytes < 1 || value.maxResponseBytes > 4194304) return issue('maxResponseBytes', 'Set a response limit within 4 MiB.')
  if (!Number.isInteger(value.maxBodyBytes) || value.maxBodyBytes < 0 || value.maxBodyBytes > 1048576) return issue('maxBodyBytes', 'Set a request body limit within 1 MiB.')
  if (value.readOnly && !apiReadOnlyMethod(value.method)) return issue('readOnly', 'Mutating methods cannot be reviewed as read-only.')
  if (!apiBodyMethod(value.method) && value.body.trim()) return issue('body', 'This request method cannot carry a body. Clear the JSON body or choose another method.')
  if (value.body.trim()) {
    try { const body = JSON.stringify(JSON.parse(value.body)); if (new TextEncoder().encode(body).length > value.maxBodyBytes) return issue('body', 'The JSON body exceeds the reviewed request body limit.') }
    catch { return issue('body', 'Enter valid JSON request data.') }
  }
  return undefined
}

export function apiStarterIssue(value: APIStarterOptions, step?: number): string { return apiStarterProblem(value, step)?.message ?? '' }

export function configureAPIStarter(starter: PlaywrightDraft, value: APIStarterOptions): PlaywrightDraft {
  const issue = apiStarterIssue(value)
  if (issue) throw new Error(issue)
  const manifestFile = starter.files.find(file => file.path === starter.manifestPath)
  const catalogFile = starter.files.find(file => file.path === 'catalog.json')
  if (!manifestFile || !catalogFile) throw new Error('The API starter workspace is unavailable. Refresh and try again.')
  const manifest = JSON.parse(manifestFile.content), catalog = JSON.parse(catalogFile.content)
  if (manifest.type !== 'api' || !manifest.browser || catalog.tests?.length !== 1) throw new Error('The API starter workspace has an unsupported structure.')
  manifest.browser = { endpoint: value.endpoint, requests: { rules: [{ method: value.method, pathPrefix: value.prefix, readOnly: value.readOnly, maxBodyBytes: apiBodyMethod(value.method) ? value.maxBodyBytes : 0 }], maxRequests: value.maxRequests, maxResponseBytes: value.maxResponseBytes } }
  const definition = catalog.tests[0]
  definition.id = value.id; definition.title = value.title.trim(); definition.selector = '@' + value.id
  const options = { method: value.method, maxRedirects: 0, ...(value.body.trim() ? { data: JSON.stringify(JSON.parse(value.body)), headers: { 'Content-Type': 'application/json' } } : {}) }
  const code = `import { test, expect } from '@playwright/test'\n\ntest(${JSON.stringify(value.title.trim())}, { tag: ${JSON.stringify('@' + value.id)} }, async ({ request }) => {\n  await test.step(${JSON.stringify(value.method + ' ' + apiPathPrefix(value.path))}, async () => {\n    const response = await request.fetch(${JSON.stringify(value.path)}, ${JSON.stringify(options)})\n    expect(response.status()).toBe(${value.status})\n  })\n})\n`
  return { ...starter, name: value.title.trim(), files: starter.files.map(file => file.path === starter.manifestPath ? { ...file, content: JSON.stringify(manifest, null, 2) + '\n' } : file.path === 'catalog.json' ? { ...file, content: JSON.stringify(catalog, null, 2) + '\n' } : file.path === 'checks/api.spec.ts' ? { ...file, content: code } : file) }
}
