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
      const key = `${project}:${kind}:${suites.map(row => row.id).join(',')}`
      untracked(() => {
        if (key === this.initialized) return
        this.initialized = key
        this.native.configure(project, suites); this.native.create()
        const preferred = this.route.snapshot.queryParamMap.get('suite')
        if (preferred && suites.some(row => row.id === preferred)) { this.native.form.controls.suite.setValue(preferred); this.sources.form.controls.suite.setValue(preferred) }
      })
    })
  }

  afterCreate(): void { void this.router.navigate(['/projects', this.sources.project(), 'catalog'], { queryParams: { created: this.native.form.controls.id.value } }) }

  canLeave(): boolean {
    const dirty = this.sources.draftDirty() || this.kind() === 'https' && this.native.dirty()
    return !dirty || window.confirm('This test has unsaved changes. Leave without saving?')
  }
}
