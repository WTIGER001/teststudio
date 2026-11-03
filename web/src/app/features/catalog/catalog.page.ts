import { TimingPolicyEditorComponent } from '../timing/timing-policy-editor.component'
import { TimingPolicyEditorService } from '../timing/timing-policy-editor.service'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { CatalogService } from './catalog.service'
import { CatalogEntryService } from './catalog-entry.service'
import { CatalogEntryComponent } from './catalog-entry.component'
import { CatalogPlanService } from './catalog-plan.service'
import { CatalogPlanComponent } from './catalog-plan.component'
import { CatalogSelectionService } from './catalog-selection.service'
import { CatalogBulkComponent } from './catalog-bulk.component'
import { CatalogBulkMetadataService } from './catalog-bulk-metadata.service'
import { CatalogTransferComponent } from './catalog-transfer.component'
import { HTTPCheckEditorComponent } from './http-check-editor.component'
import { HTTPCheckEditorService } from './http-check-editor.service'

@Component({
  standalone: true,
  imports: [FormsModule, ReactiveFormsModule, RouterLink, CatalogEntryComponent, CatalogPlanComponent, CatalogBulkComponent, CatalogTransferComponent, TimingPolicyEditorComponent, HTTPCheckEditorComponent],
  providers: [HTTPCheckEditorService, TimingPolicyEditorService, CatalogService, CatalogEntryService, CatalogPlanService, CatalogSelectionService, CatalogBulkMetadataService],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">{{ service.workspace.project()?.name || 'Project' }}</span><h1>Test catalog</h1><p class="muted">Know what you test. Keep every release accountable.</p></div>
      @if (service.workspace.can('catalog.create')) { <button class="button primary" type="button" (click)="service.create()" [disabled]="service.suites().length === 0">+ Create test</button> }
    </div>
    <section class="metrics" aria-label="Catalog summary">
      <article class="metric"><span class="eyebrow">Catalog entries</span><strong>{{ service.counts().total }}</strong><span class="small muted">Across this project</span></article>
      <article class="metric"><span class="eyebrow">Enabled</span><strong>{{ service.counts().enabled }}</strong><span class="small muted">Available for selection</span></article>
      <article class="metric"><span class="eyebrow">Labeled safe</span><strong>{{ service.counts().safe }}</strong><span class="small muted">Declared operating conditions apply</span></article>
      <article class="metric"><span class="eyebrow">Suites with tests</span><strong>{{ service.counts().suites }}</strong><span class="small muted">Independent releases</span></article>
    </section>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.success()) { <p class="notice success" role="status">{{ service.success() }}</p> }
    <ts-catalog-entry />
    <ts-catalog-plan />
    @if (service.editorOpen()) {
      <section class="panel editor-panel" aria-labelledby="test-editor-title">
        <div class="section-top"><h2 id="test-editor-title">{{ service.editing() ? 'Edit test' : 'Create HTTPS check' }}</h2><button class="button subtle" type="button" (click)="service.editorOpen.set(false)">Close editor</button></div>
        @if (service.editing()?.data?.definitionSourceId) { <p class="notice">The registered source owns the executable definition. Edit its metadata here and use Sources to update its code or selectors.</p> }
        <form [formGroup]="service.form" (ngSubmit)="service.save()">
          <div class="form-grid">
            <div class="field"><label for="test-suite">Suite</label><select id="test-suite" formControlName="suite" required>@for (suite of service.suites(); track suite.id) { <option [value]="suite.id">{{ suite.data.name }}</option> }</select></div>
            <div class="field"><label for="test-id">Permanent identifier</label><input id="test-id" formControlName="id" required pattern="[A-Za-z0-9](?:[A-Za-z0-9._]|-){0,127}" placeholder="HEALTH-001"></div>
            <div class="field"><label for="test-title">Title</label><input id="test-title" formControlName="title" required></div>
            <div class="field"><label for="test-category">Category</label><input id="test-category" formControlName="category" required></div>
            <div class="field"><label for="test-timeout">Catalog timeout ceiling (seconds)</label><input id="test-timeout" type="number" formControlName="timeout" min="1" max="3600" step="1" required><small>Adaptive timeouts cannot exceed this ceiling.</small></div>
            <div class="field"><label for="test-safety">Production-safety label</label><select id="test-safety" formControlName="safety"><option value="needs-review">Needs review</option><option value="safe">Safe</option><option value="unsafe">Unsafe</option></select></div>
            <div class="field"><label for="test-rationale">Safety rationale</label><textarea id="test-rationale" formControlName="rationale" rows="2" required placeholder="Describe requests, side effects, and required conditions."></textarea></div>
          </div>
          @if (service.editingHTTP()) { <ts-http-check-editor /> }
          <ts-timing-policy-editor />
          <button class="button primary" type="submit" [disabled]="service.saving() || service.form.invalid || !service.timingEditor.valid() || (service.editingHTTP() && !service.checkEditor.valid())">{{ service.saving() ? 'Saving…' : 'Save draft changes' }}</button>
        </form>
      </section>
    }
    <section class="panel catalog-panel" aria-labelledby="catalog-title">
      <div class="section-top"><h2 id="catalog-title">All tests <span class="count-label">{{ service.visible().length }}</span></h2><span class="small muted">Definitions and history use stable identifiers.</span></div>
      <div class="toolbar">
        <div class="search-field"><label class="sr-only" for="catalog-search">Search tests</label><span aria-hidden="true">⌕</span><input id="catalog-search" type="search" placeholder="Search by name, identifier, category, or tag…" [ngModel]="service.query()" (ngModelChange)="service.query.set($event)"></div>
        <label class="sr-only" for="catalog-suite-filter">Filter by suite</label><select id="catalog-suite-filter" [ngModel]="service.suiteFilter()" (ngModelChange)="service.suiteFilter.set($event)"><option value="">All suites</option>@for (suite of service.suites(); track suite.id) { <option [value]="suite.id">{{ suite.data.name }}</option> }</select>
        <label class="sr-only" for="catalog-state">Lifecycle state</label><select id="catalog-state" [ngModel]="service.state()" (ngModelChange)="service.state.set($event)"><option value="active">Active catalog</option><option value="enabled">Enabled</option><option value="disabled">Disabled</option><option value="archived">Archived</option><option value="all">All states</option></select>
        <label class="sr-only" for="catalog-type">Filter by type</label><select id="catalog-type" [ngModel]="service.typeFilter()" (ngModelChange)="service.typeFilter.set($event)"><option value="">All types</option>@for (type of service.types(); track type.value) { <option [value]="type.value">{{ type.label }}</option> }</select>
      </div>
      <ts-catalog-bulk />
      @if (service.loading()) { <p class="loading-state" role="status">Loading the catalog…</p> }
      @else if (service.visible().length) {
        <div class="table-scroll"><table><caption class="sr-only">Tests in this project</caption><thead><tr>@if (selection.canSelect()) { <th scope="col"><span class="sr-only">Select</span></th> }<th scope="col">Test</th><th scope="col">Suite / category</th><th scope="col">Type</th><th scope="col">Safety</th><th scope="col">State</th><th scope="col">Actions</th></tr></thead><tbody>
          @for (item of service.visible(); track item.suiteId + '/' + item.id) {
            <tr>@if (selection.canSelect()) { <td><input class="table-checkbox" type="checkbox" [checked]="selection.contains(item)" (change)="selection.toggle(item)" [attr.aria-label]="'Select ' + item.id"></td> }<td><strong>{{ item.data.title }}</strong><code class="table-subtitle">{{ item.id }} · revision {{ item.revision }}</code></td><td>{{ item.suiteName }}<span class="table-subtitle muted">{{ item.data.category }}</span></td><td><span class="badge">{{ item.data.type }}</span></td><td><span class="badge" [class.success]="item.data.safety === 'safe'" [class.warning]="item.data.safety === 'needs-review'">{{ item.data.safety }}</span></td><td><span class="state-label"><span class="status-dot" [class.inactive]="item.data.state !== 'enabled'" aria-hidden="true"></span>{{ item.data.state }}</span></td>
              <td><div class="row-actions wrap-actions"><button class="button small subtle" type="button" (click)="entry.open(item, 'history')">History</button>@if (service.workspace.can('catalog.create')) { <button class="button small subtle" type="button" (click)="entry.open(item, 'duplicate')">Duplicate</button> }@if (service.workspace.can('catalog.update')) { <button class="button small subtle" type="button" (click)="service.edit(item)">Edit</button> }@if (service.workspace.can('catalog.archive') && item.data.state !== 'archived') { <button class="button small subtle" type="button" [disabled]="service.saving()" (click)="service.action(item, 'archive')">Archive</button> }@if (service.workspace.can('catalog.restore') && item.data.state === 'archived') { <button class="button small subtle" type="button" [disabled]="service.saving()" (click)="service.action(item, 'restore')">Restore</button> }</div></td></tr>
          }
        </tbody></table></div>
      } @else {
        <div class="empty-state"><span class="empty-icon" aria-hidden="true">✓</span><h3>{{ service.items().length ? 'No matching tests.' : 'Your next good answer starts here.' }}</h3><p>{{ service.items().length ? 'Try a different search or lifecycle filter.' : 'Create a suite, then add checks or publish a versioned test catalog.' }}</p>@if (!service.suites().length) { <a class="button secondary" [routerLink]="['/projects', service.projectId(), 'configuration']">Set up a suite</a> }</div>
      }
    </section>
    @if (service.suites().length && (service.workspace.can('catalog.import') || service.workspace.can('catalog.export'))) { <ts-catalog-transfer /> }
    @if (service.suites().length) { <p><a class="button secondary" [routerLink]="['/projects', service.projectId(), 'releases']">Browse retained releases</a></p> }
    @if (service.suites().length && service.workspace.can('catalog.update')) {
    <section class="panel release-panel" aria-labelledby="release-title"><div><h2 id="release-title">Publish a release</h2><p class="small muted">Validate draft definitions and freeze an executable catalog.</p></div>
        <div class="inline-form"><label class="sr-only" for="release-suite">Suite to publish</label><select id="release-suite" [ngModel]="service.releaseSuite()" (ngModelChange)="service.releaseSuite.set($event); service.draft.set(null)">@for (suite of service.suites(); track suite.id) { <option [value]="suite.id">{{ suite.data.name }}</option> }</select><button class="button secondary" type="button" (click)="service.prepareRelease()" [disabled]="service.saving()">Validate and preview</button></div>
      </section>
    }
    @if (service.draft(); as draft) {
      <section class="panel" aria-labelledby="release-preview-title"><h2 id="release-preview-title">Release preview</h2><p>{{ draft.data.tests.length }} test definitions in this immutable snapshot.</p>
        @for (issue of service.issues(); track $index) { <p class="notice warning">{{ issue.testId }} · {{ issue.field }}: {{ issue.message }}</p> }
        @if (!service.issues().length) { <p class="notice success">Validation passed. This does not indicate that the tests have run or passed.</p>@if (service.workspace.can('catalog.publish')) { <button class="button primary" type="button" (click)="service.publish()" [disabled]="service.saving()">Publish and activate</button> } }
      </section>
    }
  `
})
export class CatalogPage { readonly service = inject(CatalogService); readonly selection = inject(CatalogSelectionService); readonly entry = inject(CatalogEntryService) }
