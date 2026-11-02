import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router'
import { ApplicationService } from './core/application.service'
import { ThemeService } from './core/theme.service'

@Component({
  selector: 'ts-root',
  standalone: true,
  imports: [FormsModule, RouterLink, RouterLinkActive, RouterOutlet],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a class="skip-link" href="#main">Skip to content</a>
    <header class="topbar">
      <a class="brand" routerLink="/projects"><span class="brand-symbol" aria-hidden="true">✓</span><span>Test<span class="brand-light">Studio</span></span></a>
      <span class="header-description">Clarity in every run.</span>
      <div class="header-actions">
        <label class="sr-only" for="theme">Color theme</label>
        <select id="theme" class="theme-select" [value]="theme.preference()" (change)="theme.set($any($event.target).value)">
          <option value="system">System theme</option><option value="light">Light theme</option><option value="dark">Dark theme</option>
        </select>
        @if (app.session.user(); as user) {
          <span class="user-name">{{ user.name }}</span>
          <button type="button" class="button subtle" (click)="app.signOut()">Sign out</button>
        }
      </div>
    </header>
    @if (app.session.loading()) {
      <main id="main" class="signin"><p role="status">Loading your workspace…</p></main>
    } @else if (!app.session.signedIn()) {
      <main id="main" class="signin">
        <section class="signin-card" aria-labelledby="signin-title">
          <span class="eyebrow">Your testing workspace</span>
          <h1 id="signin-title">A clear view of<br>what works.</h1>
          <p class="muted">Manage your test catalog, run with confidence, and keep the evidence that explains every result.</p>
          @if (app.session.error() || app.error()) { <p class="notice error" role="alert">{{ app.session.error() || app.error() }}</p> }
          @for (provider of app.session.providers().providers; track provider.id) {
            <a class="button primary full" [href]="'/auth/login/' + provider.id">Continue with {{ provider.name }}</a>
          }
          @if (app.session.providers().localEnabled) {
            <button type="button" class="button primary full" (click)="app.signInLocally()">Open local workspace <span aria-hidden="true">→</span></button>
            <p class="small muted">Local administrator access on this computer.</p>
          } @else if (app.session.providers().providers.length === 0 && !app.session.error()) {
            <p class="notice">No sign-in providers are configured for this installation.</p>
          }
          <div class="signin-footer"><span class="status-dot" aria-hidden="true"></span> Independent testing. Your infrastructure.</div>
        </section>
      </main>
    } @else {
      <div class="app-layout">
        <aside class="sidebar">
          <div class="workspace-picker"><label for="project-picker" class="eyebrow">Workspace</label>
            <select id="project-picker" [ngModel]="app.workspace.projectId()" (ngModelChange)="app.workspace.selectProject($event)">
              <option value="">All projects</option>
              @for (project of app.workspace.projects(); track project.id) { <option [value]="project.id">{{ project.data.name }}</option> }
            </select>
          </div>
          <nav aria-label="Main navigation">
            <a routerLink="/projects" routerLinkActive="active" [routerLinkActiveOptions]="{exact: true}"><span aria-hidden="true">▦</span> Projects</a>
            @if (app.workspace.projectId()) {
              <a [routerLink]="['/projects', app.workspace.projectId(), 'catalog']" routerLinkActive="active"><span aria-hidden="true">≡</span> Test catalog</a>
              <a [routerLink]="['/projects', app.workspace.projectId(), 'configuration']" routerLinkActive="active"><span aria-hidden="true">⚙</span> Project settings</a>
            }
          </nav>
          <div class="sidebar-footer"><span class="status-dot" aria-hidden="true"></span><span>TestStudio<br><small>Built for clear answers.</small></span></div>
        </aside>
        <main id="main" class="workspace-main">
          @if (app.error()) { <p class="notice error" role="alert">{{ app.error() }}</p> }
          <router-outlet />
        </main>
      </div>
    }
  `
})
export class AppComponent {
  readonly app = inject(ApplicationService)
  readonly theme = inject(ThemeService)
  constructor() { void this.app.start() }
}
