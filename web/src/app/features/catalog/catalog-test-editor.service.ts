import { computed, inject, Injectable, signal } from '@angular/core'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { ApiService, type RecordValue } from '../../core/api.service'
import type { Suite } from '../configuration/configuration.service'
import { TimingPolicyEditorService } from '../timing/timing-policy-editor.service'
import { HTTPCheckEditorService } from './http-check-editor.service'
import type { CatalogItem, TestDefinition } from './catalog.models'

@Injectable()
export class CatalogTestEditorService {
  private readonly api = inject(ApiService)
  readonly timingEditor = inject(TimingPolicyEditorService)
  readonly checkEditor = inject(HTTPCheckEditorService)
  readonly suites = signal<RecordValue<Suite>[]>([])
  readonly editorOpen = signal(false)
  readonly editing = signal<CatalogItem | null>(null)
  readonly editingHTTP = computed(() => !this.editing() || this.editing()?.data.runner === 'http')
  readonly saving = signal(false)
  readonly error = signal('')
  private project = ''
  readonly form = new FormGroup({
    suite: new FormControl('', { nonNullable: true, validators: Validators.required }),
    id: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.pattern(/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/)] }),
    title: new FormControl('', { nonNullable: true, validators: Validators.required }),
    category: new FormControl('Availability', { nonNullable: true, validators: Validators.required }),
    timeout: new FormControl(10, { nonNullable: true, validators: [Validators.required, Validators.min(1), Validators.max(3600), Validators.pattern(/^\d+$/)] }),
    safety: new FormControl('needs-review', { nonNullable: true }),
    rationale: new FormControl('', { nonNullable: true, validators: Validators.required })
  })

  configure(project: string, suites: RecordValue<Suite>[]): void { this.project = project; this.suites.set(suites) }
  dirty(): boolean { return this.form.dirty || this.checkEditor.form.dirty || this.timingEditor.form.dirty || this.timingEditor.override.dirty }
  create(): void {
    this.editing.set(null)
    this.timingEditor.reset()
    this.checkEditor.reset()
    this.form.reset({ suite: this.suites()[0]?.id ?? '', id: '', title: '', category: 'Availability', timeout: 10, safety: 'needs-review', rationale: '' })
    this.form.controls.id.enable(); this.form.controls.suite.enable()
    this.editorOpen.set(true); this.error.set('')
  }

  edit(item: CatalogItem): void {
    this.editing.set(item)
    this.timingEditor.reset(item.data.timing)
    this.checkEditor.reset(item.data.check)
    this.form.reset({ suite: item.suiteId, id: item.id, title: item.data.title, category: item.data.category, timeout: item.data.timeoutSeconds, safety: item.data.safety, rationale: item.data.rationale })
    this.form.controls.id.disable(); this.form.controls.suite.disable()
    this.editorOpen.set(true); this.error.set('')
  }

  async save(): Promise<boolean> {
    this.form.markAllAsTouched()
    this.checkEditor.form.markAllAsTouched()
    if (this.form.invalid || !this.timingEditor.valid() || (this.editingHTTP() && !this.checkEditor.valid()) || this.saving()) return false
    this.saving.set(true); this.error.set('')
    try {
      const value = this.form.getRawValue()
      const existing = this.editing()
      const test: TestDefinition = {
        ...(existing?.data ?? { description: '', tags: [], dependencies: [], requiredBindings: [], type: 'https', runner: 'http', state: 'enabled', timeoutSeconds: 10 }),
        id: value.id, title: value.title, category: value.category, timeoutSeconds: value.timeout, timing: this.timingEditor.policy(), safety: value.safety, rationale: value.rationale
      }
      if (this.editingHTTP()) test.check = { ...test.check, ...this.checkEditor.check()! }
      const base = `/api/v1/projects/${this.project}/suites/${value.suite}/tests`
      await this.api.request(existing ? `${base}/${value.id}` : base, existing ? 'PUT' : 'POST', test, existing?.revision)
      this.editorOpen.set(false); this.form.markAsPristine(); this.checkEditor.form.markAsPristine(); this.timingEditor.form.markAsPristine(); this.timingEditor.override.markAsPristine()
      return true
    } catch (error) { this.error.set(error instanceof Error ? error.message : 'The test could not be saved.'); return false }
    finally { this.saving.set(false) }
  }

}
