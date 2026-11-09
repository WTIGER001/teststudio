import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError, ApiService, type Page, type RecordValue } from '../../core/api.service'
import type { SourceUploadPreview, SourceUploadRecord } from '../../core/generated/access.types'
import { SessionService } from '../../core/session.service'
import { applicationSuites } from '../../core/application-scope.utils'
import { WorkspaceService } from '../../core/workspace.service'
import type { Suite } from '../configuration/configuration.service'
import { sourceFormat } from './source.utils'

@Injectable()
export class SourcesService {
  readonly creationDirty = signal(false)
  readonly recordingMode = signal('disabled')
  readonly recordingInputPending = signal(false)
  readonly sourceEditingOpen = signal(false)
  private readonly api = inject(ApiService)
  private readonly client = inject(AccessClientService)
  readonly workspace = inject(WorkspaceService)
  readonly session = inject(SessionService)
  readonly project = signal('')
  readonly applicationId = signal('')
  readonly suites = signal<RecordValue<Suite>[]>([])
  readonly items = signal<SourceUploadRecord[]>([])
  readonly cursor = signal('')
  readonly loading = signal(false)
  readonly saving = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly editor = signal(false)
  readonly draftDirty = signal(false)
  readonly preview = signal<SourceUploadPreview | null>(null)
  readonly content = signal('')
  readonly activate = signal(false)
  readonly form = new FormGroup({
    suite: new FormControl('', { nonNullable: true, validators: Validators.required }),
    name: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.maxLength(200)] }),
    format: new FormControl<'json' | 'yaml'>('json', { nonNullable: true, validators: Validators.required }),
    content: new FormControl('', { nonNullable: true, validators: Validators.required })
  })
  private generation = 0
  private listGeneration = 0

  constructor() {
    const destroy = inject(DestroyRef)
    this.form.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.generation++; this.saving.set(false); this.preview.set(null); this.content.set(''); this.message.set(''); this.error.set('') })
    this.form.controls.suite.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.items.set([]); this.cursor.set(''); void this.list() })
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(destroy)).subscribe(params => { this.applicationId.set(params.get('applicationId') ?? ''); this.project.set(params.get('projectId') ?? ''); void this.load() })
    destroy.onDestroy(() => { this.generation++; this.listGeneration++ })
  }

  async load(): Promise<void> {
    const generation = ++this.generation
    this.listGeneration++; this.loading.set(true); this.error.set(''); this.message.set(''); this.preview.set(null); this.content.set(''); this.items.set([]); this.suites.set([]); this.editor.set(false)
    this.form.reset({ suite: '', name: '', format: 'json', content: '' }, { emitEvent: false })
    try {
      const response = await this.api.request<Page<RecordValue<Suite>>>(`/api/v1/projects/${encodeURIComponent(this.project())}/suites`)
      if (generation !== this.generation) return
      this.suites.set(applicationSuites(response.items.filter(row => !row.data.archived), this.applicationId()))
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Suites could not be loaded.') }
    finally { if (generation === this.generation) this.loading.set(false) }
  }

  async list(more = false): Promise<void> {
    const generation = ++this.listGeneration, suite = this.form.controls.suite.value
    if (!suite) { this.loading.set(false); return }
    this.loading.set(true); this.error.set('')
    try {
      const response = await this.client.call('listSourceUploads', { project: this.project(), suite, query: { limit: 30, cursor: more ? this.cursor() : undefined } })
      if (generation !== this.listGeneration) return
      this.items.set(more ? [...this.items(), ...response.items] : response.items); this.cursor.set(response.cursor ?? '')
    } catch (error) { if (generation === this.listGeneration) { this.items.set([]); this.cursor.set(''); this.error.set(error instanceof Error ? error.message : 'Uploads could not be loaded.') } }
    finally { if (generation === this.listGeneration) this.loading.set(false) }
  }

  openEditor(): void { this.generation++; this.preview.set(null); this.content.set(''); this.error.set(''); this.message.set(''); this.editor.set(true) }

  async file(file: File | undefined): Promise<void> {
    if (!file) return
    if (file.size > 1 << 20) { this.error.set('The source file exceeds the 1 MiB limit.'); return }
    const generation = ++this.generation
    try {
      const content = new TextDecoder('utf-8', { fatal: true }).decode(await file.arrayBuffer())
      if (generation !== this.generation) return
      this.form.patchValue({ name: file.name, format: sourceFormat(file.name), content })
    } catch { if (generation === this.generation) this.error.set('The source file must be valid UTF-8 and readable.') }
  }

  async stage(): Promise<void> {
    if (this.form.invalid || this.saving()) return
    const generation = ++this.generation
    this.saving.set(true); this.error.set(''); this.message.set(''); this.preview.set(null); this.content.set(''); this.activate.set(false)
    const { suite, name, format, content } = this.form.getRawValue()
    try {
      const response = await this.client.call('stageSourceUpload', { project: this.project(), suite, body: { name, format, content } })
      if (generation !== this.generation) return
      this.preview.set(response); await this.list()
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Source validation failed.') }
    finally { if (generation === this.generation) this.saving.set(false) }
  }

  async inspect(item: SourceUploadRecord): Promise<void> { return this.openReview(item.id,item.data.suiteId) }

  async openReview(id: string, suite: string): Promise<void> {
    const generation = ++this.generation
    this.saving.set(true); this.preview.set(null); this.content.set(''); this.editor.set(false); this.error.set(''); this.message.set(''); this.activate.set(false)
    try {
      const response = await this.client.call('readSourceUpload', { project: this.project(), suite, id })
      if (generation === this.generation) this.preview.set(response)
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Upload could not be loaded.') }
    finally { if (generation === this.generation) this.saving.set(false) }
  }

  async showContent(): Promise<void> {
    const preview = this.preview()
    if (!preview || this.saving()) return
    const generation = ++this.generation
    this.saving.set(true); this.error.set('')
    try {
      const response = await this.client.call('readSourceContent', { project: this.project(), suite: preview.upload.data.suiteId, id: preview.upload.id })
      if (generation === this.generation) this.content.set(response.content)
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Source content could not be loaded.') }
    finally { if (generation === this.generation) this.saving.set(false) }
  }

  async publish(): Promise<void> {
    const preview = this.preview()
    if (!preview || this.saving() || preview.upload.data.state !== 'staged') return
    const generation = ++this.generation
    this.saving.set(true); this.error.set('')
    const args = { project: this.project(), suite: preview.upload.data.suiteId, id: preview.upload.id }
    try {
      await this.client.call('publishSourceUpload', { ...args, revision: preview.upload.revision, body: { activate: true } })
      if (generation !== this.generation) return
      const stored = await this.client.call('readSourceUpload', args)
      if (generation !== this.generation) return
      this.preview.set(stored); this.editor.set(false); this.message.set('Source version saved. No tests were started.'); await this.list()
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof ApiError && error.status === 409 ? 'The catalog, suite, or access changed. Review current data and validate the source again.' : error instanceof Error ? error.message : 'Save failed. Review the stored upload before retrying.') }
    finally { if (generation === this.generation) this.saving.set(false) }
  }
}
