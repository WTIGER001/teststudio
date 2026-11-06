import type { ProbeBinding } from '../probes/probe.models'
import { randomId } from '../../core/random-id'
import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { ApiError, ApiService, type RecordValue } from '../../core/api.service'
import { RunLookupsService } from '../runs/run-lookups.service'
import { errorMessage } from '../runs/run.utils'
import type { Schedule, SchedulePreview } from './schedule.models'
import { definitionFromForm, scheduleForm, scheduleWizardIssue } from './schedule.utils'
import { SchedulesService } from './schedules.service'
import { WizardController } from '../../shared/wizard/wizard-controller'
import { suggestedIdentifier } from '../../shared/wizard/creation.utils'

@Injectable()
export class ScheduleEditorService {
  private readonly api = inject(ApiService)
  private readonly destroy = inject(DestroyRef)
  private readonly route = inject(ActivatedRoute)
  readonly list = inject(SchedulesService)
  readonly lookups = inject(RunLookupsService)
  readonly opened = signal(false)
  readonly probe = signal<ProbeBinding | null>(null)
  readonly flow = new WizardController(3, step => { this.form.markAllAsTouched(); const issue = scheduleWizardIssue(step, this.form.getRawValue()); this.error.set(issue); return !issue })
  private suffix = randomId().slice(0, 8)
  readonly editing = signal<RecordValue<Schedule> | null>(null)
  readonly busy = signal(false)
  readonly error = signal('')
  readonly conflict = signal(false)
  readonly preview = signal<SchedulePreview | null>(null)
  readonly form = new FormGroup({
    id: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.pattern(/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/)] }),
    name: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.maxLength(256)] }),
    enabled: new FormControl(false, { nonNullable: true }), kind: new FormControl('interval', { nonNullable: true }),
    interval: new FormControl(60, { nonNullable: true }), cron: new FormControl('0 2 * * *', { nonNullable: true }), timezone: new FormControl('UTC', { nonNullable: true }),
    overlap: new FormControl('skip', { nonNullable: true }), suite: new FormControl('', { nonNullable: true, validators: [Validators.required] }),
    environment: new FormControl('', { nonNullable: true }), location: new FormControl('local', { nonNullable: true, validators: [Validators.required] }),
    release: new FormControl('', { nonNullable: true }), ids: new FormControl('', { nonNullable: true }),
    timeout: new FormControl(1800, { nonNullable: true, validators: [Validators.required, Validators.min(1), Validators.max(3600)] }),
    retries: new FormControl(0, { nonNullable: true, validators: [Validators.required, Validators.min(0), Validators.max(2)] })
  })
  private generation = 0

  constructor() {
    this.form.controls.name.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(name => { if (!this.editing() && this.form.controls.id.pristine) this.form.controls.id.setValue(suggestedIdentifier(name, this.suffix)) })
    this.form.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.generation++; this.preview.set(null) })
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.generation++; this.opened.set(false); this.editing.set(null); this.preview.set(null); this.error.set(''); this.busy.set(false) })
    this.route.queryParamMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(q => {
      if (q.get('probe')) void this.openProbe(q.get('probe')!, q.get('release') ?? '')
      else if (q.get('schedule')) void this.openSchedule(q.get('schedule')!)
    })
    this.destroy.onDestroy(() => { this.generation++ })
  }
  async open(item?: RecordValue<Schedule>): Promise<void> {
    this.suffix = randomId().slice(0, 8); this.flow.reset()
    this.probe.set(null); this.editing.set(item ?? null); this.form.reset(scheduleForm(item?.data)); this.opened.set(true); this.error.set(''); this.conflict.set(false); this.preview.set(null)
    const generation = ++this.generation
    this.list.message.set('')
    this.list.focus('schedule-editor-heading')
    try { await this.lookups.load(this.list.projectId()) }
    catch (error) { if (generation === this.generation) this.error.set(errorMessage(error, 'Schedule options could not be loaded.')) }
  }
  private async openSchedule(id: string): Promise<void> {
    try { const item = await this.api.request<RecordValue<Schedule>>(`${this.list.base()}/${encodeURIComponent(id)}`); await this.open(item) }
    catch (e) { this.error.set(errorMessage(e, 'The selected schedule is unavailable.')) }
  }
  private async openProbe(id: string, release: string): Promise<void> {
    await this.open()
    try {
      const record = await this.api.request<RecordValue<ProbeBinding>>(`/api/v1/projects/${this.list.projectId()}/probes/${encodeURIComponent(id)}`)
      this.probe.set(record.data)
      this.form.patchValue({ name: record.data.name + ' checks', suite: record.data.suiteId, environment: record.data.environmentId, ids: record.data.testId, location: record.data.executionLocation, interval: 300, timeout: 60, retries: 0, overlap: 'skip', enabled: false, release })
    } catch (e) { this.error.set(errorMessage(e, 'The selected probe is unavailable.')) }
  }
  close(): void { this.generation++; this.opened.set(false); this.list.focus('schedules-heading') }
  async reload(): Promise<void> {
    const selected = this.editing(); if (!selected) return
    this.busy.set(true)
    try { await this.open(await this.api.request<RecordValue<Schedule>>(`${this.list.base()}/${encodeURIComponent(selected.id)}`)) }
    catch (error) { this.error.set(errorMessage(error, 'Stored configuration could not be loaded.')) }
    finally { this.busy.set(false) }
  }
  suiteName(): string { const id = this.form.controls.suite.value; return this.lookups.suites().find(row => row.id === id)?.data.name || id }
  targetName(): string { const id = this.form.controls.environment.value, target = this.lookups.environments().find(row => row.id === id); return target ? `${target.data.name} · ${target.data.stage}${target.data.protected ? ' · Protected' : ''}` : id || 'No target' }
  async submit(preview = false): Promise<void> {
    if (this.busy() || !this.flow.ready()) return
    this.form.markAllAsTouched(); this.error.set(''); this.conflict.set(false)
    this.list.message.set('')
    if (this.form.invalid) { this.error.set('Complete the required fields and valid run limits.'); return }
    const generation = ++this.generation
    const item = this.editing(), definition = definitionFromForm(this.form.getRawValue(), item?.data)
    this.busy.set(true)
    try {
      if (preview) {
        const result = await this.api.request<SchedulePreview>(`${this.list.base()}/preview`, 'POST', definition)
        if (generation === this.generation) this.preview.set(result)
      } else {
        await this.api.request(`${this.list.base()}${item ? `/${encodeURIComponent(item.id)}` : ''}`, item ? 'PUT' : 'POST', definition, item?.revision)
        if (generation !== this.generation) return
        this.list.message.set('Schedule configuration saved.'); this.close(); await this.list.load()
      }
    } catch (error) {
      if (generation === this.generation) { this.error.set(errorMessage(error, 'The schedule could not be saved.')); this.conflict.set(error instanceof ApiError && error.status === 409) }
    } finally { this.busy.set(false) }
  }
}
