import { randomId } from '../../core/random-id'
import { liveTiming } from '../timing/timing.utils'
import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute, Router } from '@angular/router'
import { ApiError, ApiService, type RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { RunEvidenceService } from './run-evidence.service'
import type { Run } from './run.models'
import { historyEvidenceTest, diagnosticTests, runTestLabels, runActionFeedback, resultSummaries } from './run-diagnostics.utils'
import { runApplications } from './run-application.utils'
import { canRerun, errorMessage, terminal } from './run.utils'

@Injectable()
export class RunDetailService {
  private readonly api = inject(ApiService)
  private readonly route = inject(ActivatedRoute)
  private readonly router = inject(Router)
  private readonly destroy = inject(DestroyRef)
  readonly workspace = inject(WorkspaceService)
  readonly evidence = inject(RunEvidenceService)
  readonly projectId = signal('')
  readonly runId = signal('')
  readonly run = signal<Run | null>(null)
  readonly loading = signal(false)
  readonly mutating = signal(false)
  readonly error = signal('')
  readonly actionError = signal('')
  readonly refreshingAccess = signal(false)
  readonly shared = computed(() => !!this.run() && runApplications(this.run()!).length > 1)
  readonly testLabels = computed(() => runTestLabels(this.run()))
  readonly testChoices = computed(() => diagnosticTests(this.run()))
  readonly resultSummaries = computed(() => resultSummaries(this.run()))
  readonly canCancel = computed(() => { const run = this.run(); return !!run && !terminal(run.state) && !run.cancelRequested && this.workspace.can('runs.cancel') })
  readonly canRerun = computed(() => { const run = this.run(); return !!run && canRerun(run) && this.workspace.can('runs.create') })
  readonly finished = computed(() => { const run = this.run(); return !!run && terminal(run.state) })
  readonly observedAt = signal(Date.now())
  readonly timingProgress = computed(() => liveTiming(this.run(), this.observedAt()))
  private historyScopeApplied = false
  private generation = 0
  private timer: ReturnType<typeof setTimeout> | undefined
  private rerunKey = randomId()

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => {
      this.generation++; clearTimeout(this.timer)
      this.projectId.set(params.get('projectId') ?? ''); this.runId.set(params.get('runId') ?? ''); this.historyScopeApplied = false
      this.run.set(null); this.error.set(''); this.actionError.set(''); this.rerunKey = randomId()
      this.evidence.reset(this.projectId(), this.runId())
      void this.load()
    })
    this.destroy.onDestroy(() => { this.generation++; clearTimeout(this.timer); this.evidence.invalidate() })
  }

  async load(): Promise<void> {
    clearTimeout(this.timer)
    const generation = ++this.generation
    this.loading.set(true); this.error.set('')
    try {
      const row = await this.api.request<RecordValue<Run>>(this.evidence.base())
      if (generation !== this.generation) return
      const application = this.route.snapshot.paramMap.get('applicationId') ?? ''
      if (application && !runApplications(row.data).includes(application)) throw new Error('This saved run has no snapshot for the active application. Choose All applications to inspect project-wide history.')
      this.run.set(row.data); this.observedAt.set(Date.now())
      const requested = this.historyScopeApplied ? '' : historyEvidenceTest(row.data, this.route.snapshot.queryParamMap.get('test') ?? '')
      this.historyScopeApplied = true
      if (requested) await this.evidence.inspectTest(requested)
      else await this.evidence.load()
    } catch (error) {
      if (generation === this.generation) {
        if (error instanceof ApiError && (error.status === 403 || error.status === 404)) { this.run.set(null); this.evidence.restrict() }
        this.error.set(error instanceof ApiError && error.status === 403 ? 'This run is no longer accessible with your current permissions. Its saved results remain stored. Check project access and select Refresh to retry.' : errorMessage(error, 'The run could not be loaded.'))
      }
    }
    finally {
      if (generation === this.generation) {
        this.loading.set(false)
        if (this.run() && !this.finished()) this.timer = setTimeout(() => { void this.load() }, 2000)
      }
    }
  }

  async refreshAccess(): Promise<void> {
    if (this.refreshingAccess() || this.mutating()) return
    this.refreshingAccess.set(true)
    try { await this.workspace.refreshAccess(); await this.load() }
    finally { this.refreshingAccess.set(false) }
  }

  async cancel(): Promise<void> { await this.mutate('cancel') }
  async rerunFailures(): Promise<void> { await this.mutate('rerun-failures') }

  private async mutate(action: string): Promise<void> {
    if (this.mutating()) return
    const base = this.evidence.base()
    this.mutating.set(true); this.actionError.set('')
    try {
      const row = await this.api.request<RecordValue<Run>>(`${base}/${action}`, 'POST', {}, undefined, action === 'rerun-failures' ? this.rerunKey : undefined)
      if (base !== this.evidence.base()) return
      if (action === 'rerun-failures') await this.router.navigate(this.workspace.path('runs', row.id))
      else await this.load()
    } catch (error) { if (base === this.evidence.base()) this.actionError.set(runActionFeedback(action, error instanceof ApiError ? error.status : 0, errorMessage(error, 'The run action failed.'))) }
    finally { this.mutating.set(false) }
  }
}
