import { ChangeDetectionStrategy, Component, effect, inject, input, signal } from '@angular/core'
import { ClipboardService } from './clipboard.service'

@Component({
  selector: 'ts-copy-button', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<button type="button" class="button secondary small" [disabled]="pending()" (click)="copy()">Copy {{ name() }}</button><span class="small" role="status">{{ feedback() }}</span>`,
  styles: `:host {display:flex;align-items:center;gap:8px;flex-wrap:wrap;} [role=status] {overflow-wrap:anywhere;}`
})
export class CopyButtonComponent {
  private readonly clipboard = inject(ClipboardService)
  readonly value = input.required<string>(); readonly name = input('text')
  readonly pending = signal(false); readonly feedback = signal('')
  constructor() { effect(() => { this.value(); this.feedback.set('') }) }
  async copy(): Promise<void> {
    if (this.pending()) return
    const value = this.value(); this.pending.set(true); this.feedback.set('')
    const copied = await this.clipboard.copy(value)
    if (value === this.value()) this.feedback.set(copied ? this.name() + ' copied.' : 'Automatic copy is unavailable. Select and copy the visible field.')
    this.pending.set(false)
  }
}
