import { RecordCreationService } from '../../core/record-creation.service'
import { TimingPolicyEditorService } from '../timing/timing-policy-editor.service'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { CatalogService } from './catalog.service'
import { CatalogEntryService } from './catalog-entry.service'
import { CatalogTableComponent } from './catalog-table.component'
import { CatalogFiltersComponent } from './catalog-filters.component'
import { CatalogCategoryNodesComponent } from './catalog-category-nodes.component'
import { CatalogEntryComponent } from './catalog-entry.component'
import { CatalogPlanService } from './catalog-plan.service'
import { CatalogPlanComponent } from './catalog-plan.component'
import { CatalogSelectionService } from './catalog-selection.service'
import { CatalogBulkComponent } from './catalog-bulk.component'
import { CatalogBulkMetadataService } from './catalog-bulk-metadata.service'
import { CatalogSavedSelectionsComponent } from './catalog-saved-selections.component'
import { CatalogSavedSelectionsService } from './catalog-saved-selections.service'
import { CatalogTransferComponent } from './catalog-transfer.component'
import { HTTPCheckEditorService } from './http-check-editor.service'
import { CatalogTestEditorService } from './catalog-test-editor.service'
import { CatalogTestFormComponent } from './catalog-test-form.component'

@Component({
  standalone: true,
  imports: [CatalogTableComponent, CatalogFiltersComponent, CatalogCategoryNodesComponent, FormsModule, ReactiveFormsModule, RouterLink, CatalogEntryComponent, CatalogPlanComponent, CatalogBulkComponent, CatalogTransferComponent, CatalogSavedSelectionsComponent, CatalogTestFormComponent],
  providers: [RecordCreationService, CatalogTestEditorService, HTTPCheckEditorService, TimingPolicyEditorService, CatalogService, CatalogEntryService, CatalogPlanService, CatalogSelectionService, CatalogBulkMetadataService, CatalogSavedSelectionsService],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">{{ service.workspace.project()?.name || 'Project' }}</span><h1>Test catalog</h1><p class="muted">Know what you test. Keep every release accountable.</p></div>
      @if (service.workspace.can('catalog.create')) { <a class="button primary" [routerLink]="['/projects', service.projectId(), 'tests', 'new']" [queryParams]="{ suite: service.suiteFilter() || undefined }">+ Create test</a> }
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
        <ts-catalog-test-form (saved)="service.afterEdit()" />
      </section>
    }
    <section class="panel catalog-panel" aria-labelledby="catalog-title">
      <div class="section-top"><h2 id="catalog-title">All tests <span class="count-label">{{ service.visible().length }}</span></h2><span class="small muted">Definitions and history use stable identifiers.</span></div>
      <ts-catalog-filters />
      <ts-catalog-bulk />
      <div class="catalog-layout" [class.with-categories]="service.view() === 'categories'">
        @if (service.view() === 'categories') {
          <nav class="category-navigation" aria-label="Catalog categories"><h3>Categories</h3><button class="button small secondary" type="button" [attr.aria-current]="!service.filters().category ? 'true' : null" (click)="service.filter('category', '')">All categories</button><p class="small muted">Counts follow the other filters. Parent categories include their children.</p><ts-catalog-category-nodes [nodes]="service.categories()" [selected]="service.filters().category || ''" (choose)="service.filter('category', $event); service.filter('descendants', true)" /></nav>
        }
        <div class="catalog-results">
      @if (service.loading()) { <p class="loading-state" role="status">Loading the catalog…</p> }
      @else if (service.visible().length) {
        <ts-catalog-table />
      } @else {
        <div class="empty-state"><span class="empty-icon" aria-hidden="true">✓</span><h3>{{ service.items().length ? 'No matching tests.' : 'Your next good answer starts here.' }}</h3><p>{{ service.items().length ? 'Try another category or clear filters to see all active tests.' : 'Create a suite, then add checks or publish a versioned test catalog.' }}</p>@if (!service.suites().length) { <a class="button secondary" [routerLink]="['/projects', service.projectId(), 'configuration']">Set up a suite</a> }</div>
      }
        </div>
      </div>
    </section>
    @if (service.suites().length && (service.workspace.can('catalog.import') || service.workspace.can('catalog.export'))) { <ts-catalog-transfer /> }
    @if (service.suites().length) { <p><a class="button secondary" [routerLink]="['/projects', service.projectId(), 'releases']">Browse retained releases</a></p> }
    @if (service.suites().length) { <ts-catalog-saved-selections /> }
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
  `,
  styles: [`:host { display: block; } .catalog-layout { min-width: 0; } .with-categories { display: grid; grid-template-columns: minmax(180px, 240px) minmax(0, 1fr); gap: 20px; } .category-navigation { min-width: 0; border-right: 1px solid var(--border); padding-right: 12px; } .catalog-results { min-width: 0; } @media (max-width: 1000px) { .with-categories { grid-template-columns: minmax(0, 1fr); } .category-navigation { border-right: 0; border-bottom: 1px solid var(--border); padding: 0 0 16px; } }`]
})
export class CatalogPage { readonly service = inject(CatalogService); readonly selection = inject(CatalogSelectionService); readonly entry = inject(CatalogEntryService) }
