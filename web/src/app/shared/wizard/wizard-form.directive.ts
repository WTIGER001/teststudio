import { afterNextRender, Directive, ElementRef, inject, Injector } from '@angular/core'

@Directive({ selector: '[tsWizardForm]', standalone: true, exportAs: 'tsWizardForm' })
export class WizardFormDirective {
  private readonly injector = inject(Injector)
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef)
  reveal(): void {
    afterNextRender(() => {
      const fields = this.host.nativeElement.querySelectorAll<HTMLElement>('[aria-invalid="true"], input.ng-invalid, select.ng-invalid, textarea.ng-invalid')
      for (const field of fields) {
        let ancestor: HTMLElement | null = field.parentElement
        while (ancestor && ancestor !== this.host.nativeElement) { if (ancestor instanceof HTMLDetailsElement) ancestor.open = true; ancestor = ancestor.parentElement }
      }
      fields[0]?.focus()
    }, { injector: this.injector })
  }
}
