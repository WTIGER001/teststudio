import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { StatusPagesService } from './status-pages.service'

@Component({
  standalone: true, imports: [RouterLink], providers: [StatusPagesService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<div class="page-heading"><div><span class="eyebrow">Project status</span><h1>Status pages</h1><p class="muted">Publish selected HTTP probes for visitors and embedded dashboards.</p></div>@if (service.canManage()) { <a class="button primary" [routerLink]="['/projects', service.project(), 'status-pages', 'new']">Create status page</a> }</div>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }<button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.load()">Refresh status pages</button>
    <div class="pages">@for (page of service.pages(); track page.id) { <article class="panel"><h2>{{ page.data.title }}</h2><p>{{ page.data.probes.length }} probes · {{ page.data.published ? 'Published' : 'Private draft' }}</p><a class="button secondary" [routerLink]="['/projects', service.project(), 'status-pages', page.id]">{{ service.canManage() ? 'Manage' : 'Inspect' }} {{ page.data.title }}</a></article> } @empty { @if (!service.busy() && !service.error()) { <p class="panel">No status pages yet. Create a draft, choose probes, then publish when ready.</p> } }</div>
    @if (service.busy()) { <p role="status">Loading status pages…</p> }@if (service.cursor()) { <button class="button secondary" [disabled]="service.busy()" (click)="service.load(true)">More status pages</button> }`,
  styles: `.pages {display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,22rem),1fr));gap:20px;margin-top:20px;} h2 {overflow-wrap:anywhere;}`
})
export class StatusPagesPage { readonly service = inject(StatusPagesService) }
