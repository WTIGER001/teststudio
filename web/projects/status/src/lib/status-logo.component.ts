import { ChangeDetectionStrategy, Component, effect, input, signal } from '@angular/core'

@Component({
  selector: 'teststudio-status-logo', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `@if (!failed()) { <img [src]="source()" [alt]="description()" referrerpolicy="no-referrer" (error)="failed.set(true)"> } @else { <p role="status">Logo unavailable</p> }`,
  styles: `:host {display:flex;justify-content:center;align-items:center;min-width:0;} img {display:block;max-width:min(var(--ts-status-logo-width,320px),100%);max-height:var(--ts-status-logo-height,128px);object-fit:contain;} p {margin:0;font-size:.8rem;}`
})
export class TestStudioStatusLogoComponent {
  readonly source = input.required<string>()
  readonly description = input('Logo')
  readonly failed = signal(false)
  constructor() { effect(() => { this.source(); this.failed.set(false) }) }
}
