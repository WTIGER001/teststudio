import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { ApiError, ApiService, type RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { RunEvidenceService } from './run-evidence.service'
import type { Artifact } from './run.models'
import type { RetentionView } from './run-retention.models'
import { activePin, effectiveArtifactState, localDateTime, pendingArtifacts, pinInput } from './run-retention.utils'
import { errorMessage } from './run.utils'

@Injectable()
export class RunRetentionService {
  private readonly api = inject(ApiService)
  readonly evidence = inject(RunEvidenceService)
  readonly workspace = inject(WorkspaceService)
  readonly view = signal<RetentionView | null>(null)
  readonly now = signal(Date.now())
  readonly pinned = computed(() => activePin(this.view()?.pin, this.now()))
  readonly busy = signal(false)
  readonly error = signal('')
  readonly success = signal('')
  readonly deleting = signal<RecordValue<Artifact> | null>(null)
  readonly pinForm = new FormGroup({ reason: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.maxLength(500)] }), until: new FormControl('', { nonNullable: true }) })
  readonly deleteForm = new FormGroup({ reason: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.maxLength(500)] }) })
  private base = ''
  private generation = 0
  private pinRevision = 0

  constructor() {
    const destroy = inject(DestroyRef)
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(destroy)).subscribe(params => {
      this.generation++; this.view.set(null); this.error.set(''); this.success.set(''); this.deleting.set(null); this.pinForm.reset(); this.busy.set(false)
      this.base = `/api/v1/projects/${encodeURIComponent(params.get('projectId') ?? '')}/runs/${encodeURIComponent(params.get('runId') ?? '')}`
      void this.load()
    })
    const timer = setInterval(() => {
      this.now.set(Date.now())
      if (!this.busy()) void this.refresh()
    }, 5000)
    destroy.onDestroy(() => { this.generation++; clearInterval(timer) })
  }

  artifactState(artifact: Artifact): string { return effectiveArtifactState(artifact, this.pinned(), this.now()) }

  private async refresh(): Promise<void> {
    const pending = this.view()?.status.state === 'pending' || pendingArtifacts(this.evidence.artifacts())
    await this.load()
    if (pending && this.evidence.base() === this.base && !this.evidence.loading()) await this.evidence.load()
  }

  async load(): Promise<void> {
    const generation = ++this.generation
    try {
      const view = await this.api.request<RetentionView>(`${this.base}/retention`)
      if (generation !== this.generation) return
      this.view.set(view)
      if (this.pinForm.pristine) {
        this.pinRevision = view.revision
        this.pinForm.reset({ reason: view.pin.reason, until: localDateTime(view.pin.until) })
      }
    } catch (error) { if (generation === this.generation) this.error.set(errorMessage(error, 'Retention settings could not be loaded.')) }
  }

  async reloadPin(): Promise<void> { this.pinForm.markAsPristine(); this.error.set(''); await this.load() }

  async pin(pinned: boolean): Promise<void> {
    if (this.busy() || this.pinForm.invalid) return
    const generation = ++this.generation
    this.busy.set(true); this.error.set(''); this.success.set('')
    try {
      const values = this.pinForm.getRawValue()
      const input = pinInput(pinned, values.reason, values.until, this.now())
      const view = await this.api.request<RetentionView>(`${this.base}/retention/pin`, 'PUT', input, this.pinRevision)
      if (generation !== this.generation) return
      this.view.set(view); this.pinRevision = view.revision; this.pinForm.reset({ reason: view.pin.reason, until: localDateTime(view.pin.until) })
      this.success.set(pinned ? 'Evidence is pinned for investigation.' : 'Evidence pin removed. Normal expiry applies.')
    } catch (error) { if (generation === this.generation) this.error.set(errorMessage(error, 'The evidence pin could not be changed.') + (error instanceof ApiError && error.status === 409 ? ' Reload pin settings and review the current value.' : '')) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  beginDelete(artifact: RecordValue<Artifact>): void { this.deleting.set(artifact); this.deleteForm.reset(); this.error.set(''); this.success.set('') }
  closeDelete(): void { this.deleting.set(null); this.deleteForm.reset() }

  async deleteArtifact(): Promise<boolean> {
    const item = this.deleting()
    if (!item || this.deleteForm.invalid || this.busy()) return false
    const generation = ++this.generation
    this.busy.set(true); this.error.set(''); this.success.set('')
    try {
      await this.api.request(`${this.base}/artifacts/${encodeURIComponent(item.id)}`, 'DELETE', this.deleteForm.getRawValue(), item.revision)
      if (generation !== this.generation) return false
      this.closeDelete(); this.success.set('Deletion requested. The artifact state shows cleanup progress. Its manifest and test results remain available.')
      await this.evidence.load()
      return true
    } catch (error) { if (generation === this.generation) this.error.set(errorMessage(error, 'Artifact deletion failed.') + (error instanceof ApiError && error.status === 409 ? ' Refresh the run and reopen the deletion form.' : '')); return false }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
}
