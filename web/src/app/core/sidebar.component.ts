import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core'
import { toSignal } from '@angular/core/rxjs-interop'
import { FormsModule } from '@angular/forms'
import { NavigationEnd, Router } from '@angular/router'
import { filter, map } from 'rxjs'
import { WorkspaceService } from './workspace.service'
import { BuildInfoComponent } from './build-info.component'
import { SidebarLinksComponent } from './sidebar-links.component'
import { navigationSections, projectNavigation } from './navigation.utils'

@Component({
  selector: 'ts-sidebar', standalone: true, imports: [FormsModule, SidebarLinksComponent, BuildInfoComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <aside class="sidebar">
      <nav aria-label="Main navigation" class="sidebar-navigation">
        <ts-sidebar-links class="project-navigation" [links]="projectsLink()" (navigated)="mobileOpen.set(false)" />
      <div class="sidebar-heading">@if (showProjectPicker()) { <div class="workspace-picker"><label for="project-picker" class="eyebrow">Project</label><select id="project-picker" #projectPicker [ngModel]="workspace.projectId()" (ngModelChange)="workspace.selectProject($event, projectPicker)"><option value="">All projects</option>@for (project of workspace.projects(); track project.id) { <option [value]="project.id">{{ project.data.name }}</option> }</select>@if (workspace.projectId()) { <label for="application-picker" class="eyebrow">Application</label><select id="application-picker" #applicationPicker [ngModel]="workspace.applicationId()" (ngModelChange)="workspace.selectApplication($event, applicationPicker)"><option value="">All applications</option>@for (application of workspace.applications(); track application.id) { <option [value]="application.id">{{ application.data.name }}</option> }</select>@if (workspace.applicationError()) { <p class="small" role="alert">{{ workspace.applicationError() }}</p> } }</div> }<button type="button" class="button secondary navigation-toggle" aria-controls="sidebar-navigation" [attr.aria-expanded]="mobileOpen()" (click)="mobileOpen.set(!mobileOpen())"><span aria-hidden="true">☰</span><span class="sr-only">Navigation</span></button></div>
      <div id="sidebar-navigation" class="section-navigation" [class.mobile-open]="mobileOpen()">
        @for (section of sections(); track section.id) {
          @if (section.collapsible) {
            <details class="nav-section" [open]="section.active" [attr.aria-label]="section.title" [class.section-active]="section.active"><summary><span>{{ section.title }}</span><span class="chevron" aria-hidden="true">›</span></summary><ts-sidebar-links [links]="section.links" (navigated)="mobileOpen.set(false)" /></details>
          } @else {
            <section class="nav-section" [attr.aria-label]="section.title + ' navigation'"><h2>{{ section.title }}</h2><ts-sidebar-links [links]="section.links" (navigated)="mobileOpen.set(false)" /></section>
          }
        }
      </div>
      </nav>
      <div class="sidebar-footer" [class.mobile-open]="mobileOpen()"><ts-build-info /></div>
    </aside>
  `,
  styleUrl: './sidebar.component.css'
})
export class SidebarComponent {
  readonly workspace = inject(WorkspaceService)
  private readonly router = inject(Router)
  private readonly url = toSignal(this.router.events.pipe(filter((event): event is NavigationEnd => event instanceof NavigationEnd), map(event => event.urlAfterRedirects)), { initialValue: this.router.url })
  readonly mobileOpen = signal(false)
  readonly showProjectPicker = computed(() => this.url().split(/[?#]/)[0] !== '/projects')
  readonly projectsLink = computed(() => projectNavigation(this.url()))
  readonly sections = computed(() => navigationSections(this.workspace.projectId(), this.workspace.capabilities(), this.workspace.installationCapabilities(), this.url()))
}
