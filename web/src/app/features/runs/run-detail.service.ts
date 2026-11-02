import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute, Router } from '@angular/router'
import { ApiService, type RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { RunEvidenceService } from './run-evidence.service'
import type { Run } from './run.models'
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
  readonly canCancel = computed(() => { const run = this.run(); return !!run && !terminal(run.state) && !run.cancelRequested && this.workspace.can('runs.cancel') })
  readonly canRerun = computed(() => { const run = this.run(); return !!run && canRerun(run) && this.workspace.can('runs.create') })
  readonly finished = computed(() => { const run = this.run(); return !!run && terminal(run.state) })
  private generation = 0
  private timer: ReturnType<typeof setTimeout> | undefined
  private rerunKey = crypto.randomUUID()

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => {
      this.generation++; clearTimeout(this.timer)
      this.projectId.set(params.get('projectId') ?? ''); this.runId.set(params.get('runId') ?? '')
      this.run.set(null); this.error.set(''); this.rerunKey = crypto.randomUUID()
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
      this.run.set(row.data)
      await this.evidence.load()
    } catch (error) { if (generation === this.generation) this.error.set(errorMessage(error, 'The run could not be loaded.')) }
    finally {
      if (generation === this.generation) {
        this.loading.set(false)
        if (this.run() && !this.finished()) this.timer = setTimeout(() => { void this.load() }, 2000)
      }
    }
  }

  async cancel(): Promise<void> { await this.mutate('cancel') }
  async rerunFailures(): Promise<void> { await this.mutate('rerun-failures') }

  private async mutate(action: string): Promise<void> {
    if (this.mutating()) return
    const base = this.evidence.base()
    this.mutating.set(true); this.error.set('')
    try {
      const row = await this.api.request<RecordValue<Run>>(`${base}/${action}`, 'POST', {}, undefined, action === 'rerun-failures' ? this.rerunKey : undefined)
      if (base !== this.evidence.base()) return
      if (action === 'rerun-failures') await this.router.navigate(['/projects', this.projectId(), 'runs', row.id])
      else await this.load()
    } catch (error) { if (base === this.evidence.base()) this.error.set(errorMessage(error, 'The run action failed.')) }
    finally { this.mutating.set(false) }
  }
}
