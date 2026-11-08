import { isPlatformBrowser } from '@angular/common'
import { HttpBackend, HttpClient, HttpErrorResponse } from '@angular/common/http'
import { inject, Injectable, PLATFORM_ID } from '@angular/core'
import { catchError, finalize, map, Observable, of, shareReplay, startWith, switchMap, timer } from 'rxjs'
import type { StatusLoadError, StatusReport, StatusSource } from './models'

export interface StatusLoadState { report: StatusReport | null; loading: boolean; error: StatusLoadError | null }

@Injectable({ providedIn: 'root' })
export class TestStudioStatusClient {
  private readonly backend = inject(HttpBackend, { optional: true })
  private readonly http = this.backend ? new HttpClient(this.backend) : null
  private readonly browser = isPlatformBrowser(inject(PLATFORM_ID))
  private readonly requests = new Map<string, Observable<StatusLoadState>>()
  publicURL(baseURL: string, page: string): string { return baseURL.replace(/\/$/, '') + '/api/v1/public/status-pages/' + encodeURIComponent(page) }
  watch(source: StatusSource, period = ''): Observable<StatusLoadState> {
    if (!this.browser) return of({ report: null, loading: false, error: null })
    let url: string
    try {
      const parsed = new URL(source.url, 'http://teststudio.invalid')
      if (!['http:', 'https:'].includes(parsed.protocol) || parsed.username || parsed.password || !/\/api\/v1\/public\/status-pages\/[a-zA-Z0-9-]+$/.test(parsed.pathname)) throw new Error('Use a public TestStudio status report URL. Supply private report data through the host application.')
      if (period) parsed.searchParams.set('window', period)
      url = parsed.origin === 'http://teststudio.invalid' ? parsed.pathname + parsed.search : parsed.toString()
    } catch (error) { return of({ report: null, loading: false, error: { status: 0, withdrawn: false, message: error instanceof Error ? error.message : 'Invalid report URL.' } }) }
    if (!this.http) return of({ report: null, loading: false, error: { status: 0, withdrawn: false, message: 'Provide Angular HttpClient or supply report data directly.' } })
    const seconds = source.refreshSeconds === 0 ? 0 : Math.min(3600, Math.max(15, source.refreshSeconds || 60))
    const key = JSON.stringify([url, seconds]), existing = this.requests.get(key)
    if (existing) return existing
    const ticks = seconds ? timer(0, seconds * 1000) : of(0)
    const request = ticks.pipe(switchMap(() => this.http!.get<StatusReport>(url, { withCredentials: false, credentials: 'omit' }).pipe(
      map(report => ({ report, loading: false, error: null } as StatusLoadState)),
      catchError((error: HttpErrorResponse) => of<StatusLoadState>({ report: null, loading: false, error: { status: error.status, withdrawn: [401, 403, 404, 410].includes(error.status), message: [401, 403, 404, 410].includes(error.status) ? 'This status report is unavailable or no longer published.' : 'Status information is temporarily unavailable.' } })),
      startWith<StatusLoadState>({ report: null, loading: true, error: null })
    )), finalize(() => { if (this.requests.get(key) === request) this.requests.delete(key) }), shareReplay({ bufferSize: 1, refCount: true }))
    this.requests.set(key, request)
    return request
  }
}
