import { Directive, ElementRef, inject, input, OnInit, DestroyRef } from '@angular/core'
import { HelpHighlightService } from './help-highlight.service'

@Directive({ selector: '[tsHelpAnchor]', standalone: true })
export class HelpAnchorDirective implements OnInit {
  readonly tsHelpAnchor = input.required<string>()
  private readonly element = inject(ElementRef<HTMLElement>)
  private readonly service = inject(HelpHighlightService)
  private readonly destroy = inject(DestroyRef)
  ngOnInit(): void { this.destroy.onDestroy(this.service.register(this.tsHelpAnchor(), this.element.nativeElement)) }
}
