import { ChangeDetectionStrategy, Component, ElementRef, input, OnChanges, OnDestroy, SimpleChanges, ViewChild } from '@angular/core'

@Component({
  selector: 'ts-wizard-heading', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<h3 #heading tabindex="-1">{{ title() }}</h3><p class="small muted"><ng-content /></p>`,
  styles: [`:host { display: block; margin-bottom: 1rem; } h3 { font-size: 18px; margin-bottom: .5rem; }`]
})
export class WizardHeadingComponent implements OnChanges, OnDestroy {
  readonly title = input.required<string>()
  readonly step = input.required<number>()
  @ViewChild('heading', { static: true }) heading!: ElementRef<HTMLHeadingElement>
  private disposed = false
  ngOnChanges(changes: SimpleChanges): void { if (changes['step'] && !changes['step'].firstChange) queueMicrotask(() => { if (!this.disposed) this.heading.nativeElement.focus() }) }
  ngOnDestroy(): void { this.disposed = true }
}
