import { computed, inject, Injectable, signal } from '@angular/core'
import { ApiError, ApiService } from './api.service'
import { AccessClientService } from './access-client.service'
import type { PasswordChange, PasswordLogin } from './generated/access.types'

export interface User { id: string; name: string; email?: string }
interface SessionResponse { user: User | null; session: { csrfToken: string; expiresAt: string; mustChangePassword?: boolean; authenticationMethod?: string } | null }
interface ProviderList { providers: { id: string; name: string }[]; localEnabled: boolean; localAccountsEnabled: boolean }
export class SessionRefreshError extends Error {}

@Injectable({ providedIn: 'root' })
export class SessionService {
  private readonly api = inject(ApiService)
  private readonly client = inject(AccessClientService)
  readonly user = signal<User | null>(null)
  readonly signedIn = computed(() => this.user() !== null)
  readonly mustChangePassword = signal(false)
  readonly authenticationMethod = signal('')
  readonly ready = computed(() => this.signedIn() && !this.mustChangePassword())
  readonly authenticating = signal(false)
  readonly loading = signal(true)
  readonly error = signal('')
  readonly providers = signal<ProviderList>({ providers: [], localEnabled: false, localAccountsEnabled: false })
  onAuthenticationChanged: (() => void) | undefined
  private generation = 0

  constructor() {
    this.api.onUnauthenticated = csrf => {
      if (csrf !== this.api.csrf() || this.authenticating()) return
      this.generation++; this.clear(); this.error.set('Your session has expired. Sign in again to continue.')
    }
  }
  private clear(): void { this.api.csrf.set(''); this.user.set(null); this.mustChangePassword.set(false); this.authenticationMethod.set(''); this.onAuthenticationChanged?.() }
  async initialize(): Promise<void> {
    this.loading.set(true); this.error.set('')
    try { await this.loadProviders(); await this.refresh() }
    catch (error) { if (!(error instanceof ApiError && error.status === 401)) this.error.set(error instanceof Error ? error.message : 'Sign-in could not be loaded.') }
    finally { this.loading.set(false) }
  }
  async loadProviders(): Promise<void> { this.providers.set(await this.api.request<ProviderList>('/auth/providers')) }
  async refresh(): Promise<void> {
    const generation = ++this.generation
    const response = await this.api.request<SessionResponse>('/auth/session')
    if (generation !== this.generation) return
    const changed = this.user()?.id !== response.user?.id || this.mustChangePassword() !== (response.session?.mustChangePassword === true)
    this.api.csrf.set(response.session?.csrfToken ?? ''); this.mustChangePassword.set(response.session?.mustChangePassword === true); this.authenticationMethod.set(response.session?.authenticationMethod ?? ''); this.user.set(response.user)
    if (changed) this.onAuthenticationChanged?.()
  }
  private async authenticate(work: () => Promise<unknown>): Promise<void> {
    if (this.authenticating()) throw new Error('An authentication request is already in progress.')
    this.generation++; this.authenticating.set(true); this.error.set('')
    try {
      try { await work() }
      catch (error) { await this.refresh().catch(() => {}); throw error }
      try { await this.refresh() }
      catch { throw new SessionRefreshError('The request succeeded, but the session could not be refreshed. Refresh the session before continuing.') }
    } finally { this.authenticating.set(false) }
  }
  signInLocally(): Promise<void> { return this.authenticate(() => this.api.request('/auth/local', 'POST', {})) }
  signInWithPassword(body: PasswordLogin): Promise<void> { return this.authenticate(() => this.client.call('signInWithPassword', { body })) }
  changePassword(body: PasswordChange): Promise<void> { return this.authenticate(() => this.client.call('changeLocalPassword', { body })) }
  async signOut(): Promise<void> { await this.authenticate(() => this.api.request('/auth/logout', 'POST', {})); await this.loadProviders() }
}
