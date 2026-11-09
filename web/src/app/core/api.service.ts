import { Injectable, signal } from '@angular/core'

export interface RecordValue<T> { savedVersionId?: string; id: string; revision: number; data: T; updatedAt: string; projectId?: string }
export interface Page<T> { items: T[]; cursor?: string }

export class ApiError extends Error {
  constructor(readonly status: number, message: string) { super(message) }
}

@Injectable({ providedIn: 'root' })
export class ApiService {
  readonly csrf = signal('')
  onUnauthenticated: ((csrf: string) => void) | undefined

  async request<T>(path: string, method = 'GET', value?: unknown, revision?: number, key?: string, responseType: 'json' | 'blob' = 'json'): Promise<T> {
    const csrf = this.csrf()
    const headers: Record<string, string> = { Accept: 'application/json' }
    const binary = value instanceof Blob
    if (value !== undefined) headers['Content-Type'] = binary ? 'application/zip' : 'application/json'
    if (method !== 'GET') headers['X-CSRF-Token'] = csrf
    if (revision !== undefined) headers['If-Match'] = `"${revision}"`
    if (key) headers['Idempotency-Key'] = key
    const response = await fetch(path, { method, headers, credentials: 'same-origin', body: value === undefined ? undefined : binary ? value : JSON.stringify(value) })
    if (response.status === 401 && path !== '/auth/password') this.onUnauthenticated?.(csrf)
    if (response.ok && responseType === 'blob') return await response.blob() as T
    let data
    try { data = await response.json() }
    catch { throw new ApiError(response.ok ? 502 : response.status, response.ok ? 'The server returned an unreadable response. Refresh state before retrying a change.' : 'The request was rejected with an unreadable response. Refresh state to review current access.') }
    if (!response.ok) {
      throw new ApiError(response.status, data.message || 'The request failed.')
    }
    return data as T
  }
}
