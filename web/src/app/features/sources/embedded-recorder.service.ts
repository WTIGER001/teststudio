import { computed, DestroyRef, effect, inject, Injectable, signal, untracked } from '@angular/core'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { BrowserRecordingAction, BrowserRecordingView } from '../../core/generated/access.types'
import { BrowserRecorderService } from './browser-recorder.service'

@Injectable()
export class EmbeddedRecorderService {
  readonly recorder = inject(BrowserRecorderService)
  private readonly client = inject(AccessClientService)
  readonly view = signal<BrowserRecordingView | null>(null)
  readonly error = signal('')
  readonly pending = signal(false)
  readonly uncertain = signal(false)
  readonly checking = computed(() => ['assertingVisibility', 'inspecting'].includes(this.view()?.mode ?? ''))
  readonly external = signal(false)
  readonly checkKind = signal<'at-least' | 'exactly' | 'text'>('at-least')
  readonly count = signal(1)
  readonly expectedText = signal('')
  readonly width = signal(1280)
  readonly height = signal(720)
  readonly sizeError = signal('')
  readonly presetSelection = computed(() => { const value = `${this.view()?.width}x${this.view()?.height}`; return ['1280x720', '1920x1080', '768x1024', '390x844'].includes(value) ? value : 'custom' })
  private channel: BroadcastChannel | null = null
  readonly text = signal('')
  private generation = 0
  private recordingId = ''
  private reading = false
  private queue: Omit<BrowserRecordingAction, 'sequence'>[] = []

  constructor() {
    effect(() => {
      const row = this.recorder.selected()
      const id = row?.data.mode === 'embedded' && row.data.state === 'recording' ? row.id : ''
      if (id === this.recordingId) return
      this.generation++; this.error.set(''); this.uncertain.set(false); this.channel?.close(); this.channel = null
      this.recordingId = id
      this.reading = false; this.queue = []; this.pending.set(false); this.view.set(null); this.recorder.sources.recordingInputPending.set(false)
      this.external.set(false)
      if (id) {
        if (typeof BroadcastChannel !== 'undefined') {
          this.channel = new BroadcastChannel(`teststudio-recording-${id}`)
          this.channel.onmessage = event => { if (event.data === 'claim') this.external.set(true) }
          this.channel.postMessage('claim')
        }
        untracked(() => { void this.refresh() })
      } else {
        this.generation++; this.view.set(null); this.queue = []; this.pending.set(false); this.reading = false; this.recorder.sources.recordingInputPending.set(false)
      }
    })
    const timer = setInterval(() => { if (this.recorder.selected()?.data.state === 'recording' && !this.pending() && !this.uncertain() && !this.external()) void this.refresh() }, 700)
    inject(DestroyRef).onDestroy(() => { this.generation++; this.channel?.close(); this.queue = []; clearInterval(timer); this.view.set(null); this.recorder.sources.recordingInputPending.set(false) })
  }

  private scope() {
    return { project: this.recorder.sources.project(), suite: this.recorder.sources.form.controls.suite.value, id: this.recorder.selected()!.id }
  }

