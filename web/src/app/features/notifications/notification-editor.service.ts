import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { ApiError, ApiService, type RecordValue } from '../../core/api.service'
import { errorMessage } from '../runs/run.utils'
import type { Channel, SecretReference } from './notification.models'
import { notificationDefinition, notificationForm, notificationValidation } from './notification.utils'
import { NotificationChannelsService } from './notification-channels.service'

@Injectable()
export class NotificationEditorService {
  private readonly api = inject(ApiService)
  private readonly destroy = inject(DestroyRef)
  private readonly route = inject(ActivatedRoute)
  readonly list = inject(NotificationChannelsService)
  readonly opened = signal(false)
  readonly editing = signal<RecordValue<Channel> | null>(null)
  readonly references = signal<SecretReference[]>([])
  readonly referencesReady = signal(false)
  readonly unavailableReference = signal('')
  readonly busy = signal(false)
  readonly error = signal('')
  readonly conflict = signal(false)
  readonly form = new FormGroup({
    id: new FormControl('', { nonNullable: true }), name: new FormControl('', { nonNullable: true }),
    enabled: new FormControl(false, { nonNullable: true }), kind: new FormControl('webhook', { nonNullable: true }), secretReference: new FormControl('', { nonNullable: true }),
    suites: new FormControl('', { nonNullable: true }), environments: new FormControl('', { nonNullable: true }), stages: new FormControl('', { nonNullable: true }),
    maxAttempts: new FormControl(6, { nonNullable: true }), includeEvidenceGaps: new FormControl(false, { nonNullable: true }),
    manual: new FormControl(true, { nonNullable: true }), schedule: new FormControl(true, { nonNullable: true }), rerun: new FormControl(true, { nonNullable: true }),
    passed: new FormControl(false, { nonNullable: true }), failed: new FormControl(true, { nonNullable: true }), timedOut: new FormControl(true, { nonNullable: true }),
    flaky: new FormControl(true, { nonNullable: true }), unknown: new FormControl(true, { nonNullable: true })
  })
  private generation = 0

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => {
      this.generation++; this.opened.set(false); this.editing.set(null); this.references.set([]); this.error.set(''); this.busy.set(false)
    })
    this.destroy.onDestroy(() => { this.generation++ })
  }
  async open(item?: RecordValue<Channel>): Promise<void> {
    const generation = ++this.generation
    this.editing.set(item ?? null); this.form.reset(notificationForm(item?.data)); this.opened.set(true); this.error.set(''); this.conflict.set(false)
    this.references.set([]); this.referencesReady.set(false); this.unavailableReference.set(item?.data.secretReference ?? ''); this.list.message.set(''); this.busy.set(true)
    this.list.focus('notification-editor-heading')
    try {
      const result = await this.api.request<{ items: SecretReference[] }>(`${this.list.base()}/references`)
      if (generation !== this.generation) return
      this.references.set(result.items ?? []); this.referencesReady.set(true)
      if (result.items?.some(ref => ref.id === item?.data.secretReference)) this.unavailableReference.set('')
    } catch (error) { if (generation === this.generation) this.error.set(errorMessage(error, 'Registered reference names could not be loaded.')) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  close(): void { this.generation++; this.opened.set(false); this.busy.set(false); this.list.focus('notification-channels-heading') }
  async reload(): Promise<void> {
    const selected = this.editing(); if (!selected || this.busy()) return
    const generation = ++this.generation
    this.busy.set(true)
    try {
      const stored = await this.api.request<RecordValue<Channel>>(`${this.list.base()}/channels/${encodeURIComponent(selected.id)}`)
      if (generation === this.generation) await this.open(stored)
    } catch (error) { if (generation === this.generation) this.error.set(errorMessage(error, 'Stored configuration could not be loaded.')) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  async submit(): Promise<void> {
    if (this.busy()) return
    this.form.markAllAsTouched(); this.conflict.set(false)
    const value = this.form.getRawValue(), validation = notificationValidation(value, this.references())
    this.error.set(validation); if (validation) return
    const generation = ++this.generation, item = this.editing()
    this.busy.set(true)
    try {
      await this.api.request(`${this.list.base()}/channels${item ? `/${encodeURIComponent(item.id)}` : ''}`, item ? 'PUT' : 'POST', notificationDefinition(value), item?.revision)
      if (generation !== this.generation) return
      this.list.message.set('Notification channel saved. Changes apply to future runs.'); this.close(); await this.list.load()
    } catch (error) {
      if (generation === this.generation) { this.error.set(errorMessage(error, 'The notification channel could not be saved.')); this.conflict.set(error instanceof ApiError && error.status === 409) }
    } finally { if (generation === this.generation) this.busy.set(false) }
  }
}
