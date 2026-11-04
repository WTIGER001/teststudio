import { randomId } from '../../core/random-id'
import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { BrowserRecordingRecord } from '../../core/generated/access.types'
import { PlaywrightEditorService } from './playwright-editor.service'

@Injectable()
export class BrowserRecorderService {
  readonly editor = inject(PlaywrightEditorService)
  readonly sources = this.editor.sources
  private readonly client = inject(AccessClientService)
  readonly enabled = signal(false)
  readonly busy = signal(false)
  readonly error = signal('')
  readonly selected = signal<BrowserRecordingRecord | null>(null)
  readonly items = signal<BrowserRecordingRecord[]>([])
  readonly cursor = signal('')
  readonly form = new FormGroup({
    url: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.maxLength(2048), Validators.pattern(/^https?:\/\//)] }),
    testId: new FormControl(`REC-${randomId().slice(0, 8)}`, { nonNullable: true, validators: [Validators.required, Validators.pattern(/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/)] }),
    title: new FormControl('Recorded browser flow', { nonNullable: true, validators: [Validators.required, Validators.maxLength(120)] })
  })
  private generation = 0
  private requestId = randomId()
  private reading = false

  constructor() {
    const destroy = inject(DestroyRef)
    this.sources.form.controls.suite.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.selected.set(null); this.reading = false; void this.load() })
    this.form.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.requestId = randomId() })
    const timer = setInterval(() => { const row = this.selected(); if (row && (row.data.state === 'recording' || row.data.state === 'starting') && !this.busy() && !this.reading) void this.read(row.id) }, 1500)
    destroy.onDestroy(() => { this.generation++; clearInterval(timer) })
    void this.load()
  }

  private scope() { return { project: this.sources.project(), suite: this.sources.form.controls.suite.value } }

  async load(more = false): Promise<void> {
    const generation = ++this.generation
    this.busy.set(true); this.error.set('')
    if (!more) { this.items.set([]); this.cursor.set('') }
    try {
      const [options, page] = await Promise.all([this.client.call('readBrowserRecordingOptions', this.scope()), this.client.call('listBrowserRecordings', { ...this.scope(), query: { limit: 30, cursor: more ? this.cursor() : undefined } })])
      if (generation !== this.generation) return
      this.enabled.set(options.enabled); this.items.set(more ? [...this.items(), ...page.items] : page.items); this.cursor.set(page.cursor ?? '')
      if (!this.selected()) { const active = page.items.find(row => row.data.state === 'starting' || row.data.state === 'recording'); if (active) { this.selected.set(active); void this.read(active.id) } }
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async start(): Promise<void> {
    if (this.form.invalid || this.busy() || !this.enabled() || this.active()) return
    const generation = ++this.generation
    this.busy.set(true); this.error.set('')
    try {
      const row = await this.client.call('startBrowserRecording', { ...this.scope(), body: { id: this.requestId, ...this.form.getRawValue() } })
      if (generation === this.generation) this.accept(row)
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  active(): boolean { return this.selected()?.data.state === 'recording' || this.selected()?.data.state === 'starting' }

  async read(id: string): Promise<void> {
    if (this.reading) return
    const generation = this.generation
    this.reading = true
    try {
      const row = await this.client.call('readBrowserRecording', { ...this.scope(), id })
      if (generation === this.generation) this.accept(row)
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.reading = false }
  }

  async stop(cancel = false): Promise<void> {
    const row = this.selected()
    if (!row || this.busy()) return
    const generation = ++this.generation
    this.reading = false; this.busy.set(true); this.error.set('')
    try {
      const result = await this.client.call('changeBrowserRecordingState', { ...this.scope(), id: row.id, action: cancel ? 'cancel' : 'stop' })
      if (generation === this.generation) { this.accept(result); this.requestId = randomId() }
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  private accept(row: BrowserRecordingRecord): void { this.selected.set(row); this.items.update(items => [row, ...items.filter(item => item.id !== row.id)]) }
  private fail(error: unknown): void {
    this.error.set(error instanceof Error ? error.message : 'The browser recorder request failed.')
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) { this.selected.set(null); this.items.set([]); this.enabled.set(false) }
  }
}
