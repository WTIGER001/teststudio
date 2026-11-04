import { computed, DestroyRef, effect, inject, Injectable, signal, untracked } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiService, type Page, type RecordValue } from '../../core/api.service'
import type { RetainedTest, TestHistoryItem } from '../../core/generated/access.types'
import { WorkspaceService } from '../../core/workspace.service'
import type { TestRevision, CatalogRevision } from '../catalog/catalog-operations.models'
import { newestRevisions } from '../catalog/catalog-operations.utils'

@Injectable()
export class TestHistoryService {
  private readonly client = inject(AccessClientService)
  private readonly api = inject(ApiService)
  readonly workspace = inject(WorkspaceService)
  readonly project = signal('')
  readonly suite = signal('')
  readonly test = signal('')
  readonly returnRun = signal('')
  readonly context = signal<RetainedTest | null>(null)
  readonly runs = signal<TestHistoryItem[]>([])
  readonly cursor = signal('')
  readonly partial = signal(false)
  readonly loading = signal(false)
  readonly runLoading = signal(false)
  readonly error = signal('')
  readonly runError = signal('')
  readonly revisions = signal<CatalogRevision[]>([])
  readonly revisionLoading = signal(false)
  readonly revisionError = signal('')
  readonly runAccess = computed(() => this.workspace.can('runs.read'))
  private generation = 0
  private historyGeneration = 0
  private runQueryStarted = false
  private revisionGeneration = 0
  private revisionsLoaded = false

  constructor() {
    const route = inject(ActivatedRoute), destroy = inject(DestroyRef)
    route.paramMap.pipe(takeUntilDestroyed(destroy)).subscribe(params => {
      this.generation++; this.historyGeneration++; this.revisionGeneration++; this.runLoading.set(false); this.runQueryStarted = false; this.context.set(null); this.runs.set([]); this.revisions.set([]); this.cursor.set(''); this.error.set(''); this.runError.set(''); this.revisionError.set(''); this.revisionsLoaded = false
      this.project.set(params.get('projectId') ?? ''); this.suite.set(params.get('suiteId') ?? ''); this.test.set(params.get('testId') ?? ''); this.returnRun.set(route.snapshot.queryParamMap.get('run') ?? '')
      void this.load()
    })
    effect(() => {
      const context = this.context(), allowed = this.runAccess(), known = Object.keys(this.workspace.capabilities()).length > 0
      untracked(() => {
        if (context && allowed && !this.runQueryStarted) { this.runQueryStarted = true; void this.loadRuns() }
        if (known && !allowed) { this.historyGeneration++; this.runs.set([]); this.cursor.set(''); this.runLoading.set(false); this.runQueryStarted = false }
      })
    })
    destroy.onDestroy(() => { this.generation++; this.historyGeneration++; this.revisionGeneration++ })
  }

  async refreshAccess(): Promise<void> { this.context.set(null); await this.workspace.refreshAccess(); await this.load() }

  private scope() { return { project: this.project(), suite: this.suite(), test: this.test() } }

  async load(): Promise<void> {
    const generation = ++this.generation
    this.historyGeneration++; this.runLoading.set(false); this.runQueryStarted = false
    this.loading.set(true); this.error.set(''); this.context.set(null); this.runs.set([]); this.cursor.set(''); this.runError.set(''); this.partial.set(false)
    this.revisionGeneration++; this.revisions.set([]); this.revisionsLoaded = false; this.revisionError.set(''); this.revisionLoading.set(false)
    try {
      const value = await this.client.call('readRetainedTest', this.scope())
      if (generation !== this.generation) return
      this.context.set(value)
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Retained test context is unavailable. Refresh access and retry.') }
    finally { if (generation === this.generation) this.loading.set(false) }
  }

  async loadRuns(more = false): Promise<void> {
    if (!this.context() || this.runLoading() || more && !this.cursor()) return
    const generation = ++this.historyGeneration
    this.runQueryStarted = true; this.runLoading.set(true); this.runError.set('')
    try {
      const page = await this.client.call('readTestRunHistory', { ...this.scope(), query: { limit: 20, cursor: more ? this.cursor() : undefined } })
      if (generation !== this.historyGeneration) return
      this.runs.set(more ? [...this.runs(), ...page.items] : page.items); this.cursor.set(page.cursor ?? ''); this.partial.set(page.partial)
    } catch (error) { if (generation === this.historyGeneration) { this.runs.set([]); this.cursor.set(''); this.runError.set(error instanceof Error ? error.message : 'Run history is unavailable. Refresh access and retry.') } }
    finally { if (generation === this.historyGeneration) this.runLoading.set(false) }
  }

  async loadRevisions(open: boolean): Promise<void> {
    if (!open || this.revisionsLoaded || this.revisionLoading() || !this.context()) return
    const generation = ++this.revisionGeneration
    this.revisionLoading.set(true); this.revisionError.set('')
    try {
      const scope = this.scope()
      const page = await this.api.request<Page<RecordValue<TestRevision>>>(`/api/v1/projects/${encodeURIComponent(scope.project)}/suites/${encodeURIComponent(scope.suite)}/tests/${encodeURIComponent(scope.test)}/revisions`)
      if (generation === this.revisionGeneration) { this.revisions.set(newestRevisions(page.items)); this.revisionsLoaded = true }
    } catch (error) { if (generation === this.revisionGeneration) { this.revisions.set([]); this.revisionError.set(error instanceof Error ? error.message : 'Revision history is unavailable.') } }
    finally { if (generation === this.revisionGeneration) this.revisionLoading.set(false) }
  }
}
