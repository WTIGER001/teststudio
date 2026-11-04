import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormArray, FormControl, FormGroup, Validators } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { SourcePreparationRecord, SourceRepositoryOptions, SourceRepositoryRecord, SourceRepositorySyncRecord } from '../../core/generated/access.types'
import { SourcesService } from './sources.service'
import { WizardController } from '../../shared/wizard/wizard-controller'
import { suggestedIdentifier } from '../../shared/wizard/creation.utils'
import { repositoryConfig, repositoryPreparations, repositoryReference, repositorySyncActive, repositoryWebhookAddress, repositoryWizardProblem } from './repository.utils'

@Injectable()
export class RepositoriesService {
  readonly sources = inject(SourcesService)
  private readonly client = inject(AccessClientService)
  readonly options = signal<SourceRepositoryOptions>({ providers: [], credentials: [], webhookCredentials: [] })
  readonly rows = signal<SourceRepositoryRecord[]>([])
  readonly selected = signal<SourceRepositoryRecord | null>(null)
  readonly history = signal<SourceRepositorySyncRecord[]>([])
  readonly sync = signal<SourceRepositorySyncRecord | null>(null)
  readonly prepared = signal<SourcePreparationRecord | null>(null)
  readonly webhookAddress = computed(() => repositoryWebhookAddress(location.origin, this.sources.project(), this.selected()?.id ?? ''))
  readonly preparations = computed(() => repositoryPreparations(this.sync()?.data.preparations ?? {}))
  readonly cursor = signal('')
  readonly syncCursor = signal('')
  readonly editor = signal(false)
  readonly busy = signal(false)
  readonly loading = signal(false)
  readonly historyLoading = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly flow = new WizardController(3, step => this.validate(step))
  readonly force = new FormControl(false, { nonNullable: true })
  readonly form = new FormGroup({
    id: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.pattern('^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$')] }),
    name: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.maxLength(200)] }),
    repository: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.maxLength(140), Validators.pattern('^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$')] }),
    referenceKind: new FormControl('branch', { nonNullable: true }),
    reference: new FormControl('main', { nonNullable: true, validators: [Validators.required, Validators.maxLength(256)] }),
    credential: new FormControl('', { nonNullable: true }),
    enabled: new FormControl(true, { nonNullable: true }),
    autoActivate: new FormControl(false, { nonNullable: true }),
    push: new FormControl(false, { nonNullable: true }),
    webhookCredential: new FormControl('', { nonNullable: true }),
    intervalSeconds: new FormControl(0, { nonNullable: true, validators: [Validators.min(0), Validators.max(86400)] }),
    bindings: new FormArray<FormGroup<{ suite: FormControl<string>; manifest: FormControl<string>; runner: FormControl<string> }>>([])
  })
  private generation = 0
  private listGeneration = 0
  private historyGeneration = 0
  private detailGeneration = 0
  private polling = false
  private syncKey = crypto.randomUUID()
  private identifierSuffix = crypto.randomUUID().slice(0, 8)

  constructor() {
    const destroy = inject(DestroyRef)
    this.form.controls.name.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(name => {
      if (!this.selected() && this.form.controls.id.pristine) this.form.controls.id.setValue(suggestedIdentifier(name, this.identifierSuffix))
    })
    this.force.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.syncKey = crypto.randomUUID() })
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(destroy)).subscribe(() => {
      this.generation++; this.detailGeneration++; this.rows.set([]); this.selected.set(null); this.history.set([]); this.sync.set(null); this.prepared.set(null); this.editor.set(false); this.cursor.set(''); this.syncCursor.set(''); this.options.set({ providers: [], credentials: [], webhookCredentials: [] }); this.busy.set(false); void this.load()
    })
    const timer = setInterval(() => { if (repositorySyncActive(this.sync()?.data.state) && !this.busy() && !this.polling) void this.refreshSync() }, 5000)
    destroy.onDestroy(() => { this.generation++; this.detailGeneration++; clearInterval(timer) })
  }

  private scope() { return { project: this.sources.project() } }

  async load(more = false): Promise<void> {
    const generation = this.generation, request = ++this.listGeneration
    this.loading.set(true)
    this.error.set('')
    try {
      const [options, page] = await Promise.all([this.client.call('readRepositorySourceOptions', this.scope()), this.client.call('listRepositorySources', { ...this.scope(), query: { limit: 30, cursor: more ? this.cursor() : undefined } })])
      if (generation !== this.generation || request !== this.listGeneration) return
      this.options.set(options); this.rows.set(more ? [...this.rows(), ...page.items] : page.items); this.cursor.set(page.cursor ?? '')
    } catch (error) { if (generation === this.generation && request === this.listGeneration) this.fail(error) }
    finally { if (request === this.listGeneration) this.loading.set(false) }
  }

  addBinding(suite = '', manifest = 'teststudio-suite.json', runner = 'go'): void {
    this.form.controls.bindings.push(new FormGroup({ runner: new FormControl(runner, { nonNullable: true, validators: Validators.required }), suite: new FormControl(suite, { nonNullable: true, validators: Validators.required }), manifest: new FormControl(manifest, { nonNullable: true, validators: [Validators.required, Validators.maxLength(512)] }) }))
  }

  removeBinding(index: number): void { if (this.form.controls.bindings.length > 1) this.form.controls.bindings.removeAt(index) }

  edit(row?: SourceRepositoryRecord): void {
    this.flow.reset(); this.identifierSuffix = crypto.randomUUID().slice(0, 8)
    this.detailGeneration++; this.sync.set(null); this.prepared.set(null); this.history.set([]); this.force.setValue(false)
    this.selected.set(row ?? null); this.editor.set(true); this.error.set(''); this.message.set('')
    const config = row?.data.configuration, reference = repositoryReference(config?.reference ?? 'heads/main')
    this.form.controls.id.enable()
    this.form.reset({ id: row?.id ?? '', name: config?.name ?? '', repository: config?.repository ?? '', referenceKind: reference.kind, reference: reference.value, credential: config?.credentialReference ?? '', enabled: config?.enabled ?? true, autoActivate: config?.autoActivate ?? false, push: config?.pushEnabled ?? false, webhookCredential: config?.webhookCredentialReference ?? '', intervalSeconds: config?.syncIntervalSeconds ?? 0 })
    this.form.controls.bindings.clear()
    if (config) for (const binding of config.bindings) this.addBinding(binding.suiteId, binding.manifestPath, binding.runner)
    else this.addBinding(this.sources.form.controls.suite.value)
    if (row) this.form.controls.id.disable()
  }

  async save(): Promise<void> {
    if (!this.flow.ready() || !this.validate(2)) return
    if (this.form.invalid || this.form.controls.bindings.length === 0 || this.busy()) return
    const generation = this.generation, value = this.form.getRawValue()
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      const row = await this.client.call('saveRepositorySource', { ...this.scope(), source: value.id, revision: this.selected()?.revision ?? 0, body: repositoryConfig(value) })
      if (generation !== this.generation) return
      this.selected.set(row); this.editor.set(false); this.message.set('Repository source saved. No remote sync, build, or test was started.'); await this.load()
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  advance(): void { if (this.flow.step() < 2) this.flow.next(); else void this.save() }

  private validate(step: number): boolean {
    this.error.set('')
    const controls = this.form.controls
    if (step === 1) for (const binding of controls.bindings.controls) for (const field of Object.values(binding.controls)) field.updateValueAndValidity({ emitEvent: false })
    const fields = step === 0 ? [controls.id, controls.name, controls.repository, controls.reference, controls.credential] : step === 1 ? [controls.bindings] : [controls.intervalSeconds, controls.webhookCredential]
    for (const field of fields) { field.updateValueAndValidity(); field.markAllAsTouched() }
    if (fields.some(field => field.invalid)) { this.error.set('Complete the highlighted fields before continuing.'); return false }
    const problem = repositoryWizardProblem(this.form.getRawValue(), step)
    if (problem) {
      const field = problem.field === 'suite' ? controls.bindings.at(problem.binding!).controls.suite : controls[problem.field]
      field.setErrors({ configuration: true }); this.error.set(problem.message); return false
    }
    return true
  }

  async inspect(row: SourceRepositoryRecord): Promise<void> {
    const generation = this.generation, detail = ++this.detailGeneration
    this.busy.set(true); this.error.set(''); this.sync.set(null); this.prepared.set(null); this.history.set([]); this.syncCursor.set(''); this.editor.set(false); this.syncKey = crypto.randomUUID(); this.force.setValue(false)
    try {
      const current = await this.client.call('readRepositorySource', { ...this.scope(), source: row.id })
      if (generation !== this.generation || detail !== this.detailGeneration) return
      this.selected.set(current); await this.loadHistory()
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation && detail === this.detailGeneration) this.busy.set(false) }
  }

  async loadHistory(more = false): Promise<void> {
    const source = this.selected()?.id, generation = this.generation, detail = this.detailGeneration, request = ++this.historyGeneration
    if (!source) return
    this.historyLoading.set(true)
    try {
      const page = await this.client.call('listRepositorySyncs', { ...this.scope(), source, query: { limit: 30, cursor: more ? this.syncCursor() : undefined } })
      if (generation !== this.generation || detail !== this.detailGeneration || request !== this.historyGeneration) return
      this.history.set(more ? [...this.history(), ...page.items] : page.items); this.syncCursor.set(page.cursor ?? '')
    } catch (error) { if (generation === this.generation && request === this.historyGeneration) this.fail(error) }
    finally { if (request === this.historyGeneration) this.historyLoading.set(false) }
  }

  async syncNow(): Promise<void> {
    const row = this.selected(), generation = this.generation, detail = this.detailGeneration
    if (!row || this.busy()) return
    this.busy.set(true); this.error.set(''); this.prepared.set(null)
    try {
      const sync = await this.client.call('enqueueRepositorySync', { ...this.scope(), source: row.id, key: this.syncKey, body: { force: this.force.value } })
      if (generation !== this.generation || detail !== this.detailGeneration) return
      this.sync.set(sync); this.history.update(items => [sync, ...items.filter(item => item.id !== sync.id)]); this.syncKey = crypto.randomUUID()
      this.message.set('Repository sync queued. Tests do not run during synchronization.')
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation && detail === this.detailGeneration) this.busy.set(false) }
  }

  async inspectSync(row: SourceRepositorySyncRecord): Promise<void> { this.sync.set(row); this.prepared.set(null); await this.refreshSync() }

  async refreshSync(): Promise<void> {
    const row = this.sync(), generation = this.generation, detail = this.detailGeneration
    if (!row || this.polling) return
    this.polling = true
    try {
      const sync = await this.client.call('readRepositorySync', { ...this.scope(), source: row.data.sourceId, sync: row.id })
      if (generation !== this.generation || detail !== this.detailGeneration || this.sync()?.id !== row.id) return
      this.sync.set(sync); this.history.update(items => items.map(item => item.id === sync.id ? sync : item))
      if (!repositorySyncActive(sync.data.state)) await this.load()
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { this.polling = false }
  }

  async inspectPreparation(suite: string, id: string): Promise<void> {
    const generation = this.generation, detail = ++this.detailGeneration
    this.busy.set(true); this.error.set('')
    try { const job = await this.client.call('readSourcePreparation', { ...this.scope(), suite, id }); if (generation === this.generation && detail === this.detailGeneration) this.prepared.set(job) }
    catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation && detail === this.detailGeneration) this.busy.set(false) }
  }

  async reviewPreparation(): Promise<void> { const job = this.prepared(); if (job?.data.uploadId) { this.sources.form.controls.suite.setValue(job.data.suiteId); await this.sources.openReview(job.data.uploadId, job.data.suiteId) } }

  private fail(error: unknown): void {
    this.error.set(error instanceof ApiError && error.status === 409 ? 'The source changed. Refresh its details before saving or syncing again.' : error instanceof Error ? error.message : 'Repository source request failed.')
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) { this.rows.set([]); this.selected.set(null); this.history.set([]); this.sync.set(null); this.prepared.set(null); this.editor.set(false); this.options.set({ providers: [], credentials: [], webhookCredentials: [] }) }
  }
}
