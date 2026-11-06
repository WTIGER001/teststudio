import { ChangeDetectionStrategy, Component, input } from '@angular/core'
import { FormControl, ReactiveFormsModule } from '@angular/forms'
import { HelpAnchorDirective } from '../../features/help/help-anchor.directive'

@Component({
  selector: 'ts-probe-url-field', standalone: true, imports: [ReactiveFormsModule, HelpAnchorDirective], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<div class="field"><label [for]="id()">Website URL</label><input [id]="id()" tsHelpAnchor="probe-url" type="url" [formControl]="control()" [readOnly]="readonly()" placeholder="https://example.com" autocomplete="url" [attr.aria-describedby]="id() + '-help'"><small [id]="id() + '-help'">Paste the full address, including the path. HTTPS uses port 443 automatically; no port is needed.</small></div>`
})
export class ProbeURLFieldComponent {
  readonly control = input.required<FormControl<string>>()
  readonly id = input('probe-url')
  readonly readonly = input(false)
}
