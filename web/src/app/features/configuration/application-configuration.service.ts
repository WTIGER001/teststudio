import { DestroyRef, effect, inject, Injectable, signal, untracked } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormArray, FormControl, FormGroup, Validators } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { ApiError, ApiService, type RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import type { Application, Environment } from './configuration.service'
import type { CredentialReference } from './configuration.utils'
import type { ApplicationConfiguration, EnvironmentConfiguration, Requirement } from './application-configuration.models'

function requirementForm(value?: Requirement) {
  return new FormGroup({ name: new FormControl(value?.name ?? '', { nonNullable: true, validators: [Validators.required, Validators.pattern(/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/)] }), kind: new FormControl(value?.kind ?? 'parameter', { nonNullable: true }), valueType: new FormControl(value?.valueType ?? 'text', { nonNullable: true }), description: new FormControl(value?.description ?? '', { nonNullable: true }), required: new FormControl(value?.required ?? false, { nonNullable: true }), defaultValue: new FormControl(value?.defaultValue ?? '', { nonNullable: true }) })
}
function valueForm(name: string, kind: string, value: string) { return new FormGroup({ name: new FormControl(name, { nonNullable: true, validators: [Validators.required, Validators.pattern(/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/)] }), kind: new FormControl(kind, { nonNullable: true }), value: new FormControl(value, { nonNullable: true }) }) }

@Injectable()
export class ApplicationConfigurationService {
  readonly workspace = inject(WorkspaceService)
  private readonly api = inject(ApiService)
  private readonly route = inject(ActivatedRoute)
  readonly data = signal<ApplicationConfiguration | null>(null)
  readonly references = signal<CredentialReference[]>([])
  readonly error = signal('')
  readonly message = signal('')
  readonly busy = signal(false)
  readonly conflict = signal(false)
  readonly mode = signal<'contract' | 'environment' | ''>('')
  readonly review = signal(false)
  readonly selected = signal<EnvironmentConfiguration | null>(null)
  readonly requirements = new FormArray<ReturnType<typeof requirementForm>>([])
  readonly values = new FormArray<ReturnType<typeof valueForm>>([])
  readonly contractForm = new FormGroup({ requirements: this.requirements })
  readonly environmentForm = new FormGroup({ values: this.values })
  private project = ''
  private application = ''
  private generation = 0
  private loaded = ''

  constructor() {
    const destroy = inject(DestroyRef)
    this.route.paramMap.pipe(takeUntilDestroyed(destroy)).subscribe(params => { this.generation++; this.project = params.get('projectId') ?? ''; this.application = params.get('applicationId') ?? ''; this.loaded = ''; this.data.set(null); this.references.set([]); this.message.set(''); this.busy.set(false); this.resetEditor() })
    effect(() => {
      const project = this.workspace.projectId(), application = this.workspace.applicationId(), capabilities = this.workspace.capabilities()
      if (project === this.project && application === this.application && Object.keys(capabilities).length) untracked(() => { const key = `${project}/${application}`; if (this.loaded !== key) { this.loaded = key; void this.load() } })
    })
    destroy.onDestroy(() => { this.generation++ })
  }
  base(): string { return `/api/v1/projects/${this.project}/applications/${this.application}` }
  dirty(): boolean { return this.contractForm.dirty || this.environmentForm.dirty }
  canLeave(): boolean { return !this.busy() && (!this.dirty() || window.confirm('Configuration has unsaved changes. Leave without saving?')) }
  private resetEditor(): void { this.mode.set(''); this.review.set(false); this.selected.set(null); this.requirements.clear(); this.values.clear(); this.contractForm.markAsPristine(); this.environmentForm.markAsPristine(); this.conflict.set(false); this.error.set('') }
  cancel(): void { if (!this.busy() && (!this.dirty() || window.confirm('Discard unsaved configuration changes?'))) this.resetEditor() }

  async load(): Promise<void> {
    const generation = ++this.generation
    this.busy.set(true); this.error.set('')
    try {
      const [data, references] = await Promise.all([this.api.request<ApplicationConfiguration>(`${this.base()}/configuration`), this.workspace.can('environments.manage') ? this.api.request<{ items: CredentialReference[] }>(`${this.base()}/credential-references`) : Promise.resolve({ items: [] })])
      if (generation !== this.generation) return
      this.data.set(data); this.references.set(references.items)
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  async reload(): Promise<void> { if (this.dirty() && !window.confirm('Reload current configuration and discard unsaved changes?')) return; this.resetEditor(); await this.load() }
  async createDefaultSuite(): Promise<void> {
    const data = this.data(), generation = this.generation
    if (!data || this.busy() || this.dirty()) return
    this.busy.set(true)
    try { await this.api.request(`${this.base()}/default-suite`, 'POST', {}, data.application.revision); if (generation !== this.generation) return; this.message.set('Default suite created. No tests or runs were created.'); await this.load() }
    catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  editContract(): void {
    if (this.busy() || this.dirty() || !this.workspace.can('applications.manage')) return
    this.resetEditor(); this.message.set(''); this.mode.set('contract')
    for (const r of this.data()?.application.data.requirements ?? []) this.requirements.push(requirementForm(r))
  }
  addRequirement(): void { if (this.requirements.length < 64) { this.requirements.push(requirementForm()); this.contractForm.markAsDirty() } }
  removeRequirement(index: number): void { this.requirements.removeAt(index); this.contractForm.markAsDirty() }
  editEnvironment(row: EnvironmentConfiguration): void {
    if (this.busy() || this.dirty() || !this.workspace.can('environments.manage')) return
    this.resetEditor(); this.message.set(''); this.mode.set('environment'); this.selected.set(row)
    const requirements = this.data()?.requirements ?? []
    const entries: Record<string, Record<string, string>> = { endpoint: { ...row.data.endpoints }, parameter: { ...row.data.parameters }, secret: { ...row.data.credentialBindings } }
    if (!Object.keys(entries['endpoint']).length) entries['endpoint']['api'] = ''
    for (const r of requirements) if (!(r.name in entries[r.kind])) entries[r.kind][r.name] = ''
    for (const [kind, values] of Object.entries(entries)) for (const [name, value] of Object.entries(values)) this.values.push(valueForm(name, kind, value))
  }
  status(row: EnvironmentConfiguration, requirement: Requirement): string { return row.readiness.items.find(item => item.name === requirement.name && item.kind === requirement.kind)?.status ?? 'missing' }
  description(row: EnvironmentConfiguration, requirement: Requirement): string { return row.readiness.items.find(item => item.name === requirement.name && item.kind === requirement.kind)?.message ?? '' }
  proposal(): Requirement[] { return this.requirements.getRawValue().map(r => ({ name: r.name, kind: r.kind, description: r.description, required: r.required, ...(r.kind === 'parameter' ? { valueType: r.valueType, ...(r.defaultValue !== '' ? { defaultValue: r.defaultValue } : {}) } : {}) })) }
  environmentProposal(): Pick<Environment, 'endpoints' | 'parameters' | 'credentialBindings'> {
    const values = this.values.getRawValue(), map = (kind: string) => Object.fromEntries(values.filter(row => row.kind === kind && row.value !== '').map(row => [row.name, row.value]))
    return { endpoints: map('endpoint'), parameters: map('parameter'), credentialBindings: map('secret') }
  }
  beginReview(): void {
    this.error.set('')
    const form = this.mode() === 'contract' ? this.contractForm : this.environmentForm
    form.markAllAsTouched()
    if (form.invalid) { this.error.set('Enter valid configuration names before reviewing.'); return }
    const names = this.mode() === 'contract' ? this.requirements.getRawValue().map(r => r.name) : this.values.getRawValue().map(r => `${r.kind}/${r.name}`)
    if (new Set(names).size !== names.length) { this.error.set('Configuration names must be unique.'); return }
    this.review.set(true)
  }
  async save(): Promise<void> {
    const data = this.data(), selected = this.selected(), generation = this.generation
    if (!data || !this.review() || this.busy() || this.conflict()) return
    this.busy.set(true); this.error.set('')
    try {
      if (this.mode() === 'contract') await this.api.request<RecordValue<Application>>(`${this.base()}/configuration`, 'PUT', { requirements: this.proposal() }, data.application.revision)
      else if (selected) await this.api.request(`${this.base()}/environments/${selected.id}/configuration`, 'PUT', this.environmentProposal(), selected.revision)
      if (generation !== this.generation) return
      this.resetEditor(); this.message.set('Configuration saved. No test run was started.'); await this.load()
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  private fail(error: unknown): void {
    this.error.set(error instanceof Error ? error.message : 'Configuration could not be saved.')
    if (error instanceof ApiError && error.status === 409) { this.conflict.set(true); this.error.set('Configuration changed. Your edits remain here. Reload current configuration before saving again.') }
    if (error instanceof ApiError && [401, 403].includes(error.status)) { this.data.set(null); this.references.set([]); this.resetEditor(); this.error.set('Configuration access is unavailable. Refresh access to continue.') }
  }
}