  async refresh(): Promise<void> {
    if (this.reading || this.pending() || !this.recordingId || this.recorder.selected()?.id !== this.recordingId || this.recorder.selected()?.data.mode !== 'embedded' || this.recorder.selected()?.data.state !== 'recording') return
    const generation = this.generation
    this.reading = true
    try {
      const view = await this.client.call('readBrowserRecordingView', this.scope())
      if (generation === this.generation) { this.accept(view); this.error.set(''); this.uncertain.set(false) }
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) { this.reading = false; if (this.queue.length) void this.drain() } }
  }

  send(action: Omit<BrowserRecordingAction, 'sequence'>): void {
    if (this.external() || this.uncertain() || !this.view() || this.recorder.selected()?.data.state !== 'recording') return
    if (this.queue.length >= 100) { this.error.set('Please wait for the browser to catch up.'); return }
    this.queue.push(action)
    this.recorder.sources.recordingInputPending.set(true)
    void this.drain()
  }

  private async drain(): Promise<void> {
    if (this.pending() || this.reading || this.uncertain() || this.external()) return
    const generation = this.generation
    this.pending.set(true)
    try {
      while (!this.external() && this.queue.length && generation === this.generation) {
        const action = this.queue.shift()!
        const view = await this.client.call('sendBrowserRecordingInput', { ...this.scope(), body: { ...action, sequence: this.view()!.sequence + 1 } })
        if (generation === this.generation) this.accept(view)
      }
    } catch (error) {
      if (generation === this.generation) { this.queue = []; this.uncertain.set(true); this.fail(error) }
    } finally { if (generation === this.generation) { this.pending.set(false); if (this.external()) this.queue = []; this.recorder.sources.recordingInputPending.set(false) } }
  }

  private accept(view: BrowserRecordingView): void {
    if (view.width !== this.view()?.width || view.height !== this.view()?.height) { this.width.set(view.width); this.height.set(view.height) }
    this.view.set(view)
  }

  async useHere(): Promise<void> {
    if (this.pending()) return
    this.external.set(false); this.channel?.postMessage('claim')
    await this.refresh()
  }

  openTab(): void {
    if (this.pending() || this.uncertain()) return
    const row = this.recorder.selected()
    if (!row) return
    const path = this.recorder.sources.workspace.path('tests', 'recording', this.recorder.sources.form.controls.suite.value, row.id)
    const tab = window.open(path.map(segment => encodeURIComponent(segment).replace('%2F', '/')).join('/'), '_blank')
    if (!tab) { this.error.set('Your browser blocked the new tab. Allow popups for TestStudio, or use Open recording dialog.'); return }
    tab.opener = null
    this.external.set(true)
  }

  chooseCheck(): void { this.expectedText.set(''); this.send({ action: 'assert-pick' }) }
  addCheck(): void { this.send({ action: 'assert-add', key: this.checkKind(), ...(this.checkKind() === 'text' ? { text: this.expectedText() } : { count: Number(this.count()) }) }) }
  resize(): void {
    const width = Number(this.width()), height = Number(this.height())
    if (!Number.isInteger(width) || width < 320 || width > 1920 || !Number.isInteger(height) || height < 240 || height > 1200) { this.sizeError.set('Use whole numbers: width 320–1920 and height 240–1200.'); return }
    this.sizeError.set(''); this.send({ action: 'resize', width, height })
  }
  preset(value: string): void { const [width, height] = value.split('x').map(Number); this.width.set(width); this.height.set(height); this.resize() }

  point(event: MouseEvent): void {
    const image = event.currentTarget as HTMLElement, bounds = image.getBoundingClientRect(), view = this.view()
    if (!view) return
    image.focus()
    this.send({ action: 'click', x: Math.min(view.width - 1, Math.max(0, (event.clientX - bounds.left - image.clientLeft) * view.width / image.clientWidth)), y: Math.min(view.height - 1, Math.max(0, (event.clientY - bounds.top - image.clientTop) * view.height / image.clientHeight)) })
  }

  key(event: KeyboardEvent): void {
    if (event.key === 'Tab' || event.key === 'Escape') return
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'a') { event.preventDefault(); this.send({ action: 'key', key: 'ControlOrMeta+A' }); return }
    if (event.ctrlKey || event.metaKey || event.altKey || event.isComposing) return
    if (event.key.length === 1) { event.preventDefault(); this.send({ action: 'text', text: event.key }); return }
    if (['Enter', 'Backspace', 'Delete', 'ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) { event.preventDefault(); this.send({ action: 'key', key: event.key }) }
  }

  paste(event: ClipboardEvent): void {
    const text = event.clipboardData?.getData('text/plain')
    if (text) { event.preventDefault(); this.send({ action: 'text', text: text.slice(0, 2048) }) }
  }

  enterText(): void { if (this.text()) { this.send({ action: 'text', text: this.text() }); this.text.set('') } }

  private fail(error: unknown): void {
    this.error.set(error instanceof Error ? error.message : 'The recording browser could not update. Refresh its view to reconnect.')
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) { this.view.set(null); this.queue = []; this.uncertain.set(true) }
  }
}
