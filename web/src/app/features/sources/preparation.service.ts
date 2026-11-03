import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { SourcePreparationRecord } from '../../core/generated/access.types'
import { SourcesService } from './sources.service'

@Injectable()
export class PreparationService {
  readonly sources = inject(SourcesService)
  private readonly client = inject(AccessClientService)
  readonly runners = signal<string[]>([])
 readonly bundleRunners = signal<string[]>([])
 readonly downloadURL = signal('')
  readonly jobs = signal<SourcePreparationRecord[]>([])
  readonly selected = signal<SourcePreparationRecord | null>(null)
  readonly cursor = signal('')
  readonly busy = signal(false)
  readonly error = signal('')
  readonly form = new FormGroup({ image: new FormControl('', { nonNullable: true, validators: Validators.required }), runner: new FormControl('go', { nonNullable: true, validators: Validators.required }) })
  private generation = 0
  private reading = false
  private detailGeneration = 0
  private key = crypto.randomUUID()

  constructor() {
    const destroy = inject(DestroyRef)
    this.sources.form.controls.suite.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.key = crypto.randomUUID(); void this.load() })
    this.form.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.key = crypto.randomUUID() })
    const timer = setInterval(() => { const job = this.selected(); if (job && (job.data.state === 'received' || job.data.state === 'preparing' || job.data.state === 'discovering' || job.data.cleanupPending) && !this.busy() && !this.reading) void this.inspect(job.id) }, 5000)
    destroy.onDestroy(() => { this.generation++; clearInterval(timer); this.clearDownload() })
    void this.load()
  }

  private scope() { return { project: this.sources.project(), suite: this.sources.form.controls.suite.value } }

  async load(more = false): Promise<void> {
    const generation = ++this.generation, scope = this.scope()
    this.error.set('')
    if (!more) { this.detailGeneration++; this.reading = false; this.busy.set(false); this.jobs.set([]); this.selected.set(null); this.cursor.set(''); this.runners.set([]); this.bundleRunners.set([]); this.clearDownload() }
    if (!scope.suite) return
    try {
      const [options, page] = await Promise.all([this.client.call('readPreparationOptions', scope), this.client.call('listSourcePreparations', { ...scope, query: { limit: 30, cursor: more ? this.cursor() : undefined } })])
      if (generation !== this.generation) return
      this.runners.set(options.runners); this.bundleRunners.set(options.bundleRunners); this.jobs.set(more ? [...this.jobs(), ...page.items] : page.items); this.cursor.set(page.cursor ?? '')
      if (!options.runners.includes(this.form.controls.runner.value) && options.runners[0]) this.form.controls.runner.setValue(options.runners[0])
    } catch (error) { if (generation === this.generation) this.fail(error) }
  }

  newRequest(): void { this.detailGeneration++; this.reading = false; this.key = crypto.randomUUID(); this.selected.set(null); this.error.set(''); this.clearDownload() }

  async prepare(): Promise<void> {
    if (this.form.invalid || this.busy()) return
    const generation = ++this.generation
    this.detailGeneration++; this.reading = false
    this.busy.set(true); this.error.set(''); this.clearDownload()
    try {
      const row = await this.client.call('enqueuePrebuiltPreparation', { ...this.scope(), key: this.key, body: this.form.getRawValue() })
      if (generation !== this.generation) return
      this.selected.set(row); this.jobs.update(items => [row, ...items.filter(item => item.id !== row.id)])
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async uploadBundle(file: File, manifest: string, key: string): Promise<void> {
    if (this.busy()) return
    const generation = ++this.generation
    this.detailGeneration++; this.reading = false
    this.busy.set(true); this.error.set(''); this.clearDownload()
    try {
      const row = await this.client.call('uploadSourceBundle', { ...this.scope(), key, query: { name: file.name, manifest }, body: file })
      if (generation !== this.generation) return
      this.selected.set(row); this.jobs.update(items => [row, ...items.filter(item => item.id !== row.id)])
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  showQueued(row: SourcePreparationRecord): void {
    this.generation++; this.detailGeneration++; this.reading = false; this.busy.set(false); this.error.set(''); this.clearDownload()
    this.selected.set(row); this.jobs.update(items => [row, ...items.filter(item => item.id !== row.id)])
  }

  async downloadBundle(): Promise<void> {
    const row = this.selected()
    if (!row || this.busy()) return
    const generation = this.generation, id = row.id
    this.busy.set(true); this.error.set(''); this.clearDownload()
    try {
      const blob = await this.client.call('downloadSourceBundle', { ...this.scope(), id })
      if (generation === this.generation && this.selected()?.id === id) this.downloadURL.set(URL.createObjectURL(blob))
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  private clearDownload(): void { const url = this.downloadURL(); if (url) URL.revokeObjectURL(url); this.downloadURL.set('') }

  async inspect(id: string): Promise<void> {
    const generation = this.generation, detail = ++this.detailGeneration
    this.reading = true
    if (this.selected()?.id !== id) this.clearDownload()
    try {
      const row = await this.client.call('readSourcePreparation', { ...this.scope(), id })
      if (generation !== this.generation || detail !== this.detailGeneration) return
      this.selected.set(row); this.jobs.update(items => items.map(item => item.id === row.id ? row : item))
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (detail === this.detailGeneration) this.reading = false }
  }

  async cancel(): Promise<void> {
    const row = this.selected()
    if (!row || this.busy()) return
    const generation = this.generation
    this.detailGeneration++; this.reading = false
    this.busy.set(true); this.error.set('')
    try {
      const cancelled = await this.client.call('cancelSourcePreparation', { ...this.scope(), id: row.id, revision: row.revision })
      if (generation !== this.generation) return
      this.selected.set(cancelled); this.jobs.update(items => items.map(item => item.id === row.id ? cancelled : item))
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async review(): Promise<void> { const job = this.selected(); if (job?.data.uploadId) await this.sources.openReview(job.data.uploadId, job.data.suiteId) }

  private fail(error: unknown): void {
    this.error.set(error instanceof Error ? error.message : 'Preparation request failed.')
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) { this.selected.set(null); this.jobs.set([]); this.runners.set([]); this.bundleRunners.set([]); this.clearDownload(); this.sources.preview.set(null); this.sources.content.set('') }
  }
}
