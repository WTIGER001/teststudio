import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { FormControl, Validators } from '@angular/forms'
import { ApiError, ApiService, Page } from '../../core/api.service'
import { ProjectImageCaptureService } from './project-image-capture.service'
import { ProjectImageProcessingService } from './project-image-processing.service'
import { websiteAddress } from './project-image.utils'

export interface ProjectCover { image: string; source?: 'upload' | 'capture'; url?: string }
export interface CoverInfo { id: string; revision: number; hasImage: boolean; canManage: boolean }
interface CoverDetail extends CoverInfo { cover: ProjectCover }

@Injectable()
export class ProjectCoversService {
  private readonly api = inject(ApiService)
  private readonly images = inject(ProjectImageProcessingService)
  private readonly capture = inject(ProjectImageCaptureService)
  readonly covers = signal<Record<string, CoverInfo>>({})
  readonly selected = signal('')
  readonly name = signal('')
  readonly preview = signal('')
  readonly error = signal('')
  readonly listError = signal('')
  readonly busy = signal('')
  readonly ready = signal(false)
  readonly dirty = signal(false)
  readonly conflict = signal(false)
  readonly address = new FormControl('', { nonNullable: true, validators: Validators.required })
  readonly captureAvailable = this.capture.available()
  private revision = 0
  private generation = 0
  private disposed = false
  private draft: ProjectCover = { image: '' }

  constructor() {
    inject(DestroyRef).onDestroy(() => { this.disposed = true; this.generation++; this.capture.stop() })
    void this.load()
  }

  async load(): Promise<void> {
    this.listError.set('')
    try {
      const page = await this.api.request<Page<CoverInfo>>('/api/v1/project-covers')
      if (!this.disposed) this.covers.set(Object.fromEntries(page.items.map(row => [row.id, row])))
    } catch { if (!this.disposed) this.listError.set('Project images could not be loaded. Retry loading images.') }
  }

  imageURL(id: string): string { return `/api/v1/projects/${encodeURIComponent(id)}/cover/content?revision=${this.covers()[id]?.revision ?? 0}` }

  async open(id: string, name: string): Promise<void> {
    const generation = ++this.generation
    this.selected.set(id); this.name.set(name); this.preview.set(''); this.error.set(''); this.dirty.set(false); this.conflict.set(false); this.address.reset(''); this.ready.set(false); this.busy.set('loading')
    try {
      const detail = await this.api.request<CoverDetail>(`/api/v1/projects/${encodeURIComponent(id)}/cover`)
      if (generation !== this.generation || this.disposed) return
      if (!detail.canManage) throw new Error('You need project management permission to change this image.')
      this.revision = detail.revision; this.draft = detail.cover; this.preview.set(detail.cover.image); this.address.setValue(detail.cover.url ?? ''); this.ready.set(true)
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'Project image could not be loaded.'); this.conflict.set(true) } }
    finally { if (generation === this.generation) this.busy.set('') }
  }

  async upload(file: File): Promise<void> {
    if (!this.ready() || this.busy() || this.conflict()) return
    const generation = this.generation
    this.busy.set('processing'); this.error.set('')
    try {
      const image = await this.images.fromFile(file)
      if (generation === this.generation) this.setDraft({ image, source: 'upload' })
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'The image could not be read.') }
    finally { if (generation === this.generation) this.busy.set('') }
  }

  website(): string { return websiteAddress(this.address.value) }

  async captureTab(): Promise<void> {
    if (!this.ready() || this.busy() || this.conflict()) return
    const url = this.website(), generation = this.generation
    if (!url) { this.error.set('Enter an HTTP or HTTPS website address without credentials.'); return }
    this.busy.set('capturing'); this.error.set('')
    try {
      const image = await this.capture.capture()
      if (generation === this.generation) this.setDraft({ image, source: 'capture', url })
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof DOMException && error.name === 'NotAllowedError' ? 'Capture was cancelled or denied. Try again or upload a screenshot.' : 'Capture could not complete. Keep the selected tab open or upload a screenshot.') }
    finally { if (generation === this.generation) this.busy.set('') }
  }

  remove(): void { if (!this.busy() && !this.conflict()) this.setDraft({ image: '' }) }

  async save(): Promise<boolean> {
    if (!this.ready() || this.busy() || !this.dirty() || this.conflict()) return false
    const generation = this.generation
    this.busy.set('saving'); this.error.set('')
    try {
      const value = await this.api.request<CoverDetail>(`/api/v1/projects/${encodeURIComponent(this.selected())}/cover`, 'PUT', this.draft, this.revision)
      if (generation !== this.generation) return false
      this.covers.update(rows => ({ ...rows, [value.id]: value })); this.dirty.set(false)
      return true
    } catch (error) {
      if (generation === this.generation) {
        this.conflict.set(true)
        this.error.set(error instanceof ApiError && error.status === 409 ? 'This project image changed. Reload the current image before editing again.' : 'The save could not be confirmed. Reload the current image to review its saved state.')
      }
      return false
    } finally { if (generation === this.generation) this.busy.set('') }
  }

  canLeave(): boolean { return this.busy() !== 'saving' && (!this.dirty() || window.confirm('Discard this unsaved project image?')) }
  close(): boolean {
    if (!this.canLeave()) return false
    this.generation++; this.capture.stop(); this.selected.set(''); this.ready.set(false); this.preview.set(''); this.dirty.set(false); this.busy.set('')
    return true
  }
  private setDraft(value: ProjectCover): void { this.draft = value; this.preview.set(value.image); this.dirty.set(true) }
}
