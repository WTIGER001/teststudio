import { ChangeDetectionStrategy, Component, effect, inject, untracked } from '@angular/core'
import { ActivatedRoute, RouterLink } from '@angular/router'
import { SourcesService } from './sources.service'
import { PreparationComponent } from './preparation.component'

@Component({
  standalone: true, imports: [RouterLink, PreparationComponent], providers: [SourcesService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">{{ sources.workspace.project()?.name || 'Project' }}</span><h1>Recording workspace</h1><p class="muted">Record in this tab and choose the website's browser size. Closing this tab preserves captured steps until the session ends.</p></div><a class="button secondary" [routerLink]="sources.workspace.path('tests', 'new', 'playwright')" [queryParams]="{ suite: suite, recording: recording }">Return to test setup</a></div>
    @if (sources.error()) { <p class="notice error" role="alert">{{ sources.error() }}</p> }
    @if (sources.loading()) { <p role="status">Loading recording workspace…</p> }
    @if (sources.form.controls.suite.value) { <p class="muted">Suite: {{ suite }}</p><ts-source-preparation mode="playwright" [workspace]="true" /> }
    @else if (!sources.loading() && !sources.error()) { <p class="notice error" role="alert">This suite is unavailable. Return to test setup and choose an active suite.</p> }
  `,
  styles: [`:host { display: block; width: 100%; }`]
})
export class RecordingWorkspacePage {
  readonly sources = inject(SourcesService)
  private readonly route = inject(ActivatedRoute)
  readonly suite = this.route.snapshot.paramMap.get('suiteId') ?? ''
  readonly recording = this.route.snapshot.paramMap.get('recordingId') ?? ''
  canLeave(): boolean { return !(this.sources.draftDirty() || this.sources.creationDirty()) || window.confirm('This test has unsaved changes. Leave without saving?') }
  constructor() {
    effect(() => {
      const suites = this.sources.suites()
      if (suites.some(row => row.id === this.suite)) untracked(() => { if (!this.sources.form.controls.suite.value) this.sources.form.controls.suite.setValue(this.suite) })
    })
  }
}
