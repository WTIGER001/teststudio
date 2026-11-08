import { ChangeDetectionStrategy, Component, computed, effect, input, signal } from '@angular/core'

@Component({
  selector: 'teststudio-status-logo', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `@if (!failed()) { <img [src]="source()" [style.width.px]="sizedHeight() && aspect() ? sizedHeight() * aspect() : null" [style.max-width]="sizedHeight() ? '100%' : 'min(var(--ts-status-logo-width,320px),100%)'" [style.max-height]="sizedHeight() ? sizedHeight() + 'px' : 'var(--ts-status-logo-height,128px)'" (load)="loaded($event)" [alt]="description()" referrerpolicy="no-referrer" (error)="failed.set(true)"> } @else { <p role="status">Logo unavailable</p> }`,
  styles: `:host {display:flex;justify-content:center;align-items:center;min-width:0;} img {display:block;max-width:min(var(--ts-status-logo-width,320px),100%);max-height:var(--ts-status-logo-height,128px);height:auto;object-fit:contain;} p {margin:0;font-size:.8rem;}`
})
export class TestStudioStatusLogoComponent {
  readonly height = input(0); readonly aspect = signal(0); readonly sizedHeight = computed(() => Number.isInteger(this.height()) && this.height() > 0 && this.height() <= 1024 ? this.height() : 0)
  readonly source = input.required<string>()
  readonly description = input('Logo')
  readonly failed = signal(false)
  loaded(event: Event): void { const image = event.target as HTMLImageElement; this.aspect.set(image.naturalHeight ? image.naturalWidth / image.naturalHeight : 0) }
  constructor() { effect(() => { this.source(); this.aspect.set(0); this.failed.set(false) }) }
}
