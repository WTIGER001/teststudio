import { ActivatedRoute } from '@angular/router'
import { randomId } from '../../core/random-id'
import { DOCUMENT } from '@angular/common'
import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { BrowserRecordingPairing, BrowserRecordingRecord } from '../../core/generated/access.types'
import { PlaywrightEditorService } from './playwright-editor.service'
import { desktopRecorderCommand, recordingActive, recordingFlow, recordingPairingExpired, recordingRows, recordingStateLabel } from './browser-recorder.utils'

@Injectable()
export class BrowserRecorderService {
  readonly editor = inject(PlaywrightEditorService)
  readonly sources = this.editor.sources
  private readonly client = inject(AccessClientService)
  readonly enabled = signal(false)
  readonly mode = signal('disabled')
  readonly busy = signal(false)
  readonly error = signal('')
  readonly selected = signal<BrowserRecordingRecord | null>(null)
  readonly items = signal<BrowserRecordingRecord[]>([])
  readonly cursor = signal('')
  readonly pairing = signal<BrowserRecordingPairing | null>(null)
  readonly copied = signal('')
  readonly savedRecording = signal(false)
  readonly now = signal(Date.now())
  readonly flow = computed(() => recordingFlow(this.mode(), this.selected()?.data.state))
  readonly pairingExpired = computed(() => recordingPairingExpired(this.pairing(), this.now()))
  readonly stateLabel = recordingStateLabel
  readonly command = desktopRecorderCommand(inject(DOCUMENT).location.origin)
  readonly form = new FormGroup({
    url: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.maxLength(2048), Validators.pattern(/^https?:\/\//)] }),
    testId: new FormControl(`REC-${randomId().slice(0, 8)}`, { nonNullable: true, validators: [Validators.required, Validators.pattern(/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/)] }),
    title: new FormControl('Recorded browser flow', { nonNullable: true, validators: [Validators.required, Validators.maxLength(120)] })
  })
  private readonly route = inject(ActivatedRoute)
  private requested = this.route.snapshot.paramMap.get('recordingId') ?? this.route.snapshot.queryParamMap.get('recording') ?? ''
  private generation = 0
  private recordingDraftId = ''
  private requestId = randomId()
  private reading = false

  constructor() {
    const destroy = inject(DestroyRef)
    this.sources.form.controls.suite.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.selected.set(null); this.pairing.set(null); this.copied.set(''); this.savedRecording.set(false); this.recordingDraftId = ''; this.sources.creationDirty.set(false); this.reading = false; this.requestId = randomId(); void this.load() })
    this.form.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.requestId = randomId(); this.sources.creationDirty.set(true) })
    const timer = setInterval(() => { this.now.set(Date.now()); const row = this.selected(); if (row && recordingActive(row.data.state) && !this.busy() && !this.reading) void this.read(row.id) }, 1500)
    destroy.onDestroy(() => { this.generation++; this.pairing.set(null); this.sources.creationDirty.set(false); clearInterval(timer) })
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
      this.enabled.set(options.enabled); this.mode.set(options.mode); this.sources.recordingMode.set(options.mode); this.items.set(more ? [...this.items(), ...page.items] : page.items); this.cursor.set(page.cursor ?? '')
      if (!this.selected()) {
        const requested = this.requested; this.requested = ''
        const active = page.items.find(row => requested ? row.id === requested : recordingActive(row.data.state))
        if (active) { this.accept(active); void this.read(active.id) }
        else if (requested) await this.read(requested)
      }
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async start(): Promise<void> {
    if (this.form.invalid || this.busy() || !this.enabled() || this.active()) return
    const generation = ++this.generation
    this.busy.set(true); this.error.set('')
    try {
      const row = await this.client.call('startBrowserRecording', { ...this.scope(), body: { id: this.requestId, ...this.form.getRawValue() } })
      if (generation === this.generation) { this.sources.creationDirty.set(false); this.accept(row); if (row.data.state === 'pairing') await this.issuePairing(row.id, generation) }
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async saveRecording(): Promise<void> {
    const row = this.selected()
    if (!row || row.data.state !== 'ready' || this.editor.busy() || this.savedRecording()) return
    if (this.sources.draftDirty() && !this.recordingDraftId) { this.error.set('Save or discard the open source draft before saving this recording.'); return }
    if (!this.recordingDraftId) {
      await this.editor.importRecording(row.id)
      if (!this.editor.error()) this.recordingDraftId = this.editor.document()?.id ?? ''
    }
    if (this.editor.document()?.id !== this.recordingDraftId) { this.error.set('Reopen this recording before saving its draft.'); return }
    if (!this.editor.document()) return
    if (this.editor.saved()?.id === this.recordingDraftId && !this.sources.draftDirty()) { this.savedRecording.set(true); return }
    await this.editor.save(true)
    if (!this.editor.error() && this.editor.saved()) this.savedRecording.set(true)
  }

  async prepareRecording(): Promise<void> {
    if (!this.savedRecording() || !this.editor.preparation.bundleRunners().includes('playwright') || !this.sources.workspace.can('catalog.import')) return
    await this.editor.prepare()
    if (!this.editor.error() && this.editor.preparation.selected()) document.getElementById('preparation-title')?.focus()
  }

  async reviewSource(id: string): Promise<void> {
    this.sources.sourceEditingOpen.set(true)
    if (this.savedRecording() && this.editor.document()?.id === this.recordingDraftId) return
    await this.editor.importRecording(id)
    if (!this.editor.error() && this.selected()?.data.mode === 'embedded') this.recordingDraftId = this.editor.document()?.id ?? ''
  }

  active(): boolean { return recordingActive(this.selected()?.data.state) }

  async connect(): Promise<void> {
    const row = this.selected()
    if (this.busy() || row?.data.state !== 'pairing') return
    const generation = this.generation
    this.busy.set(true); this.error.set(''); this.pairing.set(null); this.copied.set('')
    try { await this.issuePairing(row.id, generation) }
    catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  private async issuePairing(id: string, generation: number): Promise<void> {
    const value = await this.client.call('createBrowserRecordingPairing', { ...this.scope(), id })
    if (generation === this.generation && this.selected()?.id === id && this.selected()?.data.state === 'pairing') { this.now.set(Date.now()); this.pairing.set(value) }
  }

  async copy(value: 'command' | 'code'): Promise<void> {
    const generation = this.generation, pairing = this.pairing()
    if (value === 'code' && (!pairing || this.pairingExpired() || this.selected()?.data.state !== 'pairing')) return
    try {
      await navigator.clipboard.writeText(value === 'command' ? this.command : `${pairing!.id}:${pairing!.token}`)
      if (generation === this.generation) this.copied.set(value === 'command' ? 'Desktop command copied.' : 'Connection code copied. Paste it only into your desktop recorder prompt.')
    } catch { if (generation === this.generation) this.copied.set('Automatic copy is unavailable. Select and copy the visible field.') }
  }

  newRecording(): void {
    if (this.active() || this.busy() || this.sources.draftDirty()) return
    const previous = this.selected()?.data
    this.generation++; this.requestId = randomId(); this.reading = false
    this.recordingDraftId = ''
    this.form.reset({ url: previous?.url ?? '', title: previous?.title ?? 'Recorded browser flow', testId: `REC-${randomId().slice(0, 8)}` }, { emitEvent: false })
    this.sources.creationDirty.set(false)
    this.selected.set(null); this.pairing.set(null); this.copied.set(''); this.error.set(''); this.savedRecording.set(false)
  }

  resetConfiguration(): void {
    if (this.selected() || this.busy()) return
    this.requestId = randomId()
    this.form.reset({ url: '', title: 'Recorded browser flow', testId: `REC-${randomId().slice(0, 8)}` }, { emitEvent: false })
    this.sources.creationDirty.set(false); this.error.set('')
  }

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

  private accept(row: BrowserRecordingRecord): void {
    if (this.selected()?.id !== row.id) { this.savedRecording.set(false); this.recordingDraftId = '' }
    if (row.data.state !== 'pairing' || this.selected()?.id !== row.id) { this.pairing.set(null); this.copied.set('') }
    this.selected.set(row); this.items.update(items => recordingRows(items, row))
  }
  private fail(error: unknown): void {
    this.error.set(error instanceof Error ? error.message : 'The browser recorder request failed.')
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) { this.selected.set(null); this.pairing.set(null); this.copied.set(''); this.items.set([]); this.enabled.set(false) }
  }
}
