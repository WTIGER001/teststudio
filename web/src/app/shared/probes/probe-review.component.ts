import { ChangeDetectionStrategy, Component, input } from '@angular/core'

@Component({
  selector: 'ts-probe-review', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<dl><dt>Test</dt><dd>{{ title() }} · {{ suite() }}</dd><dt>Request</dt><dd class="wrap-anywhere">{{ method() }} {{ url() }}</dd><dt>Accepted status codes</dt><dd>{{ statuses() }}</dd><dt>Timeout ceiling</dt><dd>{{ timeout() }} seconds</dd></dl>`,
  styles: `dt { font-weight: 600; } dd { margin: .25rem 0 1rem; }`
})
export class ProbeReviewComponent {
  readonly title = input.required<string>()
  readonly suite = input.required<string>()
  readonly method = input('GET')
  readonly url = input.required<string>()
  readonly statuses = input('200')
  readonly timeout = input(10)
}
