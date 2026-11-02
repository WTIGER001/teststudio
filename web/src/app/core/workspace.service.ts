import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { NavigationEnd, Router } from '@angular/router'
import { ApiService, type Page, type RecordValue } from './api.service'
import { projectFromURL } from './theme.utils'
import { SessionService } from './session.service'

export interface Project { id: string; name: string; description: string; archived: boolean }

@Injectable({ providedIn: 'root' })
export class WorkspaceService {
  private readonly api = inject(ApiService)
  private readonly session = inject(SessionService)
  private readonly router = inject(Router)
  private readonly destroy = inject(DestroyRef)
  readonly projects = signal<RecordValue<Project>[]>([])
  readonly projectId = signal(projectFromURL(this.router.url))
  readonly project = computed(() => this.projects().find(p => p.id === this.projectId())?.data ?? null)
  readonly capabilities = signal<Record<string, boolean>>({})
  private capabilityGeneration = 0

  constructor() {
    this.router.events.pipe(takeUntilDestroyed(this.destroy)).subscribe(event => {
      if (event instanceof NavigationEnd) {
        this.projectId.set(projectFromURL(event.urlAfterRedirects))
        if (this.session.signedIn()) void this.loadCapabilities()
      }
    })
  }

  async load(): Promise<void> {
    this.projects.set((await this.api.request<Page<RecordValue<Project>>>('/api/v1/projects')).items)
    await this.loadCapabilities()
  }

  async loadCapabilities(): Promise<void> {
    const generation = ++this.capabilityGeneration
    const project = this.projectId()
    try {
      const capabilities = await this.api.request<Record<string, boolean>>(`/api/v1/me/capabilities?project=${encodeURIComponent(project)}`)
      if (generation === this.capabilityGeneration) this.capabilities.set(capabilities)
    } catch { if (generation === this.capabilityGeneration) this.capabilities.set({}) }
  }

  selectProject(id: string): void { void this.router.navigate(id ? ['/projects', id, 'catalog'] : ['/projects']) }
  can(permission: string): boolean { return this.capabilities()[`teststudio.${permission}`] === true }
}
