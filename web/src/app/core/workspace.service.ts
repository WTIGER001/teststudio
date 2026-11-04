import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { NavigationEnd, Router } from '@angular/router'
import { ApiService, type Page, type RecordValue } from './api.service'
import { applicationFromURL, scopePath, switchApplicationPath } from './application-scope.utils'
import { projectFromURL } from './theme.utils'
import { SessionService } from './session.service'

export interface WorkspaceApplication { id: string; name: string; description: string }

export interface Project { id: string; name: string; description: string; archived: boolean }

@Injectable({ providedIn: 'root' })
export class WorkspaceService {
  private readonly api = inject(ApiService)
  private readonly session = inject(SessionService)
  private readonly router = inject(Router)
  private readonly destroy = inject(DestroyRef)
  readonly projects = signal<RecordValue<Project>[]>([])
  readonly projectId = signal(projectFromURL(this.router.url))
  readonly applicationId = signal(applicationFromURL(this.router.url))
  readonly applications = signal<RecordValue<WorkspaceApplication>[]>([])
  readonly application = computed(() => this.applications().find(row => row.id === this.applicationId())?.data ?? null)
  readonly applicationError = signal('')
  readonly project = computed(() => this.projects().find(p => p.id === this.projectId())?.data ?? null)
  readonly capabilities = signal<Record<string, boolean>>({})
  readonly installationCapabilities = signal<Record<string, boolean>>({})
  private capabilityGeneration = 0
  private projectGeneration = 0
  private applicationProject = ''

  constructor() {
    this.router.events.pipe(takeUntilDestroyed(this.destroy)).subscribe(event => {
      if (event instanceof NavigationEnd) {
        this.projectId.set(projectFromURL(event.urlAfterRedirects))
        this.applicationId.set(applicationFromURL(event.urlAfterRedirects))
        if (this.session.ready()) void this.loadCapabilities()
      }
    })
  }

  async load(): Promise<void> {
    this.applicationProject = ''
    const generation = ++this.projectGeneration, user = this.session.user()?.id
    const response = await this.api.request<Page<RecordValue<Project>>>('/api/v1/projects')
    if (generation !== this.projectGeneration || user !== this.session.user()?.id || !this.session.ready()) return
    this.projects.set(response.items); await this.loadCapabilities()
  }

  reset(): void { this.projectGeneration++; this.capabilityGeneration++; this.applicationProject = ''; this.projects.set([]); this.applications.set([]); this.applicationError.set(''); this.capabilities.set({}); this.installationCapabilities.set({}) }

  async refreshAccess(): Promise<void> {
    try { await this.load() }
    catch { this.projects.set([]); await this.loadCapabilities() }
  }

  async loadCapabilities(): Promise<void> {
    const generation = ++this.capabilityGeneration
    const project = this.projectId()
    this.capabilities.set({}); this.installationCapabilities.set({})
    try {
      const [capabilities, installation] = await Promise.all([this.readCapabilities(project), project ? this.readCapabilities('') : Promise.resolve(null)])
      if (generation === this.capabilityGeneration) { this.capabilities.set(capabilities); this.installationCapabilities.set(installation ?? capabilities); await this.loadApplications(project, generation) }
    } catch { if (generation === this.capabilityGeneration) { this.capabilities.set({}); this.installationCapabilities.set({}) } }
  }

  readCapabilities(project: string): Promise<Record<string, boolean>> { return this.api.request<Record<string, boolean>>(`/api/v1/me/capabilities?project=${encodeURIComponent(project)}`) }

  private async loadApplications(project: string, generation: number): Promise<void> {
    if (!project || !this.can('projects.read')) { this.applicationProject = ''; this.applications.set([]); this.applicationError.set(''); return }
    if (this.applicationProject === project) return
    this.applications.set([]); this.applicationError.set('')
    try {
      const page = await this.api.request<Page<RecordValue<WorkspaceApplication>>>(`/api/v1/projects/${encodeURIComponent(project)}/applications`)
      if (generation === this.capabilityGeneration) { this.applications.set(page.items); this.applicationProject = project }
    } catch { if (generation === this.capabilityGeneration) this.applicationError.set('Applications could not be loaded. Refresh access to retry.') }
  }

  path(...parts: string[]): string[] { return scopePath(this.projectId(), this.applicationId(), ...parts) }
  async selectApplication(id: string, picker?: HTMLSelectElement): Promise<void> { try { await this.router.navigate(switchApplicationPath(this.projectId(), id, this.router.url)) } finally { if (picker) picker.value = this.applicationId() } }
  async selectProject(id: string, picker?: HTMLSelectElement): Promise<void> { try { await this.router.navigate(id ? ['/projects', id, 'applications'] : ['/projects']) } finally { if (picker) picker.value = this.projectId() } }
  can(permission: string): boolean { return this.capabilities()[`teststudio.${permission}`] === true }
  canInstallation(permission: string): boolean { return this.installationCapabilities()[`teststudio.${permission}`] === true }
}
