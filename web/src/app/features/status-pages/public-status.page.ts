import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { PublicStatusService } from './public-status.service'
import { StatusViewComponent } from './status-view.component'

@Component({
  standalone: true, imports: [StatusViewComponent], providers: [PublicStatusService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `@if (service.report(); as report) { <ts-status-view [report]="report" (windowChanged)="service.select($event)" /> } @else { <main><h1>Service status</h1>@if (service.busy()) { <p role="status">Loading status…</p> }@if (service.error()) { <p role="alert">{{ service.error() }}</p><button class="button secondary" type="button" (click)="service.refresh()">Retry status</button> }</main> }`,
  styles: `:host { display:block; min-height:100dvh; } ts-status-view { min-height:100dvh; --ts-status-min-height:100dvh; } main { max-width:72rem; margin-inline:auto; padding:40px 20px; box-sizing:border-box; }`
})
export class PublicStatusPage { readonly service = inject(PublicStatusService) }
