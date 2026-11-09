import { authenticationReturnPath } from '../features/auth/auth-navigation.utils'
import { inject, Injectable, signal } from '@angular/core'
import { Router } from '@angular/router'
import { SessionService } from './session.service'
import type { PasswordLogin, PasswordChange } from './generated/access.types'
import { WorkspaceService } from './workspace.service'

@Injectable({ providedIn: 'root' })
export class ApplicationService {
  private readonly router = inject(Router)
  readonly session = inject(SessionService)
  readonly workspace = inject(WorkspaceService)
  readonly error = signal('')
  readonly message = signal('')

  constructor() { this.session.onAuthenticationChanged = () => { this.workspace.reset(); this.error.set('') } }

  async start(): Promise<void> {
    await this.session.initialize()
    if (this.session.ready()) await this.loadWorkspace()
  }

  async signInLocally(): Promise<void> {
    this.error.set('')
    try { await this.session.signInLocally(); await this.clearAuthenticationFeedback(); await this.loadWorkspace() }
    catch (error) { this.error.set(error instanceof Error ? error.message : 'Sign-in failed.') }
  }

  async signOut(): Promise<void> {
    try {
      await this.session.signOut()
      this.workspace.reset(); this.message.set('')
      await this.router.navigate(['/projects'], { queryParams: { signIn: 'signed-out' } })
    } catch (error) { this.error.set(error instanceof Error ? error.message : 'Sign-out failed.') }
  }

  async signInWithPassword(body: PasswordLogin): Promise<void> { this.error.set(''); this.message.set(''); await this.session.signInWithPassword(body); await this.clearAuthenticationFeedback(); if (this.session.ready()) await this.loadWorkspace() }
  async changePassword(body: PasswordChange): Promise<void> {
    const required = this.session.mustChangePassword()
    this.message.set('')
    await this.session.changePassword(body)
    this.workspace.reset()
    if (this.session.ready()) { await this.loadWorkspace(); if (required) this.message.set('Password changed. Earlier sessions are invalid.') }
  }
  async refreshAuthentication(): Promise<void> { await this.session.loadProviders(); await this.session.refresh(); this.workspace.reset(); if (this.session.ready()) await this.loadWorkspace() }

  private async clearAuthenticationFeedback(): Promise<void> { await this.router.navigateByUrl(authenticationReturnPath(window.location.pathname + window.location.search + window.location.hash), { replaceUrl: true }) }

  private async loadWorkspace(): Promise<void> {
    try { await this.workspace.load() }
    catch (error) { this.error.set(error instanceof Error ? error.message : 'Projects could not be loaded.') }
  }
}
