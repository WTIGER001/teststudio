import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { SavedSelectionPreview, SavedSelectionRecord } from '../../core/generated/access.types'
import { CatalogService } from './catalog.service'
import { CatalogSelectionService } from './catalog-selection.service'
import { emptySavedSelectionForm, savedSelectionDefinition, savedSelectionForm, savedSelectionFromItems } from './catalog-saved-selections.utils'

@Injectable()
export class CatalogSavedSelectionsService {
  readonly catalog = inject(CatalogService)
  private readonly selected = inject(CatalogSelectionService)
  private readonly client = inject(AccessClientService)
  readonly items = signal<SavedSelectionRecord[]>([])
  readonly editing = signal<SavedSelectionRecord | null>(null)
  readonly preview = signal<SavedSelectionPreview | null>(null)
  readonly cursor = signal('')
  readonly loading = signal(false)
  readonly busy = signal(false)
  readonly conflict = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly form = new FormGroup({
    suite: new FormControl('', { nonNullable: true, validators: Validators.required }),
    id: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.pattern(/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/)] }),
    name: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.maxLength(200), Validators.pattern(/\S/)] }),
    description: new FormControl('', { nonNullable: true, validators: Validators.maxLength(4096) }),
    kind: new FormControl('explicit', { nonNullable: true }), tests: new FormControl('', { nonNullable: true }),
    query: new FormControl('', { nonNullable: true }), category: new FormControl('', { nonNullable: true }), descendants: new FormControl(false, { nonNullable: true }), tags: new FormControl('', { nonNullable: true }), owner: new FormControl('', { nonNullable: true }), application: new FormControl('', { nonNullable: true }), type: new FormControl('', { nonNullable: true }), runner: new FormControl('', { nonNullable: true }), safety: new FormControl('', { nonNullable: true }), source: new FormControl('', { nonNullable: true })
  })
  private generation = 0
  private loadGeneration = 0

  constructor() {
    const destroy = inject(DestroyRef)
    this.form.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => this.preview.set(null))
    this.form.controls.suite.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(suite => { this.generation++; this.loadGeneration++; this.loading.set(false); this.reset(suite); this.items.set([]); this.cursor.set(''); void this.load() })
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.generation++; this.loadGeneration++; this.reset(''); this.items.set([]); this.cursor.set(''); this.loading.set(false) })
    destroy.onDestroy(() => { this.generation++; this.loadGeneration++ })
  }

  reset(suite = this.form.controls.suite.value): void { this.generation++; this.editing.set(null); this.preview.set(null); this.conflict.set(false); this.error.set(''); this.message.set(''); this.busy.set(false); this.form.reset(emptySavedSelectionForm(suite), { emitEvent: false }) }
  edit(row: SavedSelectionRecord): void { this.generation++; this.busy.set(false); this.apply(row); this.error.set(''); this.message.set('') }
  useSelected(): void {
    try { const value = savedSelectionFromItems(this.selected.items()); this.form.controls.suite.setValue(value.suite); this.form.patchValue({ kind: 'explicit', tests: value.tests }); this.form.markAsDirty() }
    catch (error) { this.fail(error) }
  }

  async load(more = false): Promise<void> {
    const suite = this.form.controls.suite.value
    if (!suite || more && !this.cursor()) return
    const generation = ++this.loadGeneration
    this.loading.set(true)
    try {
      const page = await this.client.call('listSavedSelections', { ...this.scope(), query: { cursor: more ? this.cursor() : undefined, limit: 30, archived: true } })
      if (generation === this.loadGeneration && suite === this.form.controls.suite.value) { this.items.set(more ? [...this.items(), ...page.items] : page.items); this.cursor.set(page.cursor ?? '') }
    } catch (error) { if (generation === this.loadGeneration) this.fail(error) }
    finally { if (generation === this.loadGeneration) this.loading.set(false) }
  }

  async reload(): Promise<void> {
    const row = this.editing()
    if (!row || this.busy()) return
    const generation = this.generation
    this.busy.set(true); this.error.set('')
    try { const current = await this.client.call('readSavedSelection', { ...this.scope(), id: row.id }); if (generation === this.generation) this.apply(current) }
    catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async save(): Promise<void> {
    if (this.form.invalid || this.busy() || this.conflict() || this.editing()?.data.archived) return
    const row = this.editing(), generation = this.generation
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      const body = savedSelectionDefinition(this.form.getRawValue())
      const updated = row ? await this.client.call('updateSavedSelection', { ...this.scope(), id: row.id, revision: row.revision, body }) : await this.client.call('createSavedSelection', { ...this.scope(), body })
      if (generation !== this.generation) return
      this.apply(updated); this.message.set('Saved selection stored. Test execution starts separately after a resolved preview.'); await this.load()
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async changeState(action: 'archive' | 'restore'): Promise<void> {
    const row = this.editing(), generation = this.generation
    if (!row || this.busy() || this.conflict() || this.form.dirty) return
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      const updated = await this.client.call('changeSavedSelectionState', { ...this.scope(), id: row.id, action, revision: row.revision })
      if (generation !== this.generation) return
      this.apply(updated); this.message.set(action === 'archive' ? 'Saved selection archived. Existing runs keep their frozen selection.' : 'Saved selection restored. Preview its current tests before starting a run.'); await this.load()
    } catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async resolve(): Promise<void> {
    const row = this.editing(), generation = this.generation
    if (!row || row.data.archived || this.busy() || this.conflict() || this.form.dirty) return
    this.busy.set(true); this.error.set(''); this.preview.set(null)
    try { const preview = await this.client.call('previewSavedSelection', { ...this.scope(), id: row.id, body: {} }); if (generation === this.generation) this.preview.set(preview) }
    catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  private scope() { return { project: this.catalog.projectId(), suite: this.form.controls.suite.value } }
  private apply(row: SavedSelectionRecord): void { this.editing.set(row); this.conflict.set(false); this.preview.set(null); this.form.reset(savedSelectionForm(this.form.controls.suite.value, row.data), { emitEvent: false }) }
  private fail(error: unknown): void { this.error.set(error instanceof Error ? error.message : 'Saved selection operation failed.'); if (error instanceof ApiError && error.status === 409) { this.conflict.set(true); this.error.set('The saved selection changed. Reload saved selection and review its current revision before retrying.') } }
}
