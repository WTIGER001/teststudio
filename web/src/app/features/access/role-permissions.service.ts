import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { RolePoliciesService } from './role-policies.service'
import { AccessClientService } from '../../core/access-client.service'
import type { PermissionChoice } from '../../core/generated/access.types'
import type { RoleRuleForm } from './role-policy.forms'

@Injectable()
export class RolePermissionCatalog {
  private readonly client = inject(AccessClientService)
  private readonly cache = new Map<string, Promise<PermissionChoice[]>>()
  reset(): void { this.cache.clear() }
  read(project: string, object: string, refresh = false): Promise<PermissionChoice[]> {
    const key = JSON.stringify([project, object]); if (refresh) this.cache.delete(key)
    let response = this.cache.get(key)
    if (!response) { response = this.client.call('permissionCatalog', { query: { project, object } }).then(page => page.items); this.cache.set(key, response); void response.catch(() => { if (this.cache.get(key) === response) this.cache.delete(key) }) }
    return response
  }
}

@Injectable()
export class RoleRulePermissions {
  private readonly catalog = inject(RolePermissionCatalog)
  private readonly roles = inject(RolePoliciesService)
  readonly choices = signal<PermissionChoice[]>([])
  readonly loading = signal(false)
  readonly error = signal('')
  readonly loadedObject = signal('')
  private generation = 0
  private project = ''
  private row: RoleRuleForm | null = null
  constructor() { inject(DestroyRef).onDestroy(() => { this.generation++; this.choices.set([]) }) }
  select(project: string, row: RoleRuleForm): void { this.generation++; this.project = project; this.row = row; this.choices.set([]); void this.load() }
  async load(refresh = false): Promise<void> {
    const project = this.project, row = this.row; if (!row) return
    const generation = ++this.generation, object = row.controls.object.value.trim(); this.loading.set(true); this.error.set('')
    try { const choices = await this.catalog.read(project, object, refresh); if (generation === this.generation) { this.choices.set(choices); this.loadedObject.set(object) } }
    catch (error) { if (generation === this.generation) { this.choices.set([]); this.error.set(error instanceof Error ? error.message : 'Rule permissions could not be loaded.'); await this.roles.mutationError(error) } }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
}
