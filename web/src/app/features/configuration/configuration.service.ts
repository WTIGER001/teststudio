import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { ApiService, type Page, type RecordValue } from '../../core/api.service'
import type { SuiteLifecycle } from '../../core/generated/access.types'
import { replaceConfiguredSuite } from './configuration.utils'
import { WorkspaceService } from '../../core/workspace.service'

export interface Application { id: string; name: string; description: string }
export interface Suite { id: string; name: string; description: string; applications: string[]; requiresTarget: boolean; activeReleaseId?: string; archived: boolean }
export interface Environment { id: string; name: string; applicationId: string; stage: string; protected: boolean; disabled: boolean; endpoints: Record<string, string> }
type Kind = 'applications' | 'suites' | 'environments'

@Injectable()
export class ConfigurationService {
  private readonly api = inject(ApiService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  readonly workspace = inject(WorkspaceService)
  readonly projectId = signal('')
  readonly kind = signal<Kind>('applications')
  readonly canCreate = computed(() => this.workspace.can(this.kind() === 'applications' ? 'projects.manage' : this.kind() === 'suites' ? 'suites.manage' : 'environments.manage'))
  readonly applications = signal<RecordValue<Application>[]>([])
  readonly suites = signal<RecordValue<Suite>[]>([])
  readonly environments = signal<RecordValue<Environment>[]>([])
  readonly loading = signal(false)
  readonly saving = signal(false)
  readonly error = signal('')
  readonly success = signal('')
  readonly form = new FormGroup({
    id: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.pattern(/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/)] }),
    name: new FormControl('', { nonNullable: true, validators: Validators.required }),
    application: new FormControl('', { nonNullable: true }),
    stage: new FormControl('development', { nonNullable: true }),
    protected: new FormControl(false, { nonNullable: true }),
    endpoint: new FormControl('api', { nonNullable: true }),
    url: new FormControl('', { nonNullable: true }),
    requiresTarget: new FormControl(true, { nonNullable: true })
  })
  private generation = 0

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => { this.projectId.set(params.get('projectId') ?? ''); void this.load() })
  }

  async load(): Promise<void> {
    const generation = ++this.generation
    this.loading.set(true)
    this.error.set('')
    try {
      const base = `/api/v1/projects/${this.projectId()}`
      const [applications, suites, environments] = await Promise.all([
        this.api.request<Page<RecordValue<Application>>>(`${base}/applications`),
        this.api.request<Page<RecordValue<Suite>>>(`${base}/suites`),
        this.api.request<Page<RecordValue<Environment>>>(`${base}/environments`)
      ])
      if (generation !== this.generation) return
      this.applications.set(applications.items); this.suites.set(suites.items); this.environments.set(environments.items)
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Project settings could not be loaded.') }
    finally { if (generation === this.generation) this.loading.set(false) }
  }

  applySuite(state: SuiteLifecycle): void { this.suites.update(rows => replaceConfiguredSuite(rows, state)) }

  select(kind: Kind): void { this.kind.set(kind); this.success.set(''); this.error.set(''); this.form.controls.id.setValue(''); this.form.controls.name.setValue('') }

  async create(): Promise<void> {
    if (this.form.invalid || this.saving()) return
    this.saving.set(true); this.error.set(''); this.success.set('')
    try {
      const value = this.form.getRawValue()
      let data: Application | Suite | Environment = { id: value.id, name: value.name, description: '' }
      if (this.kind() === 'suites') data = { ...data, applications: value.application ? [value.application] : [], requiresTarget: value.requiresTarget, archived: false }
      if (this.kind() === 'environments') data = { id: value.id, name: value.name, applicationId: value.application, stage: value.stage, protected: value.protected, disabled: false, endpoints: { [value.endpoint]: value.url } }
      await this.api.request(`/api/v1/projects/${this.projectId()}/${this.kind()}/${value.id}`, 'PUT', data, 0)
      this.success.set(`${value.name} was created.`)
      this.form.controls.id.setValue(''); this.form.controls.name.setValue('')
      await this.load()
    } catch (error) { this.error.set(error instanceof Error ? error.message : 'Creation failed.') }
    finally { this.saving.set(false) }
  }
}
