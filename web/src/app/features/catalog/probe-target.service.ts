import { randomId } from '../../core/random-id'
import { computed, DestroyRef, effect, inject, Injectable, signal, untracked } from '@angular/core'
import { toSignal } from '@angular/core/rxjs-interop'
import { FormControl } from '@angular/forms'
import { ApiError, ApiService, Page, RecordValue } from '../../core/api.service'
import { RecordCreationService } from '../../core/record-creation.service'
import { applicationEnvironments } from '../../core/application-scope.utils'
import { WorkspaceService } from '../../core/workspace.service'
import type { Application, Environment } from '../configuration/configuration.service'
import { suggestedIdentifier } from '../../shared/wizard/creation.utils'
import { CreateTestService } from './create-test.service'
import { matchProbeAddress, parseProbeAddress } from './probe-address.utils'
import { probeTargets, probeURL } from './https-wizard.utils'

@Injectable()
export class ProbeTargetService {
  private readonly creation = inject(CreateTestService)
  private readonly api = inject(ApiService)
  private readonly records = inject(RecordCreationService)
  readonly workspace = inject(WorkspaceService)
  readonly mode = new FormControl('url', { nonNullable: true })
  readonly address = new FormControl('', { nonNullable: true, validators: control => parseProbeAddress(control.value) ? null : { address: true } })
  readonly environment = new FormControl('', { nonNullable: true })
  readonly choice = new FormControl('', { nonNullable: true })
  readonly application = new FormControl('', { nonNullable: true })
  readonly stage = new FormControl('development', { nonNullable: true })
  readonly protected = new FormControl(false, { nonNullable: true })
  private readonly selectedMode = toSignal(this.mode.valueChanges, { initialValue: 'url' })
  private readonly selectedAddress = toSignal(this.address.valueChanges, { initialValue: '' })
  private readonly selectedEnvironment = toSignal(this.environment.valueChanges, { initialValue: '' })
  private readonly selectedChoice = toSignal(this.choice.valueChanges, { initialValue: '' })
  private readonly selectedApplication = toSignal(this.application.valueChanges, { initialValue: '' })
  private readonly suiteId = toSignal(this.creation.native.form.controls.suite.valueChanges, { initialValue: this.creation.native.form.controls.suite.value })
  private readonly allEnvironments = signal<RecordValue<Environment>[]>([])
  private readonly allApplications = signal<RecordValue<Application>[]>([])
  readonly loading = signal(false)
  readonly loaded = signal(false)
  readonly error = signal('')
  readonly locked = signal(false)
  readonly targetSaved = signal(false)
  readonly applicationSaved = signal(false)
  readonly direct = computed(() => this.selectedMode() === 'url')
  readonly parsed = computed(() => parseProbeAddress(this.selectedAddress()))
  private readonly suite = computed(() => this.creation.native.suites().find(row => row.id === this.suiteId())?.data)
  readonly environments = computed(() => probeTargets(applicationEnvironments(this.allEnvironments(), this.creation.sources.applicationId()), this.creation.native.suites(), this.suiteId()))
  readonly matches = computed(() => matchProbeAddress(this.selectedAddress(), applicationEnvironments(this.allEnvironments(), this.creation.sources.applicationId()), this.suite()))
  readonly match = computed(() => this.matches().length === 1 ? this.matches()[0] : this.matches().find(row => row.key === this.selectedChoice()))
  readonly target = computed(() => this.direct() ? this.match()?.environment : this.environments().find(row => row.id === this.selectedEnvironment()))
  readonly endpoints = computed(() => Object.keys(this.target()?.data.endpoints ?? {}))
  readonly applications = computed(() => {
    const active = this.creation.sources.applicationId()
    const linked = active ? [active] : this.suite()?.applications ?? []
    return linked.length ? linked.map(id => ({ id, name: this.allApplications().find(row => row.id === id)?.data.name || id })) : this.allApplications().map(row => ({ id: row.id, name: row.data.name }))
  })
  readonly applicationId = computed(() => this.applications().length === 1 ? this.applications()[0].id : this.selectedApplication())
  readonly needsTarget = computed(() => this.direct() && !!this.parsed() && !this.matches().length)
  readonly needsApplication = computed(() => this.needsTarget() && this.loaded() && !this.applications().length)
  private readonly suffix = randomId().slice(0, 8)
  private disposed = false
  private generation = 0

