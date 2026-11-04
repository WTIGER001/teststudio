import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { CatalogService } from './catalog.service'

@Component({
  selector: 'ts-catalog-filters', standalone: true, imports: [FormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="toolbar">
      <div class="search-field"><label class="sr-only" for="catalog-search">Search tests</label><span aria-hidden="true">⌕</span><input id="catalog-search" type="search" placeholder="Search by name, identifier, category, or tag…" [ngModel]="service.query()" (ngModelChange)="service.query.set($event)"></div>
      <label class="sr-only" for="catalog-suite-filter">Filter by suite</label><select id="catalog-suite-filter" [ngModel]="service.suiteFilter()" (ngModelChange)="service.suiteFilter.set($event)"><option value="">All suites</option>@for (suite of service.suites(); track suite.id) { <option [value]="suite.id">{{ suite.data.name }}</option> }</select>
      @if (!service.probeOverview) { <label class="sr-only" for="catalog-type">Filter by type</label><select id="catalog-type" [ngModel]="service.typeFilter()" (ngModelChange)="service.typeFilter.set($event)"><option value="">All types</option>@for (type of service.types(); track type.value) { <option [value]="type.value">{{ type.label }}</option> }</select> }
    </div>
    <div class="row-actions browse-controls">
      <fieldset><legend class="sr-only">Catalog view</legend><label><input type="radio" name="catalog-view" value="table" [ngModel]="service.view()" (ngModelChange)="service.view.set($event)"> Table</label><label><input type="radio" name="catalog-view" value="categories" [ngModel]="service.view()" (ngModelChange)="service.view.set($event)"> By category</label></fieldset>
      <div class="sort-control"><label for="catalog-sort">Sort tests</label><select id="catalog-sort" [ngModel]="service.filters().sort || 'title'" (ngModelChange)="service.filter('sort', $event)"><option value="title">Name A–Z</option><option value="title-desc">Name Z–A</option><option value="id">Identifier</option><option value="category">Category</option><option value="updated">Recently updated</option></select></div>
      <button class="button small subtle" type="button" (click)="service.clearFilters()">Clear filters</button>
    </div>
    @if (service.applicationNotice()) { <p class="small muted">{{ service.applicationNotice() }}</p> }
    <p class="small muted" role="status">Showing {{ service.visible().length }} of {{ service.page().matched }} matching tests</p>
    <details class="advanced-filters"><summary>Advanced filters</summary>
      <div class="form-grid">
        <div class="field"><label for="catalog-state">Lifecycle state</label><select id="catalog-state" [ngModel]="service.state()" (ngModelChange)="service.state.set($event)"><option value="active">Active catalog</option><option value="draft">Draft</option><option value="enabled">Enabled</option><option value="disabled">Disabled</option><option value="archived">Archived</option><option value="all">All states</option></select></div>
        <div class="field"><label for="catalog-category">Category path</label><input id="catalog-category" placeholder="Availability / Local" [ngModel]="service.categoryInput()" (ngModelChange)="service.filter('category', $event)"><label class="checkbox-field"><input type="checkbox" [ngModel]="service.filters().descendants !== false" (ngModelChange)="service.filter('descendants', $event)"> Include child categories</label></div>
        <div class="field"><label for="catalog-tag">Tag</label><select id="catalog-tag" [ngModel]="service.filters().tag || ''" (ngModelChange)="service.filter('tag', $event)"><option value="">All tags</option>@for (option of service.options().tag; track option.value) { <option [value]="option.value">{{ option.label }}</option> }</select></div>
        <div class="field"><label for="catalog-owner">Owner</label><select id="catalog-owner" [ngModel]="service.filters().owner || ''" (ngModelChange)="service.filter('owner', $event)"><option value="">All owners</option>@for (option of service.options().owner; track option.value) { <option [value]="option.value">{{ option.label }}</option> }</select></div>
        <div class="field"><label for="catalog-runner">Runner</label><select id="catalog-runner" [ngModel]="service.filters().runner || ''" (ngModelChange)="service.filter('runner', $event)"><option value="">All runners</option>@for (option of service.options().runner; track option.value) { <option [value]="option.value">{{ option.label }}</option> }</select></div>
        <div class="field"><label for="catalog-safety">Safety label</label><select id="catalog-safety" [ngModel]="service.filters().safety || ''" (ngModelChange)="service.filter('safety', $event)"><option value="">All safety labels</option>@for (option of service.options().safety; track option.value) { <option [value]="option.value">{{ option.label }}</option> }</select></div>
        <div class="field"><label for="catalog-source">Source</label><select id="catalog-source" [ngModel]="service.filters().source || ''" (ngModelChange)="service.filter('source', $event)"><option value="">All sources</option>@for (option of service.options().source; track option.value) { <option [value]="option.value">{{ option.label }}</option> }</select></div>
      </div>
      <p class="small muted">Filters narrow the working catalog. They do not change definitions, publish a release, or clear selected tests.</p>
    </details>
    @if (service.filters().category) { <p class="small">Category: <strong>{{ service.categoryTitle() }}</strong>{{ service.filters().descendants === false ? ' · Exact category' : ' · Includes child categories' }} <button class="button small subtle" type="button" (click)="service.filter('category', '')">Clear category</button></p> }
  `,
  styles: [`:host { display: block; min-width: 0; } .browse-controls { flex-wrap: wrap; gap: 12px; padding-bottom: 12px; } .sort-control { display: flex; align-items: center; gap: 8px; min-width: 0; } .sort-control label { white-space: nowrap; } fieldset { display: flex; flex-wrap: wrap; gap: 12px; border: 0; padding: 0; margin: 0; } fieldset label { display: inline-flex; align-items: center; gap: 6px; } input[type=radio], input[type=checkbox] { width: auto; } .advanced-filters { margin-bottom: 12px; } summary { cursor: pointer; } .form-grid { margin-top: 16px; } .field { align-content: start; } select { max-width: 100%; min-width: 0; } .toolbar { flex-wrap: wrap; }`]
})
export class CatalogFiltersComponent { readonly service = inject(CatalogService) }
