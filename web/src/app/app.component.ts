import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink, RouterOutlet } from '@angular/router'
import { ApplicationService } from './core/application.service'
import { ThemeService } from './core/theme.service'
import { SidebarComponent } from './core/sidebar.component'
import { PasswordSigninComponent } from './features/auth/password-signin.component'
import { RequiredPasswordComponent } from './features/auth/required-password.component'
import { AuthFeedbackComponent } from './features/auth/auth-feedback.component'
import { AuthNavigationService } from './features/auth/auth-navigation.service'
import { HelpComponent } from './features/help/help.component'

@Component({
  selector: 'ts-root',
  standalone: true,
  imports: [RouterLink, RouterOutlet, SidebarComponent, HelpComponent, PasswordSigninComponent, RequiredPasswordComponent, AuthFeedbackComponent],
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
        <ts-help />
        @if (app.session.user(); as user) {
          @if (app.session.ready()) { <a class="user-name" routerLink="/account" aria-label="Your account">{{ user.name }}</a> } @else { <span class="user-name">{{ user.name }}</span> }
          <button type="button" class="button subtle" [disabled]="app.session.authenticating()" (click)="app.signOut()">Sign out</button>
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
          <ts-auth-feedback />
          @if (app.session.error() || app.error()) { <p class="notice error" role="alert">{{ app.session.error() || app.error() }}</p> }
          @if (app.session.providers().localAccountsEnabled) { <ts-password-signin /> }
          @if (app.message()) { <p class="notice success" role="status">{{ app.message() }}</p> }
          @for (provider of app.session.providers().providers; track provider.id) {
            <a class="button primary full" [href]="authentication.providerPath(provider.id)" [attr.aria-disabled]="app.session.authenticating()" (click)="app.session.guardProviderNavigation($event)">Continue with {{ provider.name }}</a>
          }
          @if (app.session.providers().localEnabled) {
            <button type="button" class="button primary full" [disabled]="app.session.authenticating()" (click)="app.signInLocally()">Open local workspace <span aria-hidden="true">→</span></button>
            <p class="small muted">Local administrator access on this computer.</p>
          } @else if (app.session.providers().providers.length === 0 && !app.session.providers().localAccountsEnabled && !app.session.error()) {
            <p class="notice">No sign-in providers are configured for this installation.</p>
          }
          <div class="signin-footer"><span class="status-dot" aria-hidden="true"></span> Independent testing. Your infrastructure.</div>
        </section>
      </main>
    } @else if (app.session.mustChangePassword()) {
      <main id="main" class="signin"><ts-required-password /></main>
    } @else {
      <div class="app-layout">
        <ts-sidebar />
        <main id="main" class="workspace-main"><ts-auth-feedback />
          @if (app.error()) { <p class="notice error" role="alert">{{ app.error() }}</p> }
          @if (app.message()) { <p class="notice success" role="status">{{ app.message() }}</p> }
          <router-outlet />
        </main>
      </div>
    }
  `
})
export class AppComponent {
  readonly authentication=inject(AuthNavigationService)
  readonly app = inject(ApplicationService)
  readonly theme = inject(ThemeService)
  constructor() { void this.app.start() }
}
