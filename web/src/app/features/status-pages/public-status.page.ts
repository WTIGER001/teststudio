import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { PublicStatusService } from './public-status.service'
import { StatusViewComponent } from './status-view.component'
import { HelpComponent } from '../help/help.component'

@Component({
  standalone: true, imports: [StatusViewComponent, HelpComponent], providers: [PublicStatusService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<div class="public-help"><ts-help /></div>@if (service.report(); as report) { <ts-status-view [report]="report" (windowChanged)="service.select($event)" /> } @else { <main><h1>Service status</h1>@if (service.busy()) { <p role="status">Loading status…</p> }@if (service.error()) { <p role="alert">{{ service.error() }}</p><button class="button secondary" type="button" (click)="service.refresh()">Retry status</button> }</main> }`,
  styles: `:host { display:block; min-height:100dvh; } ts-status-view { min-height:100dvh; --status-min-height:100dvh; --status-top:56px; } .public-help { position:absolute; top:8px; right:8px; z-index:3; text-align:right; padding:8px 8px 0; } main { padding:40px 20px; }`
})
export class PublicStatusPage { readonly service = inject(PublicStatusService) }