  constructor() {
    const destroy = inject(DestroyRef)
    effect(() => { const project = this.creation.sources.project(), capabilities = this.workspace.capabilities(); if (project && this.workspace.projectId() === project && Object.keys(capabilities).length) untracked(() => { void this.load(project) }) })
    effect(() => { const rows = this.environments(); untracked(() => { if (!rows.some(row => row.id === this.environment.value)) this.environment.setValue(rows.length === 1 ? rows[0].id : '') }) })
    effect(() => {
      const direct = this.direct(), address = this.parsed(), match = this.match(), target = this.target()
      untracked(() => {
        if (this.locked()) return
        const check = this.creation.native.checkEditor.form.controls
        if (direct && address) { check.endpoint.setValue(match?.endpoint || 'api'); check.path.setValue(match?.path || address.path) }
        else if (!direct) { const endpoint = Object.keys(target?.data.endpoints ?? {})[0]; if (endpoint) check.endpoint.setValue(endpoint) }
      })
    })
    destroy.onDestroy(() => { this.disposed = true; this.generation++ })
  }

  async load(project = this.creation.sources.project()): Promise<void> {
    if (this.locked()) return
    const generation = ++this.generation
    this.loading.set(true); this.loaded.set(false); this.error.set('')
    try {
      const base = `/api/v1/projects/${project}`
      const [environments, applications] = await Promise.all([
        this.workspace.can('environments.read') ? this.api.request<Page<RecordValue<Environment>>>(`${base}/environments`) : Promise.resolve({ items: [] }),
        this.workspace.can('projects.read') ? this.api.request<Page<RecordValue<Application>>>(`${base}/applications`) : Promise.resolve({ items: [] })
      ])
      if (generation !== this.generation || this.disposed) return
      this.allEnvironments.set(environments.items); this.allApplications.set(applications.items); this.loaded.set(true)
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Target options could not be loaded.') }
    finally { if (generation === this.generation) this.loading.set(false) }
  }

  proposedApplicationId(): string { return this.needsApplication() ? suggestedIdentifier(this.parsed()?.hostname || '', this.suffix) : this.applicationId() }
  proposedEnvironmentId(): string { return suggestedIdentifier(this.parsed()?.hostname || '', this.suffix) }
  applicationName(): string { return this.needsApplication() ? this.parsed()?.hostname || '' : this.applications().find(row => row.id === this.applicationId())?.name || this.applicationId() }
  requestURL(): string { return this.direct() ? this.parsed() ? this.address.value : 'Enter the website URL.' : probeURL(this.target()?.data.endpoints[this.creation.native.checkEditor.form.controls.endpoint.value], this.creation.native.checkEditor.form.controls.path.value) }
  environmentId(): string { return this.target()?.id || (this.targetSaved() ? this.proposedEnvironmentId() : '') }

  issue(run = false): string {
    if (!this.direct()) return run && (!this.target() || !this.endpoints().includes(this.creation.native.checkEditor.form.controls.endpoint.value)) ? 'Choose a registered environment with this endpoint before running.' : ''
    if (!this.parsed()) return 'Enter a complete HTTP or HTTPS URL without credentials or a fragment. HTTPS needs no port; it uses 443 automatically.'
    if (!this.loaded()) return this.error() || 'Wait for target options to load, or retry loading them.'
    if (this.matches().length && !this.match()) return 'Several registered targets match this address. Choose which target and its protection settings to use.'
    if (run && this.target()?.data.disabled) return 'This registered target is disabled. Save a test or choose an enabled registered target.'
    if (this.needsTarget()) {
      if (!this.workspace.can('environments.manage')) return 'Saving this new address requires environment management permission. Choose an existing registered target instead.'
      if (this.needsApplication() && !this.workspace.can('projects.manage')) return 'Saving a new application for this address requires project management permission.'
      if (!this.needsApplication() && !this.applications().some(row => row.id === this.applicationId())) return 'Choose which application owns this new target under Advanced target options.'
      if (this.parsed()?.protocol === 'http:' && this.stage.value !== 'local') return 'For a new plain HTTP target, explicitly choose Local under Advanced target options. Use HTTPS for other stages.'
    }
    return ''
  }

  async ensureTarget(project: string): Promise<void> {
    if (!this.needsTarget() || this.targetSaved()) return
    this.locked.set(true)
    try {
      const base = `/api/v1/projects/${project}`, name = this.parsed()!.hostname, applicationId = this.proposedApplicationId(), id = this.proposedEnvironmentId()
      if (this.needsApplication() && !this.applicationSaved()) {
        const path = `${base}/applications/${applicationId}`
        await this.records.create(path, { id: applicationId, name, description: '' }, 'PUT', path)
        this.applicationSaved.set(true)
      }
      const path = `${base}/environments/${id}`
      await this.records.create(path, { id, name, applicationId, stage: this.stage.value, protected: this.protected.value, disabled: false, endpoints: { api: this.parsed()!.origin } }, 'PUT', path)
      this.targetSaved.set(true)
    } catch (error) { if (!this.applicationSaved() && error instanceof ApiError && error.status < 500) this.locked.set(false); throw error }
  }
}
