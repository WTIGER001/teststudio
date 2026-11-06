import { ChangeDetectionStrategy, Component, input, output } from '@angular/core'

@Component({
  selector: 'ts-wizard-stepper', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<nav [attr.aria-label]="label()" [class.single-row]="singleRow()" [attr.tabindex]="singleRow() ? 0 : null"><ol>@for (name of steps(); track $index; let index = $index) { <li [class.current]="index === current()" [class.complete]="index < current()"><button type="button" [disabled]="disabled() || !interactive() || index > reached()" [attr.aria-current]="index === current() ? 'step' : null" (click)="selected.emit(index)"><span class="step-number" aria-hidden="true">{{ index < current() ? '✓' : index + 1 }}</span><span>{{ name }}</span>@if (index < current()) { <span class="sr-only">Completed</span> }</button></li> }</ol></nav>`,
  styles: [` .single-row { overflow-x: auto; padding-bottom: .25rem; } .single-row ol { flex-wrap: nowrap; } .single-row li { flex: 1 0 max-content; } .single-row button { white-space: nowrap; } :host { display: block; margin: 1.25rem 0; } ol { list-style: none; display: flex; flex-wrap: wrap; gap: .75rem 1.25rem; padding: 0; margin: 0; } li { flex: 1 1 8rem; border-bottom: 2px solid var(--border); padding-bottom: .75rem; } li.current { border-color: var(--accent); } li.complete { border-color: var(--success); } button { background: none; border: 0; padding: 0; color: var(--muted); display: inline-flex; gap: .6rem; align-items: center; text-align: left; font-size: 12px; } button:disabled { opacity: 1; } .current button { color: var(--text); font-weight: 700; } .step-number { display: inline-grid; place-items: center; width: 25px; height: 25px; border-radius: 50%; background: var(--surface-muted); flex: none; } .current .step-number { background: var(--accent); color: var(--surface); } .complete .step-number { color: var(--success); }`]
})
export class WizardStepperComponent {
  readonly steps = input.required<readonly string[]>()
  readonly current = input.required<number>()
  readonly reached = input(0)
  readonly disabled = input(false)
  readonly interactive = input(true)
  readonly label = input('Setup progress')
  readonly singleRow = input(false)
  readonly selected = output<number>()
}
