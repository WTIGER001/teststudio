import { effect, inject, Injectable, untracked } from '@angular/core'
import { toSignal } from '@angular/core/rxjs-interop'
import { ActivatedRoute, Router } from '@angular/router'
import { map } from 'rxjs'
import { SourcesService } from '../sources/sources.service'
import { CatalogTestEditorService } from './catalog-test-editor.service'
import { creationChoices, creationKind } from './create-test.utils'

@Injectable()
export class CreateTestService {
  readonly sources = inject(SourcesService)
  readonly native = inject(CatalogTestEditorService)
  private readonly router = inject(Router)
  private readonly route = inject(ActivatedRoute)
  readonly kind = toSignal(this.route.paramMap.pipe(map(params => creationKind(params.get('kind')))), { initialValue: creationKind(this.route.snapshot.paramMap.get('kind')) })
  readonly choices = creationChoices
  private initialized = ''

  constructor() {
    effect(() => {
      const project = this.sources.project(), suites = this.sources.suites(), kind = this.kind()
      const key = `${project}:${this.sources.applicationId()}:${kind}`
      untracked(() => {
        this.native.configure(project, suites)
        const preferred = this.route.snapshot.queryParamMap.get('suite')
        if (preferred && suites.some(row => row.id === preferred) && !this.sources.form.controls.suite.value) this.sources.form.controls.suite.setValue(preferred)
        if (key === this.initialized) { if (!this.native.form.controls.suite.value && suites.length) this.native.form.controls.suite.setValue(suites[0].id); return }
        this.initialized = key
        this.native.create()
        if (preferred && suites.some(row => row.id === preferred)) this.native.form.controls.suite.setValue(preferred)
      })
    })
  }

  afterCreate(): void { void this.router.navigate(this.sources.workspace.path(this.route.snapshot.queryParamMap.get('from') === 'probes' ? 'probes' : 'catalog'), { queryParams: { created: this.native.form.controls.id.value } }) }

  canLeave(): boolean {
    const dirty = this.sources.draftDirty() || this.sources.creationDirty() || this.kind() === 'https' && this.native.dirty()
    return !dirty || window.confirm('This test has unsaved changes. Leave without saving?')
  }
}
