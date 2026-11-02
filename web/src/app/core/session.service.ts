import { computed, inject, Injectable, signal } from '@angular/core'
import { ApiError, ApiService } from './api.service'

export interface User { id: string; name: string; email?: string }
interface SessionResponse { user: User | null; session: { csrfToken: string; expiresAt: string } | null }
interface ProviderList { providers: { id: string; name: string }[]; localEnabled: boolean }

@Injectable({ providedIn: 'root' })
export class SessionService {
  private readonly api = inject(ApiService)
  readonly user = signal<User | null>(null)
  readonly signedIn = computed(() => this.user() !== null)
  readonly loading = signal(true)
  readonly error = signal('')
  readonly providers = signal<ProviderList>({ providers: [], localEnabled: false })

  constructor() {
    this.api.onUnauthenticated = () => {
      this.user.set(null)
      this.api.csrf.set('')
      this.error.set('Your session has expired. Sign in again to continue.')
    }
  }

  async initialize(): Promise<void> {
    this.loading.set(true)
    this.error.set('')
    try {
      this.providers.set(await this.api.request<ProviderList>('/auth/providers'))
      await this.refresh()
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) this.error.set(error instanceof Error ? error.message : 'Sign-in could not be loaded.')
    } finally { this.loading.set(false) }
  }

  async refresh(): Promise<void> {
    const response = await this.api.request<SessionResponse>('/auth/session')
    this.api.csrf.set(response.session?.csrfToken ?? '')
    this.user.set(response.user)
  }

  async signInLocally(): Promise<void> {
    await this.api.request('/auth/local', 'POST', {})
    await this.refresh()
  }

  async signOut(): Promise<void> {
    await this.api.request('/auth/logout', 'POST', {})
    this.api.csrf.set('')
    this.user.set(null)
  }
}
