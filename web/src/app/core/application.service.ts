import { inject, Injectable, signal } from '@angular/core'
import { Router } from '@angular/router'
import { SessionService } from './session.service'
import { WorkspaceService } from './workspace.service'

@Injectable({ providedIn: 'root' })
export class ApplicationService {
  private readonly router = inject(Router)
  readonly session = inject(SessionService)
  readonly workspace = inject(WorkspaceService)
  readonly error = signal('')

  async start(): Promise<void> {
    await this.session.initialize()
    if (this.session.signedIn()) await this.loadWorkspace()
  }

  async signInLocally(): Promise<void> {
    this.error.set('')
    try { await this.session.signInLocally(); await this.loadWorkspace() }
    catch (error) { this.error.set(error instanceof Error ? error.message : 'Sign-in failed.') }
  }

  async signOut(): Promise<void> {
    try {
      await this.session.signOut()
      this.workspace.projects.set([])
      this.workspace.capabilities.set({})
      await this.router.navigate(['/projects'])
    } catch (error) { this.error.set(error instanceof Error ? error.message : 'Sign-out failed.') }
  }

  private async loadWorkspace(): Promise<void> {
    try { await this.workspace.load() }
    catch (error) { this.error.set(error instanceof Error ? error.message : 'Projects could not be loaded.') }
  }
}
