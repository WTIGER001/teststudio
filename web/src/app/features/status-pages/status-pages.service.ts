import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import type { StatusPageRecord } from '../../core/generated/access.types'
import { WorkspaceService } from '../../core/workspace.service'

@Injectable()
export class StatusPagesService {
  private readonly client = inject(AccessClientService); private readonly workspace = inject(WorkspaceService); private readonly destroy = inject(DestroyRef)
  readonly project = signal(''); readonly pages = signal<StatusPageRecord[]>([]); readonly cursor = signal(''); readonly busy = signal(false); readonly error = signal('')
  readonly canManage = computed(() => this.workspace.projectCapabilities()['teststudio.statuspages.manage'] === true)
  private generation = 0
  constructor() {
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => { this.project.set(params.get('projectId') || ''); void this.load() })
    this.destroy.onDestroy(() => { this.generation++ })
  }
  async load(more = false): Promise<void> {
    const generation = ++this.generation; this.busy.set(true); this.error.set(''); if (!more) this.pages.set([])
    try { const page = await this.client.call('listStatusPages', { project: this.project(), query: { cursor: more ? this.cursor() : undefined } }); if (generation === this.generation) { this.pages.update(rows => more ? [...rows, ...page.items] : page.items); this.cursor.set(page.cursor || '') } }
    catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Status pages could not be loaded.') }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
}
