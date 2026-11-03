import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { SuiteManagerComponent } from './suite-manager.component'
import { SuiteManagerService } from './suite-manager.service'
import { ConfigurationService } from './configuration.service'

@Component({
  standalone: true,
  imports: [ReactiveFormsModule, SuiteManagerComponent],
  providers: [ConfigurationService, SuiteManagerService],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">{{ service.workspace.project()?.name || 'Project' }}</span><h1>Project settings</h1><p class="muted">Connect your applications, execution targets, and suites.</p></div></div>
    <div class="tabs" aria-label="Configuration sections">
      <button type="button" [class.active]="service.kind() === 'applications'" [attr.aria-pressed]="service.kind() === 'applications'" (click)="service.select('applications')">Applications <span>{{ service.applications().length }}</span></button>
      <button type="button" [class.active]="service.kind() === 'suites'" [attr.aria-pressed]="service.kind() === 'suites'" (click)="service.select('suites')">Suites <span>{{ service.suites().length }}</span></button>
      <button type="button" [class.active]="service.kind() === 'environments'" [attr.aria-pressed]="service.kind() === 'environments'" (click)="service.select('environments')">Environments <span>{{ service.environments().length }}</span></button>
    </div>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.success()) { <p class="notice success" role="status">{{ service.success() }}</p> }
    <div class="two-column">
      <section class="panel" aria-labelledby="configuration-title">
        <h2 id="configuration-title">Registered {{ service.kind() }}</h2>
        @if (service.loading()) { <p role="status" class="muted">Loading settings…</p> }
        @switch (service.kind()) {
          @case ('applications') {
            @for (item of service.applications(); track item.id) { <article class="resource-row"><div><h3>{{ item.data.name }}</h3><code>{{ item.id }}</code></div><span class="badge">Application</span></article> }
            @empty { <div class="empty-state compact"><h3>No applications yet.</h3><p>Add an application before registering its environments.</p></div> }
          }
          @case ('suites') {
            @for (item of service.suites(); track item.id) { <article class="resource-row"><div><h3>{{ item.data.name }}</h3><code>{{ item.id }}</code></div><div class="row-actions"><span class="badge" [class.success]="!item.data.archived && item.data.activeReleaseId" [class.warning]="item.data.archived">{{ item.data.archived ? 'Archived' : item.data.activeReleaseId ? 'Published' : 'No active release' }}</span><button type="button" class="button small secondary" (click)="manager.open(item.id)" [attr.aria-label]="'Inspect suite ' + item.data.name">Inspect suite</button></div></article> }
            @empty { <div class="empty-state compact"><h3>Your catalog starts with a suite.</h3><p>A suite groups tests and their immutable releases.</p></div> }
          }
          @case ('environments') {
            @for (item of service.environments(); track item.id) { <article class="resource-row"><div><h3>{{ item.data.name }}</h3><code>{{ item.id }}</code></div><span class="badge">{{ item.data.stage }}</span><span class="badge" [class.warning]="item.data.protected">{{ item.data.protected ? 'Protected' : 'Standard' }}</span></article> }
            @empty { <div class="empty-state compact"><h3>No environments yet.</h3><p>Register where an application runs and whether it needs protected execution.</p></div> }
          }
        }
      </section>
      @if (service.canCreate()) {
        <section class="panel" aria-labelledby="add-resource-title"><h2 id="add-resource-title">Add {{ service.kind() === 'suites' ? 'a suite' : service.kind() === 'environments' ? 'an environment' : 'an application' }}</h2>
          <form [formGroup]="service.form" (ngSubmit)="service.create()">
            <div class="field"><label for="resource-name">Name</label><input id="resource-name" formControlName="name" required autocomplete="off"></div>
            <div class="field"><label for="resource-id">Permanent identifier</label><input id="resource-id" formControlName="id" required pattern="[A-Za-z0-9](?:[A-Za-z0-9._]|-){0,127}" autocomplete="off"></div>
            @if (service.kind() !== 'applications') {
              <div class="field"><label for="resource-application">Application {{ service.kind() === 'suites' ? '(optional)' : '' }}</label><select id="resource-application" formControlName="application" [required]="service.kind() === 'environments'"><option value="">Select an application</option>@for (app of service.applications(); track app.id) { <option [value]="app.id">{{ app.data.name }}</option> }</select></div>
            }
            @if (service.kind() === 'suites') { <label class="checkbox-field"><input type="checkbox" formControlName="requiresTarget">Requires a target environment</label><p class="small muted">Turn this off for pure unit test suites.</p> }
            @if (service.kind() === 'environments') {
              <div class="field"><label for="environment-stage">Stage</label><select id="environment-stage" formControlName="stage"><option value="development">Development</option><option value="beta">Beta</option><option value="production">Production</option><option value="local">Local</option></select></div>
              <div class="field"><label for="endpoint-name">Endpoint name</label><input id="endpoint-name" formControlName="endpoint" required></div>
              <div class="field"><label for="endpoint-url">Endpoint URL</label><input id="endpoint-url" type="url" formControlName="url" required placeholder="https://example.com"></div>
              <label class="checkbox-field"><input type="checkbox" formControlName="protected">Require production-safe tests</label><p class="small muted">Protection is separate from the named stage.</p>
            }
            <button class="button primary" type="submit" [disabled]="service.form.invalid || service.saving()">{{ service.saving() ? 'Creating…' : 'Create' }}</button>
          </form>
        </section>
      }
    </div>
    @if (service.kind() === 'suites') { <ts-suite-manager /> }
  `
})
export class ConfigurationPage { readonly service = inject(ConfigurationService); readonly manager = inject(SuiteManagerService) }
