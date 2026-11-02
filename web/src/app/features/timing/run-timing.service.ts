import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { ApiError, ApiService, type RecordValue } from '../../core/api.service'
import { RunDetailService } from '../runs/run-detail.service'
import { errorMessage } from '../runs/run.utils'
import type { TimingPrediction, TimingWindow } from './timing.models'

@Injectable()
export class RunTimingService {
  private readonly api = inject(ApiService)
  private readonly destroy = inject(DestroyRef)
  readonly detail = inject(RunDetailService)
  readonly baseline = signal<RecordValue<TimingWindow> | null>(null)
  readonly selected = signal<TimingPrediction | null>(null)
  readonly loading = signal(false)
  readonly saving = signal(false)
  readonly error = signal('')
  readonly success = signal('')
  readonly reason = new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.maxLength(500), Validators.pattern(/\S/)] })
  readonly form = new FormGroup({ reason: this.reason })
  private generation = 0

  constructor() {
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => this.close())
    this.destroy.onDestroy(() => { this.generation++ })
  }

  close(): void { this.generation++; this.baseline.set(null); this.selected.set(null); this.error.set(''); this.success.set(''); this.reason.reset(''); this.loading.set(false) }

  async open(prediction: TimingPrediction): Promise<void> {
    const generation = ++this.generation
    this.selected.set(prediction); this.baseline.set(null); this.error.set(''); this.success.set(''); this.loading.set(true); this.reason.reset('')
    try {
      const row = await this.api.request<RecordValue<TimingWindow>>(this.base(prediction))
      if (generation === this.generation) this.baseline.set(row)
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof ApiError && error.status === 404 ? 'This timing context has no clean observations yet.' : errorMessage(error, 'The baseline is unavailable.')) }
    finally { if (generation === this.generation) this.loading.set(false) }
  }

  async reset(): Promise<void> {
    const prediction = this.selected(), row = this.baseline(), generation = this.generation
    if (!prediction || !row || this.reason.invalid || this.saving()) return
    this.saving.set(true); this.error.set(''); this.success.set('')
    try {
      const updated = await this.api.request<RecordValue<TimingWindow>>(`${this.base(prediction)}/reset`, 'POST', { reason: this.reason.value.trim() }, row.revision)
      if (generation !== this.generation) return
      this.baseline.set(updated); this.reason.reset(''); this.success.set('Baseline reset. Future runs learn new observations. This run keeps its captured timing criteria.')
    } catch (error) { if (generation === this.generation) this.error.set(errorMessage(error, 'The baseline could not be reset. Reload it to review the current revision.')) }
    finally { this.saving.set(false) }
  }

  private base(prediction: TimingPrediction): string { return `/api/v1/projects/${encodeURIComponent(this.detail.projectId())}/timing-baselines/${encodeURIComponent(prediction.baselineId)}` }
}
