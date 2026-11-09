import { ProbeBulkService } from './probe-bulk.service'
import { DestroyRef, effect, inject, Injectable, signal, untracked } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute, Router } from '@angular/router'
import { combineLatest } from 'rxjs'
import type { Application } from '../configuration/configuration.service'
import { ApiError, ApiService, type Page, type RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { randomId } from '../../core/random-id'
import type { ProbePage, ProbeSummary, ProbeBinding } from './probe.models'

@Injectable()
export class ProbeOverviewService {
  readonly workspace = inject(WorkspaceService)
  private readonly api = inject(ApiService)
  private readonly bulk = inject(ProbeBulkService)
  private readonly route = inject(ActivatedRoute)
  private readonly router = inject(Router)
  readonly project = signal('')
  readonly applications = signal<RecordValue<Application>[]>([])
  readonly page = signal<ProbePage | null>(null)
  readonly loading = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly action = signal<ProbeSummary | null>(null)
  readonly kind = signal<'run' | 'archive' | 'restore'>('run')
  readonly busy = signal(false)
  readonly ready = signal(false)
  readonly filters = new FormGroup({ search: new FormControl('', { nonNullable: true }), application: new FormControl('', { nonNullable: true }), stage: new FormControl('', { nonNullable: true }), state: new FormControl('', { nonNullable: true }), freshness: new FormControl('', { nonNullable: true }), schedule: new FormControl('', { nonNullable: true }), hours: new FormControl('24', { nonNullable: true }), archived: new FormControl(false, { nonNullable: true }) })
  private generation = 0
  private key = ''
  private request: Record<string, unknown> = {}
  constructor() {
    const destroy = inject(DestroyRef)
    combineLatest([this.route.paramMap, this.route.queryParamMap]).pipe(takeUntilDestroyed(destroy)).subscribe(([p, q]) => {
      if(this.project() !== (p.get('projectId') ?? '')){this.bulk.selected.set([]);this.bulk.rows.set([]);this.bulk.opened.set(false)}
      this.generation++; this.project.set(p.get('projectId') ?? ''); this.page.set(null); this.action.set(null)
      this.filters.setValue({ search: q.get('search') ?? '', application: q.get('application') ?? '', stage: q.get('stage') ?? '', state: q.get('state') ?? '', freshness: q.get('freshness') ?? '', schedule: q.get('schedule') ?? '', hours: q.get('hours') ?? '24', archived: q.get('archived') === 'true' })
      void this.load(); void this.loadApplications()
    })
    effect(() => { const project=this.project(), caps=this.workspace.capabilities(); if(project && Object.keys(caps).length && this.workspace.can('projects.read')) untracked(() => { void this.loadApplications() }) })
    destroy.onDestroy(() => { this.generation++ })
  }
  private async loadApplications(): Promise<void> { const project=this.project(); this.applications.set([]); if(!this.workspace.can('projects.read'))return; try { const page=await this.api.request<Page<RecordValue<Application>>>(`/api/v1/projects/${project}/applications`); if(project===this.project())this.applications.set(page.items) }catch{} }
  base(): string { return `/api/v1/projects/${encodeURIComponent(this.project())}/probes` }
  path(...suffix: string[]): string[] { return ['/projects', this.project(), ...suffix] }
  async apply(): Promise<void> {
    const values = this.filters.getRawValue()
    await this.router.navigate([], { relativeTo: this.route, queryParams: Object.fromEntries(Object.entries(values).filter(([,v]) => !!v)) })
  }
  async load(more = false): Promise<void> {
    const generation = ++this.generation
    this.loading.set(true); this.error.set('')
    const q = new URLSearchParams(Object.entries(this.filters.getRawValue()).filter(([,v]) => v !== '' && v !== false).map(([k,v]) => [k, String(v)]))
    q.set('limit', '30'); if (more && this.page()?.cursor) q.set('cursor', this.page()!.cursor!)
    try {
      const result = await this.api.request<ProbePage>(`${this.base()}?${q}`)
      if (generation === this.generation) this.page.set(more && this.page() ? { ...result, items: [...this.page()!.items, ...result.items] } : result)
    } catch (e) { if (generation === this.generation) { this.page.set(null); if(e instanceof ApiError && [401,403].includes(e.status)){this.bulk.selected.set([]);this.bulk.rows.set([]);this.bulk.opened.set(false)} this.error.set(e instanceof Error ? e.message : 'Probe results are unavailable.') } }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
  async review(row: ProbeSummary, kind: 'run' | 'archive' | 'restore'): Promise<void> {
    this.action.set(row); this.kind.set(kind); this.key = randomId(); this.ready.set(false); this.error.set('')
    if (kind !== 'run') { this.ready.set(true); return }
    if (!row.activeReleaseId) { this.error.set('Save the test and wait for setup first, or use Save and run from test creation.'); return }
    this.request = { suiteId: row.suiteId, environmentId: row.environmentId, testIds: [row.testId], executionLocation: row.executionLocation, releaseId: row.activeReleaseId, timeoutSeconds: 60, retries: 0 }
    this.busy.set(true)
    try { await this.api.request(`/api/v1/projects/${this.project()}/runs/estimate`, 'POST', this.request); this.ready.set(true) }
    catch (e) { this.error.set(e instanceof Error ? e.message : 'The run selection is unavailable.') }
    finally { this.busy.set(false) }
  }
  async confirm(): Promise<void> {
    const row = this.action(); if (!row || this.busy() || !this.ready()) return
    this.busy.set(true); this.error.set('')
    try {
      if (this.kind() === 'run') {
        const run = await this.api.request<{id: string}>(`/api/v1/projects/${this.project()}/runs`, 'POST', this.request, undefined, this.key)
        await this.router.navigate(this.path('runs', run.id), { queryParams: { test: row.testId } })
      } else {
        const binding: ProbeBinding = { id: row.id, name: row.name, suiteId: row.suiteId, testId: row.testId, environmentId: row.environmentId, endpoint: row.endpoint, executionLocation: row.executionLocation, archived: this.kind() === 'archive' }
        await this.api.request(`${this.base()}/${row.id}`, 'PUT', binding, row.revision)
        this.message.set(binding.archived ? 'Probe archived. Tests, schedules, and saved history remain available.' : 'Probe restored. Schedules keep their existing state.'); this.action.set(null); await this.load()
      }
    } catch (e) {
      this.error.set(e instanceof Error ? e.message : 'Action failed. Retry on this page with the same request identity. After reload, inspect stored state before trying again.')
      if (e instanceof ApiError && [401,403,409].includes(e.status)) { this.ready.set(false); if (e.status !== 409) { this.page.set(null); this.action.set(null) } }
    } finally { this.busy.set(false) }
  }
}
