import { NgTemplateOutlet } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject, input, signal } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { EmbeddedRecorderService } from './embedded-recorder.service'

@Component({
  selector: 'ts-embedded-recorder', standalone: true, imports: [FormsModule, NgTemplateOutlet], changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './embedded-recorder.component.html',
  styles: [`
    .browser-view { display: block; width: 100%; height: auto; border: 2px solid var(--border); border-radius: .5rem; cursor: pointer; }
    .browser-view:focus-visible { outline: 3px solid var(--accent); outline-offset: 3px; }
    .browser-address { overflow-wrap: anywhere; } .row-actions { margin: 1rem 0; } li { margin: .5rem 0; } details { margin: 1rem 0; }
    dialog { width: min(96vw, 110rem); max-width: 96vw; max-height: 94vh; resize: both; overflow: auto; padding: 1.25rem; border: 1px solid var(--border); border-radius: .75rem; background: var(--surface); color: var(--text); }
    dialog::backdrop { background: rgb(0 0 0 / .5); } .section-top { gap: 1rem; flex-wrap: wrap; } .check-review { margin: 1rem 0; }
    fieldset { border: 0; padding: 0; min-width: 0; } .form-grid { margin: 1rem 0; }
    @media (max-width: 600px) { dialog { padding: .75rem; width: 96vw; } }
  `]
})
export class EmbeddedRecorderComponent {
  readonly service = inject(EmbeddedRecorderService)
  readonly workspace = input(false)
  readonly expanded = signal(false)
  validCount(): boolean { const count = Number(this.service.count()); return Number.isInteger(count) && count >= (this.service.checkKind() === 'at-least' ? 1 : 0) && count <= 10000 }
  open(dialog: HTMLDialogElement): void { dialog.showModal(); this.expanded.set(true) }
  close(dialog: HTMLDialogElement): void { dialog.close(); this.expanded.set(false) }
}
