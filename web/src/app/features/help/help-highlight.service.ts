import { DOCUMENT } from '@angular/common'
import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { NavigationEnd, Router } from '@angular/router'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { filter } from 'rxjs'

@Injectable({ providedIn: 'root' })
export class HelpHighlightService {
  private readonly document = inject(DOCUMENT)
  private readonly anchors = new Map<string, HTMLElement>()
  private current?: HTMLElement
  readonly active = signal('')
  readonly message = signal('')
  constructor() {
    inject(Router).events.pipe(filter(e => e instanceof NavigationEnd), takeUntilDestroyed(inject(DestroyRef))).subscribe(() => this.clear())
  }
  register(id: string, element: HTMLElement): () => void {
    if (!element.closest('ts-help-example')) this.anchors.set(id, element)
    return () => { if (this.anchors.get(id) === element) this.anchors.delete(id); if (this.current === element) this.clear() }
  }
  locate(id: string): HTMLElement | undefined {
    const element = this.anchors.get(id)
    if (!element?.isConnected || !element.getClientRects().length || element.matches(':disabled') || element.closest('[inert], dialog')) {
      this.message.set('This control is not available on the current step. Close help and choose the relevant step; your entered values stay in place.')
      return undefined
    }
    return element
  }
  show(id: string, element: HTMLElement): void {
    this.clear(); this.current = element; this.active.set(id)
    element.style.outline = '3px solid var(--accent)'; element.style.outlineOffset = '4px'
    element.scrollIntoView({ block: 'center', behavior: 'instant' }); element.focus()
  }
  clear(): void {
    if (this.current) { this.current.style.removeProperty('outline'); this.current.style.removeProperty('outline-offset') }
    this.current = undefined; this.active.set(''); this.message.set('')
  }
  focusHelp(): void { this.document.querySelector<HTMLButtonElement>('ts-help > button')?.focus() }
}
