import { DatePipe, JsonPipe } from '@angular/common'
import { afterRenderEffect, ChangeDetectionStrategy, Component, ElementRef, inject, viewChild } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { CatalogEntryService } from './catalog-entry.service'

@Component({
  selector: 'ts-catalog-entry', standalone: true, imports: [DatePipe, JsonPipe, ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.entry(); as entry) {
      <section class="panel" aria-labelledby="catalog-entry-title"><div class="section-top"><h2 #heading tabindex="-1" id="catalog-entry-title">{{ service.mode() === 'history' ? 'Revision history' : 'Duplicate test' }} · {{ entry.id }}</h2><button class="button subtle" type="button" (click)="service.close()">Close details</button></div>
        @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
        @if (service.mode() === 'duplicate') {
          <p class="small muted">Create a new draft identifier and independent history. The original test keeps its releases and results.</p>
          <form [formGroup]="service.form" (ngSubmit)="service.duplicate()"><div class="field"><label for="duplicate-test-id">New permanent identifier</label><input id="duplicate-test-id" formControlName="id" required pattern="[A-Za-z0-9](?:[A-Za-z0-9._]|-){0,127}"></div><button class="button primary" type="submit" [disabled]="service.newID.invalid || service.loading()">Create duplicate</button></form>
        } @else {
          <p class="small muted">{{ entry.data.title }} · {{ entry.suiteName }}</p>
          @for (revision of service.revisions(); track revision.id) { <details class="revision-entry"><summary>Revision {{ revision.number }} · {{ revision.data.at | date:'medium' }} · {{ revision.data.actor }}</summary><pre class="log-view">{{ revision.data.test | json }}</pre></details> } @empty { <p class="small muted">{{ service.loading() ? 'Loading revisions…' : 'No revisions are available.' }}</p> }
        }
      </section>
    }
  `
})
export class CatalogEntryComponent {
  readonly service = inject(CatalogEntryService)
  private readonly heading = viewChild<ElementRef<HTMLHeadingElement>>('heading')
  constructor() { afterRenderEffect(() => { if (this.service.entry()) this.heading()?.nativeElement.focus() }) }
}
