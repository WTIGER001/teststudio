import { ChangeDetectionStrategy, Component, effect, input, signal } from '@angular/core'

@Component({
 selector: 'ts-status-logo', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
 template: `@if (!failed()) { <img [src]="source()" [alt]="description()" referrerpolicy="no-referrer" (error)="failed.set(true)"> } @else { <p class="small" role="status">Logo unavailable</p> }`,
 styles: `:host {display:flex;justify-content:center;align-items:center;min-width:0;} img {display:block;max-width:min(320px,100%);max-height:128px;object-fit:contain;} p {margin:0;}`
})
export class StatusLogoComponent {
 readonly source = input.required<string>(); readonly description = input('Logo'); readonly failed = signal(false)
 constructor() { effect(() => { this.source(); this.failed.set(false) }) }
}
