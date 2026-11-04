import { DatePipe, JsonPipe } from '@angular/common'
import { afterRenderEffect, ChangeDetectionStrategy, Component, ElementRef, inject, viewChild } from '@angular/core'
import { RouterLink } from '@angular/router'
import { CatalogService } from './catalog.service'

@Component({
  selector: 'ts-catalog-publication-review', standalone: true, imports: [DatePipe, JsonPipe, RouterLink], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.publicationDenied()) { <p><button class="button secondary" type="button" (click)="service.workspace.refreshAccess()">Refresh access</button></p> }
    @if (service.review(); as review) {
      <section class="panel" aria-labelledby="release-preview-title"><h2 #heading tabindex="-1" id="release-preview-title">Release preview</h2>
        <p>{{ review.total }} definitions · {{ review.enabled }} enabled · {{ review.unchanged }} unchanged.</p>
        @if (!review.activeReleaseId) { <p class="notice">This suite has no active release. These definitions form its first active catalog when you activate it.</p> }
        @if (service.reviewStale()) { <p class="notice warning" role="alert">This review no longer matches current work. Select Validate and preview to create a new review before publishing.</p> }
        @for (issue of review.issues; track $index) { <p class="notice warning">{{ issue.testId }} · {{ issue.field }}: {{ issue.message }}</p> }
        @if (!review.issues.length) { <p class="notice success">Validation passed. This does not indicate that the tests have run or passed.</p> }
        <h3>Changes from the active release</h3>
        @for (change of review.changes; track change.id) {
          <article class="change-card" [attr.aria-label]="change.id + ' publication change'"><div class="section-top"><h4>{{ change.title }}</h4><span class="badge">{{ change.action }}</span></div><code>{{ change.id }}</code>
            @if (change.fields.length) { <p class="small">Changed fields: @for (field of change.fields; track field) { <code class="field-name">{{ field }}</code> }</p> }
            @if (change.safetyChanged) { <p class="notice warning">Safety: {{ change.before?.safety }} → {{ change.after?.safety }}. Review the rationale and operating conditions.</p> }
            <details><summary>Before and after definition</summary><pre class="log-view">{{ { before: change.before, after: change.after } | json }}</pre></details>
          </article>
        } @empty { <p class="small muted">No test definition changes from the active release.</p> }
        @if (review.artifactChanged || review.timingChanged || review.captureChanged) { <p class="notice">Release settings change: @if (review.artifactChanged) { executable source; } @if (review.timingChanged) { timing defaults; } @if (review.captureChanged) { capture defaults; } inspect Advanced review details before publishing.</p> }
        <h3>Schedules you can inspect</h3>
        <p class="small muted">Activating changes future runs that follow this suite's active release. Pinned schedules keep their selected release. Publishing alone leaves the active release unchanged.</p>
        <ul>@for (schedule of review.schedules; track schedule.id) { <li>{{ schedule.name }} · {{ schedule.enabled ? 'Enabled' : 'Disabled' }} · {{ schedule.followsActive ? 'Follows active release' : 'Pinned release' }}@if (schedule.releaseId) { <code class="table-subtitle">{{ schedule.releaseId }}</code> }</li> } @empty { <li>No readable schedules are listed for this suite.</li> }</ul>
        <details><summary>Advanced review details</summary><dl class="detail-grid"><dt>Review expires</dt><dd>{{ review.expiresAt | date:'medium' }}</dd><dt>Active release</dt><dd class="wrap-anywhere">{{ review.activeReleaseId || 'None' }}</dd><dt>Draft revision</dt><dd>{{ review.draftRevision }}</dd></dl><pre class="log-view">{{ { before: review.beforeSettings, after: review.afterSettings } | json }}</pre></details>
        @if (service.workspace.can('catalog.publish') && !review.issues.length) {
          <div class="row-actions publication-actions"><button class="button secondary" type="button" (click)="service.publish(false)" [disabled]="service.saving() || service.reviewStale() || service.publicationAttempt() === true">Publish release</button><button class="button primary" type="button" (click)="service.publish(true)" [disabled]="service.saving() || service.reviewStale() || service.publicationAttempt() === false">Publish and activate</button></div>
          <p class="small muted">Publication freezes this reviewed draft. Existing runs keep their original release. Starting a run remains a separate action.</p>
        }
        <p><a [routerLink]="['/projects', review.projectId, 'releases']" [queryParams]="{ suiteId: review.suiteId }">Open Suite releases</a></p>
      </section>
    }
  `,
  styles: [`:host { display: block; min-width: 0; } code { overflow-wrap: anywhere; } .change-card { border: 1px solid var(--border); border-radius: 8px; padding: 14px; margin: 12px 0; min-width: 0; } .change-card h4 { margin: 0; overflow-wrap: anywhere; } .change-card .section-top { flex-wrap: wrap; gap: 12px; margin-bottom: 8px; } .field-name { display: inline-block; margin-right: 8px; } .publication-actions { flex-wrap: wrap; margin-top: 20px; } summary { cursor: pointer; } li { margin: 8px 0; }`]
})
export class CatalogPublicationReviewComponent {
  readonly service = inject(CatalogService)
  private readonly heading = viewChild<ElementRef<HTMLHeadingElement>>('heading')
  constructor() { afterRenderEffect(() => { if (this.service.review()) this.heading()?.nativeElement.focus() }) }
}
