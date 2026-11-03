import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core'
import { toSignal } from '@angular/core/rxjs-interop'
import { FormsModule } from '@angular/forms'
import { NavigationEnd, Router } from '@angular/router'
import { filter, map } from 'rxjs'
import { WorkspaceService } from './workspace.service'
import { SidebarLinksComponent } from './sidebar-links.component'
import { navigationSections } from './navigation.utils'

@Component({
  selector: 'ts-sidebar', standalone: true, imports: [FormsModule, SidebarLinksComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <aside class="sidebar">
      <div class="sidebar-heading"><div class="workspace-picker"><label for="project-picker" class="eyebrow">Workspace</label><select id="project-picker" [ngModel]="workspace.projectId()" (ngModelChange)="workspace.selectProject($event)"><option value="">All projects</option>@for (project of workspace.projects(); track project.id) { <option [value]="project.id">{{ project.data.name }}</option> }</select></div><button type="button" class="button secondary navigation-toggle" aria-controls="sidebar-navigation" [attr.aria-expanded]="mobileOpen()" (click)="mobileOpen.set(!mobileOpen())"><span aria-hidden="true">☰</span><span class="sr-only">Navigation</span></button></div>
      <nav id="sidebar-navigation" aria-label="Main navigation" [class.mobile-open]="mobileOpen()">
        @for (section of sections(); track section.id) {
          @if (section.collapsible) {
            <details class="nav-section" [open]="section.active" [attr.aria-label]="section.title" [class.section-active]="section.active"><summary><span>{{ section.title }}</span><span class="chevron" aria-hidden="true">›</span></summary><ts-sidebar-links [links]="section.links" (navigated)="mobileOpen.set(false)" /></details>
          } @else {
            <section class="nav-section" [attr.aria-label]="section.title + ' navigation'"><h2>{{ section.title }}</h2><ts-sidebar-links [links]="section.links" (navigated)="mobileOpen.set(false)" /></section>
          }
        }
      </nav>
      <div class="sidebar-footer"><span class="status-dot" aria-hidden="true"></span><span>TestStudio<br><small>Built for clear answers.</small></span></div>
    </aside>
  `,
  styleUrl: './sidebar.component.css'
})
export class SidebarComponent {
  readonly workspace = inject(WorkspaceService)
  private readonly router = inject(Router)
  private readonly url = toSignal(this.router.events.pipe(filter((event): event is NavigationEnd => event instanceof NavigationEnd), map(event => event.urlAfterRedirects)), { initialValue: this.router.url })
  readonly mobileOpen = signal(false)
  readonly sections = computed(() => navigationSections(this.workspace.projectId(), this.workspace.capabilities(), this.workspace.installationCapabilities(), this.url()))
}
