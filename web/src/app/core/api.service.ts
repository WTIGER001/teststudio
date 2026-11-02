import { Injectable, signal } from '@angular/core'

export interface RecordValue<T> { id: string; revision: number; data: T; updatedAt: string; projectId?: string }
export interface Page<T> { items: T[]; cursor?: string }

export class ApiError extends Error {
  constructor(readonly status: number, message: string) { super(message) }
}

@Injectable({ providedIn: 'root' })
export class ApiService {
  readonly csrf = signal('')
  onUnauthenticated: (() => void) | undefined

  async request<T>(path: string, method = 'GET', value?: unknown, revision?: number, key?: string): Promise<T> {
    const headers: Record<string, string> = { Accept: 'application/json' }
    if (value !== undefined) headers['Content-Type'] = 'application/json'
    if (method !== 'GET') headers['X-CSRF-Token'] = this.csrf()
    if (revision !== undefined) headers['If-Match'] = `"${revision}"`
    if (key) headers['Idempotency-Key'] = key
    const response = await fetch(path, { method, headers, credentials: 'same-origin', body: value === undefined ? undefined : JSON.stringify(value) })
    const data = await response.json().catch(() => ({ message: 'The server returned an unreadable response.' }))
    if (!response.ok) {
      if (response.status === 401) this.onUnauthenticated?.()
      throw new ApiError(response.status, data.message || 'The request failed.')
    }
    return data as T
  }
}
