import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { debounceTime } from 'rxjs'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute, Router } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError, ApiService } from '../../core/api.service'
import type { StatusPageEntry, StatusPageRecord, StatusPageReport, StatusPageDefinition, CatalogListEntry } from '../../core/generated/access.types'
import type { Page, RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { randomId } from '../../core/random-id'
import { StatusLogoProcessingService } from './status-logo-processing.service'
import { WizardController } from '../../shared/wizard/wizard-controller'
import type { ProbePage, ProbeSummary } from '../probes/probe.models'

@Injectable()
export class StatusEditorService {
  private readonly client = inject(AccessClientService); private readonly api = inject(ApiService); private readonly router = inject(Router); private readonly workspace = inject(WorkspaceService); private readonly destroy = inject(DestroyRef)
  readonly project = signal(''); readonly id = signal(''); readonly record = signal<StatusPageRecord | null>(null); readonly probes = signal<ProbeSummary[]>([]); readonly entries = signal<StatusPageEntry[]>([])
  readonly preview = signal<StatusPageReport | null>(null); readonly busy = signal(false); readonly error = signal(''); readonly message = signal(''); readonly locked = signal(false)
  readonly checks = signal<CatalogListEntry[]>([]); readonly environments = signal<{ id: string; name: string }[]>([]); readonly checksError = signal(''); readonly checksLoading = signal(false); readonly checksCursor = signal('')
  private checksLoaded = false
  readonly canManage = computed(() => this.workspace.projectCapabilities()['teststudio.statuspages.manage'] === true)
  readonly canPublish = computed(() => this.workspace.projectCapabilities()['teststudio.statuspages.publish'] === true)
  readonly canDelete = computed(() => this.workspace.projectCapabilities()['teststudio.statuspages.delete'] === true)
  readonly mode = signal<'create' | 'edit' | 'view' | 'publish'>('view')
  readonly editing = computed(() => ['create', 'edit'].includes(this.mode()))
  readonly origin = location.origin
  readonly slugState = signal<'idle' | 'checking' | 'available' | 'taken' | 'error'>('idle')
  readonly slugMessage = signal('')
  readonly publicURL = computed(() => this.record() ? location.origin + '/status/' + (this.record()!.data.published?.slug || this.record()!.data.publicToken) : '')
  readonly draftURL = computed(() => this.record() ? location.origin + '/status/' + (this.record()!.data.slug || this.record()!.data.publicToken) : '')
  readonly embed = computed(() => '<iframe src="' + this.publicURL() + '" title="Service status" style="width:100%;height:600px;border:0" loading="lazy" referrerpolicy="no-referrer"></iframe>')
  readonly logoMode = signal<'none' | 'upload' | 'url'>('none'); readonly logoUploading = signal(false); readonly logoError = signal('')
  private readonly logoProcessing = inject(StatusLogoProcessingService)
  readonly form = new FormGroup({ logo: new FormControl('', { nonNullable: true }), logoAlt: new FormControl('', { nonNullable: true }), slug: new FormControl('', { nonNullable: true }), title: new FormControl('', { nonNullable: true }), background: new FormControl('#ffffff', { nonNullable: true }), defaultWindow: new FormControl('24h', { nonNullable: true }), maxAgeSeconds: new FormControl(900, { nonNullable: true }) })
  readonly flow = new WizardController(3, step => this.validate(step))
  private generation = 0; private entriesDirty = false; private attempted = false; private slugGeneration = 0; private logoGeneration = 0
  constructor() {
    const route = inject(ActivatedRoute); this.mode.set(route.snapshot.data['mode'] || 'view')
    route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => { this.checksLoaded=false; this.checks.set([]); this.checksCursor.set(''); this.environments.set([]); this.project.set(params.get('projectId') || ''); this.id.set(params.get('pageId') || randomId()); this.record.set(null); this.preview.set(null); this.entries.set([]); this.flow.reset(); this.entriesDirty = false; this.attempted = false; this.logoMode.set('none'); this.logoError.set(''); this.form.reset({ logo: '', logoAlt: '', slug: '', title: '', background: '#ffffff', defaultWindow: '24h', maxAgeSeconds: 900 }); void this.load(params.has('pageId')) })
    this.form.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.preview.set(null) })
    this.form.controls.slug.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.slugGeneration++; this.slugState.set('idle'); this.slugMessage.set(''); if (this.error().toLowerCase().includes('vanity url')) this.error.set('') })
    this.form.controls.slug.valueChanges.pipe(debounceTime(300), takeUntilDestroyed(this.destroy)).subscribe(() => { void this.checkSlug() })
    this.form.controls.logo.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { if (this.error().toLowerCase().includes('logo')) this.error.set('') })
    this.destroy.onDestroy(() => { this.generation++; this.slugGeneration++; this.logoGeneration++ })
  }
  logoPreview(): string { const value = this.form.controls.logo.value; if (value.startsWith('data:image/png;base64,') || value.startsWith('data:image/jpeg;base64,')) return value; try { const url = new URL(value); return url.protocol === 'https:' && !url.username && !url.password ? value : '' } catch { return '' } }
  logoSource(mode: 'none' | 'upload' | 'url'): void { this.logoGeneration++; this.logoUploading.set(false); this.logoMode.set(mode); this.logoError.set(''); this.form.controls.logo.setValue(''); if (mode === 'none') this.form.controls.logoAlt.setValue(''); this.form.markAsDirty() }
  async uploadLogo(file: File): Promise<void> {
    if (this.logoUploading() || this.busy() || this.locked() || !this.canManage()) return
    const generation = ++this.logoGeneration; this.logoUploading.set(true); this.logoError.set(''); this.form.markAsDirty()
    try { const value = await this.logoProcessing.fromFile(file); if (generation === this.logoGeneration) this.form.controls.logo.setValue(value) }
    catch (error) { if (generation === this.logoGeneration) this.logoError.set(error instanceof Error ? error.message : 'The logo could not be prepared.') }
    finally { if (generation === this.logoGeneration) this.logoUploading.set(false) }
  }
  dirty(): boolean { return this.form.dirty || this.entriesDirty }
  definition(): StatusPageDefinition { return { id: this.id(), ...this.form.getRawValue(), slug: this.normalizedSlug(), defaultWindow: this.form.controls.defaultWindow.value as StatusPageDefinition['defaultWindow'], title: this.form.controls.title.value.trim(), maxAgeSeconds: Number(this.form.controls.maxAgeSeconds.value), probes: this.entries() } }
  validate(step: number): boolean {
    const value = this.definition(); let error = ''
    if (step === 0 && (!value.title || !/^#[a-fA-F0-9]{6}$/.test(value.background) || value.maxAgeSeconds < 60 || value.maxAgeSeconds > 86400)) error = 'Enter a title, valid background, and freshness from 60 through 86400 seconds.'
    if (step === 0 && this.normalizedSlug() && (!/^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(this.normalizedSlug()) || /^[a-f0-9]{40}$/.test(this.normalizedSlug()))) error = 'Use 1–63 letters, numbers, or hyphens for the vanity URL.'
    if (step === 0 && this.slugState() === 'taken') error = 'That vanity URL is already reserved. Choose another.'
    if (step === 0 && this.logoError()) error = this.logoError()
    if (step === 0 && this.logoUploading()) error = 'Wait for the logo to finish processing.'
    if (step === 0 && this.logoMode() !== 'none' && !value.logo?.trim()) error = 'Choose a logo image or select No logo.'
    if (step === 0 && this.logoMode() === 'url') { try { const url = new URL(value.logo || ''); if (url.protocol !== 'https:' || url.username || url.password) error = 'Enter a public HTTPS logo URL without embedded credentials.' } catch { error = 'Enter a public HTTPS logo URL.' } }
    if (step === 1 && (!value.probes.length || value.probes.length > 100 || value.probes.some(entry => !entry.title.trim()))) error = 'Select 1–100 checks and give each a public title.'
    this.error.set(error); return !error
  }
  choose(probe: ProbeSummary): void { if (this.entries().some(entry => entry.probeId === probe.id)) return; this.entries.update(rows => [...rows, { probeId: probe.id, title: probe.name, description: '' }]); this.changed() }
  remove(index: number): void { this.entries.update(rows => rows.filter((_, position) => position !== index)); this.changed() }
  edit(index: number, field: 'title' | 'description', value: string): void { this.entries.update(rows => rows.map((entry, position) => position === index ? { ...entry, [field]: value } : entry)); this.changed() }
  move(index: number, direction: number): void { const rows = [...this.entries()], target = index + direction; if (target < 0 || target >= rows.length) return; [rows[index], rows[target]] = [rows[target], rows[index]]; this.entries.set(rows); this.changed() }
  selected(id: string): boolean { return this.entries().some(entry => entry.probeId === id) }
  entryKey(entry: StatusPageEntry): string { return entry.probeId || [entry.suiteId, entry.testId, entry.environmentId, entry.executionLocation].join('/') }
  chooseCheck(key: string, environmentId: string, executionLocation: string, monitor: boolean): void {
    const check = this.checks().find(row => row.suiteId + '/' + row.id === key)
    if (!check) return
    const entry: StatusPageEntry = { suiteId: check.suiteId, testId: check.id, environmentId, executionLocation, monitor, title: check.data.title, description: '' }
    if (this.entries().some(row => this.entryKey(row) === this.entryKey(entry))) return
    this.entries.update(rows => [...rows, entry]); this.changed()
  }
  async loadChecks(open: boolean, more = false): Promise<void> {
    if (!open || this.checksLoading() || this.checksLoaded && !more) return
    const generation = this.generation; this.checksLoading.set(true); this.checksError.set('')
    try {
      const [page, environments] = await Promise.all([this.client.call('listProjectCatalog', { project: this.project(), query: { limit: 100, state: 'enabled', cursor: more ? this.checksCursor() : undefined } }), this.api.request<Page<RecordValue<{ id: string; name: string }>>>('/api/v1/projects/' + this.project() + '/environments')])
      if (generation !== this.generation) return
      this.checks.set(more ? [...this.checks(), ...page.items] : page.items); this.checksCursor.set(page.cursor || ''); this.environments.set(environments.items.map(row => row.data)); this.checksLoaded = true
    } catch (error) { if (generation === this.generation) this.checksError.set(error instanceof Error ? error.message : 'Checks could not be loaded.') }
    finally { if (generation === this.generation) this.checksLoading.set(false) }
  }
  private changed(): void { this.entriesDirty = true; this.preview.set(null) }
  async load(existing = true): Promise<void> {
    this.logoGeneration++; this.logoUploading.set(false); const generation = ++this.generation; this.busy.set(true); this.error.set(''); this.locked.set(false)
    try {
      const choices: ProbeSummary[] = []; let cursor = ''
      do { const page = await this.api.request<ProbePage>('/api/v1/projects/' + this.project() + '/probes?limit=100&hours=24' + (cursor ? '&cursor=' + encodeURIComponent(cursor) : '')); choices.push(...page.items); cursor = page.cursor || '' } while (cursor && generation === this.generation)
      if (generation !== this.generation) return
      this.probes.set(choices)
      if (existing) { const row = await this.client.call('readStatusPage', { project: this.project(), page: this.id() }); if (generation !== this.generation) return; this.record.set(row); this.form.reset(row.data); this.logoMode.set(row.data.logo?.startsWith('data:') ? 'upload' : row.data.logo ? 'url' : 'none'); this.logoError.set(''); this.entries.set(row.data.probes); this.entriesDirty = false; if (!this.editing()) { const preview = await this.client.call('previewStatusPage', { project: this.project(), page: this.id() }); if (generation === this.generation) this.preview.set(preview) } }
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'The status page could not be loaded.') }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  normalizedSlug(): string { return this.form.controls.slug.value.trim().toLowerCase() }
  async checkSlug(): Promise<void> {
    if (!this.editing() || !this.canManage()) return
    const slug = this.normalizedSlug(), generation = ++this.slugGeneration
    if (!slug) { this.slugState.set('idle'); this.slugMessage.set('Leave blank to use the permanent generated URL.'); return }
    if (!/^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(slug) || /^[a-f0-9]{40}$/.test(slug)) { this.slugState.set('error'); this.slugMessage.set('Use 1–63 letters, numbers, or hyphens. Generated addresses are reserved.'); return }
    this.slugState.set('checking'); this.slugMessage.set('Checking availability…')
    try { const result = await this.client.call('checkStatusPageSlug', { project: this.project(), slug, query: { pageId: this.id() } }); if (generation !== this.slugGeneration || slug !== this.normalizedSlug()) return; this.slugState.set(result.available ? 'available' : 'taken'); this.slugMessage.set(result.available ? 'Available for this page. Reserved when you save.' : 'That name is already reserved. Choose another.') }
    catch { if (generation === this.slugGeneration) { this.slugState.set('error'); this.slugMessage.set('Availability could not be checked. Saving still validates uniqueness.') } }
  }
  async save(): Promise<void> {
    if (this.busy() || this.locked() || !this.canManage() || !this.flow.ready()) return
    this.attempted = true
    await this.request(() => this.client.call('saveStatusPage', { project: this.project(), page: this.id(), revision: this.record()?.revision || 0, body: this.definition() }), async row => {
      this.record.set(row); this.form.markAsPristine(); this.entriesDirty = false; this.attempted = false; this.message.set('Private draft saved. Publish separately to update the public page.')
      await this.router.navigate(['/projects', this.project(), 'status-pages', this.id()], { replaceUrl: this.mode() === 'create' })
    })
  }
  async showPreview(window?: string): Promise<void> {
    if (!this.record() || this.dirty() || this.busy()) return
    await this.request(() => this.client.call('previewStatusPage', { project: this.project(), page: this.id(), query: { window } }), report => { this.preview.set(report) }, false)
  }
  async publish(): Promise<void> {
    if (!this.record() || this.dirty() || this.busy() || this.locked() || !this.canPublish() || this.mode() !== 'publish' || !this.preview()) return
    await this.request(() => this.client.call('publishStatusPage', { project: this.project(), page: this.id(), revision: this.record()!.revision }), async row => { this.record.set(row); await this.router.navigate(['/projects', this.project(), 'status-pages', this.id()]) })
  }
  async unpublish(): Promise<void> {
    if (!this.record() || this.busy() || this.locked() || !this.canPublish() || !window.confirm('Unpublish this page? Existing embeds will stop showing status on their next refresh.')) return
    await this.request(() => this.client.call('unpublishStatusPage', { project: this.project(), page: this.id(), revision: this.record()!.revision }), row => { this.record.set(row); this.message.set('Public access withdrawn. Your draft remains.') })
  }
  async delete(): Promise<void> {
    if (!this.record() || this.busy() || this.locked() || !this.canDelete() || !window.confirm('Delete this status page and withdraw its public URL?')) return
    await this.request(() => this.client.call('deleteStatusPage', { project: this.project(), page: this.id(), revision: this.record()!.revision }), async () => { this.form.markAsPristine(); this.entriesDirty = false; await this.router.navigate(['/projects', this.project(), 'status-pages']) })
  }
  async reload(): Promise<void> { if (this.dirty() && !window.confirm('Discard unsaved values and reload the saved status page?')) return; this.flow.reset(); await this.load(!!this.record() || this.attempted || this.router.url.includes('/' + this.id())) }
  private async request<T>(operation: () => Promise<T>, apply: (value: T) => void | Promise<void>, mutation = true): Promise<void> {
    const generation = this.generation; this.busy.set(true); this.error.set(''); this.message.set('')
    try { const value = await operation(); if (generation === this.generation) await apply(value) }
    catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The operation could not be confirmed.'); const duplicate = error instanceof ApiError && error.status === 409 && error.message.startsWith('that vanity URL'); if (duplicate) { this.slugState.set('taken'); this.slugMessage.set('That name is reserved. Choose another.'); this.flow.go(0) }; this.locked.set(mutation && !duplicate && (!(error instanceof ApiError) || error.status === 409 || error.status >= 500)) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
}
