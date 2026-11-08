import { DatePipe, DecimalPipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, effect, inject, input, output } from '@angular/core'
import { StatusChecksService } from './status-checks.service'
import { StatusCheck, StatusCheckLoader, StatusCheckRange, StatusSource, statusLabel } from './models'

@Component({
  selector: 'teststudio-status-checks', standalone: true, imports: [DatePipe, DecimalPipe], providers: [StatusChecksService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<section aria-label="Individual checks">
    <h3>Individual checks</h3><p>One row per final logical execution. No aggregation. Newest completion first.</p>
    <p>{{ range().since | date:'medium':timezone():locale() }} – {{ range().until | date:'medium':timezone():locale() }}</p>
    @if (narrowed()) { <button type="button" (click)="resetRange.emit()">Return to full period</button> }
    @if (service.error()) { <p role="alert">{{ service.error() }}</p><button type="button" (click)="service.load()">Retry individual checks</button> }
    <div class="table-scroll" tabindex="0" aria-label="Individual check results"><table><caption>Individual check results</caption><thead><tr><th scope="col">Completed</th><th scope="col">Outcome</th><th scope="col">Duration</th>@if (inspectable()) { <th scope="col">Details</th> }</tr></thead><tbody>
      @for (check of service.items(); track $index) {
        <tr><td><time [attr.datetime]="check.at">{{ check.at | date:'mediumDate':timezone():locale() }}<br>{{ check.at | date:'mediumTime':timezone():locale() }}</time></td><td>{{ label(check.state, 'outcomes') }}</td><td>@if (check.durationMs !== null) { {{ check.durationMs | number:'1.0-2':locale() }} ms @if (check.censored) { <span>(censored)</span> } } @else { Unavailable }</td>@if (inspectable()) { <td>@if (check.runId) { <button type="button" (click)="checkRequested.emit(check)">Inspect check</button> }</td> }</tr>
      } @empty { @if (!service.loading() && !service.error()) { <tr><td [attr.colspan]="inspectable() ? 4 : 3">No retained individual results in this period. Summaries can outlive individual detail.</td></tr> } }
    </tbody></table></div>
    @if (service.loading()) { <p role="status">Loading individual checks…</p> }
    @if (service.partial()) { <p>Some stored results were filtered. Continue to scan older checks.</p> }
    @if (service.cursor()) { <button type="button" [disabled]="service.loading()" (click)="service.load(true)">Load older checks</button> }
  </section>`,
  styles: `:host {display:block;min-width:0;font:inherit;color:inherit;} section {margin-top:16px;} h3 {font-size:1rem;margin:12px 0;} p,table {font-size:.85rem;} button {font:inherit;color:inherit;background:transparent;border:1px solid currentColor;border-radius:var(--ts-status-radius,5px);padding:6px 10px;cursor:pointer;} button:focus-visible,.table-scroll:focus-visible {outline:2px solid currentColor;outline-offset:3px;} .table-scroll {overflow:auto;max-width:100%;margin:12px 0;} table {border-collapse:collapse;width:100%;text-align:left;} caption {text-align:left;padding:8px 0;} th,td {padding:10px 8px;border-bottom:1px solid color-mix(in srgb,currentColor 22%,transparent);vertical-align:top;white-space:nowrap;} button:disabled {opacity:.6;cursor:wait;}`
})
export class TestStudioStatusChecksComponent {
  readonly range = input.required<StatusCheckRange>()
  readonly row = input(0)
  readonly source = input<StatusSource | null>(null)
  readonly loader = input<StatusCheckLoader | null>(null)
  readonly narrowed = input(false)
  readonly inspectable = input(false)
  readonly locale = input('en-US')
  readonly timezone = input<string | undefined>(undefined)
  readonly resetRange = output<void>()
  readonly checkRequested = output<StatusCheck>()
  readonly service = inject(StatusChecksService)
  readonly label = statusLabel
  constructor() { effect(() => { this.service.configure({ ...this.range(), row: this.row(), limit: 50 }, this.source(), this.loader()) }) }
}
