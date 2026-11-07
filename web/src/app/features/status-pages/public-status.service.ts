import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { ActivatedRoute, Router } from '@angular/router'
import { combineLatest } from 'rxjs'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import type { StatusPageReport } from '../../core/generated/access.types'

@Injectable()
export class PublicStatusService {
  private readonly route = inject(ActivatedRoute); private readonly router = inject(Router); private readonly destroy = inject(DestroyRef)
  readonly report = signal<StatusPageReport | null>(null); readonly error = signal(''); readonly busy = signal(false)
  private token = ''; private window = ''; private request?: AbortController; private timer?: ReturnType<typeof setTimeout>; private generation = 0
  constructor() {
    combineLatest([this.route.paramMap, this.route.queryParamMap]).pipe(takeUntilDestroyed(this.destroy)).subscribe(([params, query]) => { this.token = params.get('token') || ''; this.window = query.get('window') || ''; void this.refresh() })
    this.destroy.onDestroy(() => { this.generation++; this.request?.abort(); clearTimeout(this.timer) })
  }
  select(window: string): void { void this.router.navigate([], { relativeTo: this.route, queryParams: { window }, replaceUrl: true }) }
  async refresh(): Promise<void> {
    this.request?.abort(); clearTimeout(this.timer); const generation = ++this.generation; this.request = new AbortController(); this.busy.set(true); this.error.set('')
    try {
      const response = await fetch('/api/v1/public/status-pages/' + encodeURIComponent(this.token) + (this.window ? '?window=' + encodeURIComponent(this.window) : ''), { credentials: 'omit', cache: 'no-store', signal: this.request.signal })
      if (!response.ok) throw new Error(response.status === 404 ? 'This status page is unavailable or no longer published.' : 'Status information is temporarily unavailable. Try again shortly.')
      const report = await response.json() as StatusPageReport
      if (generation === this.generation) this.report.set(report)
    } catch (error) { if (generation === this.generation) { this.report.set(null); this.error.set(error instanceof Error ? error.message : 'Status information is unavailable.') } }
    finally { if (generation === this.generation) { this.busy.set(false); this.timer = setTimeout(() => { void this.refresh() }, 60000) } }
  }
}
