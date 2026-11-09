import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { combineLatest } from 'rxjs'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError, ApiService, type Page, type RecordValue } from '../../core/api.service'
import type { ReleaseRegistry, SourceCatalogRelease } from '../../core/generated/access.types'
import type { Suite } from '../configuration/configuration.service'
import { compareReleases, comparisonBase, filterReleases, githubCommit } from './release.utils'

@Injectable()
export class ReleasesService {
  private readonly api = inject(ApiService)
  private readonly client = inject(AccessClientService)
  readonly project = signal('')
  readonly suites = signal<RecordValue<Suite>[]>([])
  readonly registry = signal<ReleaseRegistry | null>(null)
  readonly selected = signal<SourceCatalogRelease | null>(null)
  readonly query = signal('')
  readonly form = new FormGroup({ suite: new FormControl('', { nonNullable: true }), base: new FormControl('', { nonNullable: true }) })
  readonly base = signal('')
  readonly busy = signal(false)
  readonly conflict = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly visible = computed(() => filterReleases(this.registry()?.items ?? [], this.query()))
  readonly comparison = computed(() => compareReleases(this.registry()?.items.find(release => release.id === this.base()), this.selected()))
  readonly commit = computed(() => githubCommit(this.selected()))
  private generation = 0
  private requestedRelease = ''

  constructor() {
    const route = inject(ActivatedRoute), destroy = inject(DestroyRef)
    combineLatest([route.paramMap, route.queryParamMap]).pipe(takeUntilDestroyed(destroy)).subscribe(([params, query]) => {
      this.generation++; this.project.set(params.get('projectId') ?? ''); this.suites.set([]); this.registry.set(null); this.selected.set(null); this.error.set(''); this.message.set(''); this.form.reset(); this.requestedRelease = query.get('releaseId') ?? ''; void this.initialize(query.get('suiteId') ?? '')
    })
    this.form.controls.suite.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.selected.set(null); this.registry.set(null); this.message.set(''); void this.refresh() })
    this.form.controls.base.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(value => this.base.set(value))
    destroy.onDestroy(() => { this.generation++ })
  }

  private async initialize(preferred: string): Promise<void> {
    const generation = this.generation
    this.busy.set(true)
    try {
      const suites = await this.api.request<Page<RecordValue<Suite>>>(`/api/v1/projects/${this.project()}/suites`)
      if (generation !== this.generation) return
      this.suites.set(suites.items); this.form.controls.suite.setValue(suites.items.some(suite => suite.id === preferred) ? preferred : suites.items[0]?.id ?? '', { emitEvent: false }); await this.refresh()
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async refresh(): Promise<void> {
    const suite = this.form.controls.suite.value, generation = ++this.generation, selected = this.selected()?.id || this.requestedRelease
    if (!suite) { this.busy.set(false); return }
    this.busy.set(true); this.error.set('')
    try {
      const registry = await this.client.call('readReleaseRegistry', { project: this.project(), suite })
      if (generation !== this.generation) return
      this.registry.set(registry); this.conflict.set(false)
      const release = registry.items.find(release => release.id === selected) ?? registry.items.find(release => release.id === registry.activeReleaseId) ?? registry.items[0]
      this.selected.set(release ?? null); this.form.controls.base.setValue(comparisonBase(registry.items, release?.id ?? '', registry.activeReleaseId ?? '')); this.requestedRelease = ''
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  inspect(release: SourceCatalogRelease): void { this.selected.set(release); this.form.controls.base.setValue(comparisonBase(this.registry()?.items ?? [], release.id, this.registry()?.activeReleaseId ?? '')); this.error.set(''); this.message.set('') }

  async activate(): Promise<void> {
    const release = this.selected(), registry = this.registry(), generation = this.generation
    if (!release || !registry?.canActivate || this.busy() || this.conflict() || registry.activeReleaseId === release.id) return
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      await this.client.call('activateRetainedRelease', { project: this.project(), suite: registry.suiteId, id: release.id, revision: registry.suiteRevision })
      if (generation !== this.generation) return
      this.message.set('Version activated for future runs. Existing runs and pinned schedules retain their versions.'); await this.refresh()
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  private fail(error: unknown): void {
    if (error instanceof ApiError && error.status === 409) { this.conflict.set(true); this.error.set('Suite settings or the active version changed. Refresh retained versions before activating again.'); return }
    this.error.set(error instanceof Error ? error.message : 'Retained versions could not be loaded.')
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) { this.registry.set(null); this.selected.set(null) }
  }
}
