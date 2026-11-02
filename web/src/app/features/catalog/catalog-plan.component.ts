import { DatePipe, JsonPipe } from '@angular/common'
import { afterRenderEffect, ChangeDetectionStrategy, Component, ElementRef, inject, viewChild } from '@angular/core'
import { CatalogPlanService } from './catalog-plan.service'

@Component({
  selector: 'ts-catalog-plan', standalone: true, imports: [DatePipe, JsonPipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.plan(); as row) {
      <section class="panel" aria-labelledby="catalog-plan-title">
        <div class="section-top"><h2 #heading tabindex="-1" id="catalog-plan-title">Review {{ row.data.kind }} changes</h2><button class="button subtle" type="button" (click)="service.clear()" [disabled]="service.saving()">Close preview</button></div>
        <p class="small muted">Suite {{ row.data.suiteId }} · catalog revision {{ row.data.catalogRevision }} · expires {{ row.data.expiresAt | date:'mediumTime' }}. This preview does not change the catalog.</p>
        @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }} Reload the catalog and create a new preview if its revisions changed.</p> }
        @if (row.data.issues.length) { <ul class="notice error" aria-label="Catalog validation errors">@for (issue of row.data.issues; track $index) { <li>{{ issue.testId || 'Catalog' }} · {{ issue.field }}: {{ issue.message }}</li> }</ul> }
        <div class="table-scroll"><table><caption class="sr-only">Proposed catalog changes</caption><thead><tr><th scope="col">Test</th><th scope="col">Action</th><th scope="col">State</th><th scope="col">Safety</th><th scope="col">Comparison</th></tr></thead><tbody>
          @for (change of row.data.changes; track $index) { <tr><td><strong>{{ change.test.title }}</strong><code class="table-subtitle">{{ change.test.id }}</code></td><td>{{ change.action }}</td><td>{{ change.before?.state || 'New' }} → {{ change.test.state }}</td><td>{{ change.before?.safety || 'New' }} → {{ change.test.safety }}</td><td><details><summary>Inspect definitions</summary><div class="definition-comparison"><div><h3>Before</h3><pre>{{ change.before | json }}</pre></div><div><h3>After</h3><pre>{{ change.test | json }}</pre></div></div></details></td></tr> }
        </tbody></table></div>
        <button class="button primary more-button" type="button" [disabled]="service.saving() || row.data.issues.length > 0" (click)="service.apply()">{{ service.saving() ? 'Applying…' : 'Apply reviewed changes' }}</button>
      </section>
    }
  `
})
export class CatalogPlanComponent {
  readonly service = inject(CatalogPlanService)
  private readonly heading = viewChild<ElementRef<HTMLHeadingElement>>('heading')
  constructor() { afterRenderEffect(() => { if (this.service.plan()) this.heading()?.nativeElement.focus() }) }
}
