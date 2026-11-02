import { DOCUMENT } from '@angular/common'
import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { resolvedTheme, themePreference, type ThemePreference } from './theme.utils'

@Injectable({ providedIn: 'root' })
export class ThemeService {
  private readonly document = inject(DOCUMENT)
  private readonly destroy = inject(DestroyRef)
  private readonly media = matchMedia('(prefers-color-scheme: dark)')
  private readonly systemDark = signal(this.media.matches)
  readonly preference = signal<ThemePreference>(this.read())
  readonly resolved = computed(() => resolvedTheme(this.preference(), this.systemDark()))

  constructor() {
    const update = () => { this.systemDark.set(this.media.matches); this.apply() }
    this.media.addEventListener('change', update)
    this.destroy.onDestroy(() => this.media.removeEventListener('change', update))
    this.apply()
  }

  set(value: string): void {
    this.preference.set(themePreference(value))
    try { localStorage.setItem('teststudio.theme', this.preference()) } catch {}
    this.apply()
  }

  private read(): ThemePreference {
    try { return themePreference(localStorage.getItem('teststudio.theme')) } catch { return 'system' }
  }

  private apply(): void {
    this.document.documentElement.dataset['theme'] = this.resolved()
    this.document.documentElement.style.colorScheme = this.resolved()
  }
}
