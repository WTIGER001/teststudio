import { randomId } from '../../core/random-id'
import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { PlaywrightDraft, PlaywrightDraftRecord } from '../../core/generated/access.types'
import { PreparationService } from './preparation.service'
import { editorPathError, protectedEditorFile, replaceEditorFile, sameEditorDocument } from './playwright-editor.utils'
import { configureAPIStarter, type APIStarterOptions } from './api-draft.utils'

@Injectable()
export class PlaywrightEditorService {
  readonly preparation = inject(PreparationService)
  readonly sources = this.preparation.sources
  private readonly client = inject(AccessClientService)
  readonly drafts = signal<PlaywrightDraftRecord[]>([])
  readonly cursor = signal('')
  readonly saved = signal<PlaywrightDraftRecord | null>(null)
  readonly document = signal<PlaywrightDraft | null>(null)
  readonly selectedPath = signal('')
  readonly busy = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  private generation = 0
  private key = randomId()

  constructor() {
    const destroy = inject(DestroyRef)
    this.sources.form.controls.suite.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.saved.set(null); this.document.set(null); this.sources.draftDirty.set(false); void this.load() })
    const warn = (event: BeforeUnloadEvent) => { if (this.sources.draftDirty() || this.sources.creationDirty()) event.preventDefault() }
    window.addEventListener('beforeunload', warn)
    destroy.onDestroy(() => { this.generation++; window.removeEventListener('beforeunload', warn); this.sources.draftDirty.set(false) })
    void this.load()
  }

  private scope() { return { project: this.sources.project(), suite: this.sources.form.controls.suite.value } }

  async load(more = false): Promise<void> {
    const generation = ++this.generation
    this.busy.set(true); this.error.set(''); this.message.set('')
    if (!more) { this.drafts.set([]); this.cursor.set('') }
    try {
      const page = await this.client.call('listPlaywrightDrafts', { ...this.scope(), query: { limit: 30, cursor: more ? this.cursor() : undefined } })
      if (generation !== this.generation) return
      this.drafts.set(more ? [...this.drafts(), ...page.items] : page.items); this.cursor.set(page.cursor ?? '')
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async create(type: 'browser' | 'api' = 'browser', options?: APIStarterOptions): Promise<boolean> {
    if (this.busy() || this.sources.draftDirty()) return false
    const generation = ++this.generation
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      const starter = await this.client.call('readPlaywrightStarter', { ...this.scope(), query: { type } })
      if (generation !== this.generation) return false
      const document = type === 'api' && options ? configureAPIStarter(starter, options) : starter
      this.saved.set(null); this.document.set({ ...document, id: randomId() }); this.selectedPath.set(document.files[0]?.path ?? ''); this.sources.draftDirty.set(true)
      return true
    } catch (error) { if (generation === this.generation) this.fail(error); return false }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async open(id: string): Promise<void> {
    if (this.busy() || this.sources.draftDirty()) return
    const generation = ++this.generation
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      const row = await this.client.call('readPlaywrightDraft', { ...this.scope(), id })
      if (generation === this.generation) this.accept(row)
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async importPreparation(id: string): Promise<void> {
    if (this.busy() || this.sources.draftDirty()) return
    const generation = ++this.generation
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      const document = await this.client.call('readPreparationEditorSource', { ...this.scope(), id })
      if (generation !== this.generation) return
      this.saved.set(null); this.document.set(document); this.selectedPath.set(document.files[0]?.path ?? ''); this.sources.draftDirty.set(true)
      this.message.set('Retained source opened as an unsaved copy. Save this test before preparing it. The original bundle and versions stay unchanged.')
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async importRecording(id: string): Promise<void> {
    if (this.busy() || this.sources.draftDirty()) return
    const generation = ++this.generation
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      const document = await this.client.call('readBrowserRecordingSource', { ...this.scope(), id })
      if (generation !== this.generation) return
      if (document.id === id) {
        try {
          const retained = await this.client.call('readPlaywrightDraft', { ...this.scope(), id })
          if (generation === this.generation) { this.accept(retained); this.message.set('This recording already has a saved test. Its saved version is open.') }
          return
        } catch (error) { if (!(error instanceof ApiError) || error.status !== 404) throw error }
      }
      this.saved.set(null); this.document.set(document); this.selectedPath.set(document.files[0]?.path ?? ''); this.sources.draftDirty.set(true)
      this.message.set('Recording opened for review. Save the test to keep this version. Start-origin navigation is bound to the selected environment in the visible source. The original recording stays in your history. Its catalog starts with a needs-review safety label.')
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  private accept(row: PlaywrightDraftRecord): void {
    this.saved.set(row); this.document.set(row.data); this.selectedPath.set(row.data.files[0]?.path ?? ''); this.sources.draftDirty.set(false); this.key = randomId()
    this.drafts.update(items => [row, ...items.filter(item => item.id !== row.id)])
  }

  editName(name: string): void { const value = this.document(); if (value) this.edit({ ...value, name }) }
  editFile(path: string, content: string): void { const value = this.document(); if (value) this.edit({ ...value, files: replaceEditorFile(value.files, path, content) }) }
  private edit(value: PlaywrightDraft): void {
    this.document.set(value); this.sources.draftDirty.set(!sameEditorDocument(value, this.saved()?.data)); this.message.set('')
  }
  discard(): void {
    const row = this.saved()
    if (row) this.accept(row)
    else { this.document.set(null); this.sources.draftDirty.set(false) }
    this.error.set(''); this.message.set('')
  }
  addFile(path: string): boolean {
    const value = this.document()
    if (!value) return false
    const error = editorPathError(path, value.files, !!value.manifestPath)
    this.error.set(error)
    if (error) return false
    this.edit({ ...value, files: [...value.files, { path, content: '' }] }); this.selectedPath.set(path)
    return true
  }
  removeFile(path: string): void {
    const value = this.document()
    if (!value || protectedEditorFile(value, path)) return
    const files = value.files.filter(file => file.path !== path)
    this.edit({ ...value, files }); this.selectedPath.set(files[0]?.path ?? '')
  }

  async save(recoverIdentical = false): Promise<void> {
    const original = this.document()
    if (!original || this.busy() || !original.name.trim()) return
    const document = { ...original, saveAsTest: true, ...(this.sources.workspace.applicationId() ? { applicationId: this.sources.workspace.applicationId() } : {}) }
    const generation = ++this.generation
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      const path = this.selectedPath()
      const row = await this.client.call('savePlaywrightDraft', { ...this.scope(), id: document.id, revision: this.saved()?.revision ?? 0, body: document })
      if (generation !== this.generation) return
      this.accept(row); this.selectedPath.set(path)
      this.message.set('Test saved. No tests were started.')
    } catch (error) {
      if (recoverIdentical && !(error instanceof ApiError && (error.status === 401 || error.status === 403))) {
        try {
          const retained = await this.client.call('readPlaywrightDraft', { ...this.scope(), id: document.id })
          if (generation === this.generation && sameEditorDocument(retained.data, document)) { this.accept(retained); this.message.set('The test was saved. Its confirmation was recovered. No tests were started.'); return }
        } catch {}
      }
      if (generation === this.generation) this.fail(error)
    }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async prepare(): Promise<void> {
    const row = this.saved()
    if (!row || this.sources.draftDirty() || this.busy() || this.preparation.busy()) return
    const generation = ++this.generation
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      const job = await this.client.call('preparePlaywrightDraft', { ...this.scope(), id: row.id, revision: row.revision, key: this.key })
      if (generation !== this.generation) return
      this.preparation.showQueued(job); this.message.set(`Revision ${row.revision} queued. Follow Suite preparation below, then review and save. No test run was started.`)
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  private fail(error: unknown): void {
    this.error.set(error instanceof ApiError && error.status === 409 ? 'This test changed on the server. Copy your unsaved code, discard local changes, then reopen the saved test before editing again.' : error instanceof Error ? error.message : 'The editor request failed.')
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) { this.document.set(null); this.saved.set(null); this.drafts.set([]); this.sources.draftDirty.set(false) }
  }
}
