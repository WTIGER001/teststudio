import { RecordCreationService } from '../../core/record-creation.service'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { TimingPolicyEditorService } from '../timing/timing-policy-editor.service'
import { SourcesService } from '../sources/sources.service'
import { PreparationComponent } from '../sources/preparation.component'
import { SourcePreviewComponent } from '../sources/source-preview.component'
import { HTTPCheckEditorService } from './http-check-editor.service'
import { CatalogTestEditorService } from './catalog-test-editor.service'
import { HTTPSWizardComponent } from './https-wizard.component'
import { CreateTestService } from './create-test.service'

@Component({
  standalone: true, imports: [RouterLink, ReactiveFormsModule, HTTPSWizardComponent, PreparationComponent, SourcePreviewComponent],
  providers: [RecordCreationService, SourcesService, TimingPolicyEditorService, HTTPCheckEditorService, CatalogTestEditorService, CreateTestService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">{{ service.sources.workspace.project()?.name || 'Project' }}</span><h1>Create test</h1><p class="muted">Choose what you want to test, then follow its creation workflow.</p></div><a class="button secondary" [routerLink]="service.sources.workspace.path('catalog')">Back to catalog</a></div>
    @if (!service.kind()) {
      <section aria-labelledby="test-type-title"><h2 id="test-type-title">What type of test?</h2><div class="type-choices">@for (choice of service.choices; track choice.id) { <a class="panel type-choice" [routerLink]="service.sources.workspace.path('tests', 'new', choice.id)" queryParamsHandling="preserve" [attr.aria-label]="choice.title"><span class="type-symbol" aria-hidden="true">{{ choice.symbol }}</span><h3>{{ choice.title }}</h3><p class="muted">{{ choice.description }}</p><span class="choose-type">Choose {{ choice.title }} →</span></a> }</div></section>
      <p class="muted">Already have tests? <a [routerLink]="['/projects', service.sources.project(), 'sources']">Upload a source bundle or connect a repository.</a></p>
    } @else if (service.kind() === 'unknown') {
      <p class="notice error" role="alert">This test type is unavailable. Choose a supported type.</p><a class="button secondary" [routerLink]="service.sources.workspace.path('tests', 'new')">Choose a test type</a>
    } @else {
      <p class="type-return"><a [routerLink]="service.sources.workspace.path('tests', 'new')" queryParamsHandling="preserve">← Choose another test type</a></p>
      @if (service.sources.error()) { <p class="notice error" role="alert">{{ service.sources.error() }}</p> }
      @if (service.sources.loading()) { <p role="status">Loading suites…</p> }
      @else if (!service.sources.suites().length) { <section class="panel"><h2>Create a suite first</h2><p class="muted">Tests belong to a suite, which holds their releases and run settings.</p><a class="button primary" [routerLink]="service.sources.workspace.path('suites')">Create or manage suites</a></section> }
      @else if (service.kind() === 'https') {
        @if (service.sources.workspace.can('catalog.create')) { <section class="panel" aria-labelledby="create-probe-title"><h2 id="create-probe-title">Configure an HTTPS probe</h2><p class="muted">Name the test, configure its request, then review before saving.</p><ts-https-wizard /></section> }
        @else { <p class="notice">You need catalog creation permission to add an HTTPS probe.</p> }
      } @else {
        <section class="panel" aria-labelledby="creation-suite-title"><h2 id="creation-suite-title">Destination suite</h2><fieldset [disabled]="service.sources.draftDirty() || service.sources.creationDirty()" [formGroup]="service.sources.form"><div class="field"><label for="creation-suite">Suite for this test</label><select id="creation-suite" formControlName="suite"><option value="">Choose a suite</option>@for (suite of service.sources.suites(); track suite.id) { <option [value]="suite.id">{{ suite.data.name }}</option> }</select></div></fieldset><a class="small" [routerLink]="service.sources.workspace.path('suites')">Create or manage suites</a></section>
        @if (service.sources.form.controls.suite.value) {
          @if (service.kind() === 'playwright') { <ts-source-preparation mode="playwright" /> }
          @else if (service.kind() === 'api') { <ts-source-preparation mode="api" /> }
          @else { <section class="panel"><h2>Import a Go suite</h2><p class="muted">Upload its complete source ZIP below. For repository synchronization, <a [routerLink]="['/projects', service.sources.project(), 'sources']">manage repository sources</a>.</p></section><ts-source-preparation mode="go" /> }
          <ts-source-preview />
        }
      }
    }
  `,
  styles: [`h2 { margin-bottom: .75rem; } .type-choices { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 16rem), 1fr)); gap: 1.25rem; margin: 1rem 0; } .type-choice { color: var(--text); display: flex; flex-direction: column; gap: .75rem; margin: 0; transition: border-color 140ms; } .type-choice:hover { border-color: var(--accent); text-decoration: none; } .type-symbol { font-size: 30px; color: var(--accent); } .type-choice h3 { font-size: 18px; } .choose-type { color: var(--accent); margin-top: auto; padding-top: 1rem; font-weight: 650; font-size: 12px; } .type-return { margin-bottom: 1.25rem; } fieldset { border: 0; padding: 0; margin: 0 0 1rem; min-width: 0; }`]
})
export class CreateTestPage {
  readonly service = inject(CreateTestService)
  canLeave(): boolean { return this.service.canLeave() }
}
