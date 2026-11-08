import { DecimalPipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, input } from '@angular/core'
import { monitoredLabel, StatusRow } from './models'

@Component({
  selector: 'teststudio-status-summary', standalone: true, imports: [DecimalPipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <figure [attr.aria-label]="label()">
      @if (row().metric !== 'outcomes') {
        <svg viewBox="0 0 100 100" aria-hidden="true"><circle class="unknown" cx="50" cy="50" r="40" pathLength="100" /><circle class="up" cx="50" cy="50" r="40" pathLength="100" [attr.stroke-dasharray]="arc('up') + ' 100'" /><circle class="down" cx="50" cy="50" r="40" pathLength="100" [attr.stroke-dasharray]="arc('down') + ' 100'" [attr.stroke-dashoffset]="-arc('up')" /></svg>
        <div class="donut-label">@if (row().availability.percent !== null) { <strong>{{ row().availability.percent | number:'1.2-2':locale() }}%</strong><span>availability</span> } @else { <strong>—</strong><span>No data</span> }</div>
        <figcaption>{{ coverage(row().availability.coveragePercent) }} monitored</figcaption>
      } @else {
        <strong class="pass-rate">{{ row().passRate === null || row().passRate === undefined ? '—' : (row().passRate | number:'1.2-2':locale()) + '%' }}</strong><figcaption>pass rate</figcaption>
        <p>{{ row().totals?.counts?.skipped || 0 }} skipped · {{ row().totals?.counts?.unknown || 0 }} unknown</p>
      }
    </figure>`,
  styles: `:host {display:block;} figure {margin:0;position:relative;width:var(--ts-status-summary-width,140px);text-align:center;color:inherit;} svg {width:120px;height:120px;display:block;margin:0 auto;} circle {fill:none;stroke-width:9;transform:rotate(-90deg);transform-origin:50% 50%;} .unknown {stroke:var(--ts-status-unknown,#8993a5);} .up {stroke:var(--ts-status-passed,#16854b);} .down {stroke:var(--ts-status-failed,#dc3545);} .donut-label {position:absolute;top:42px;width:100%;display:grid;font-size:10px;line-height:1.3;} strong {font-size:18px;} figcaption,p {font-size:11px;margin:4px 0;} .pass-rate {display:block;} @media(max-width:450px) {figure {width:108px;} svg {width:104px;height:104px;} .donut-label {top:36px;} strong {font-size:16px;}}`
})
export class TestStudioStatusSummaryComponent {
  readonly row = input.required<StatusRow>()
  readonly locale = input('en-US')
  readonly coverage = monitoredLabel
  label(): string { const row = this.row(); return row.metric === 'outcomes' ? `${row.passRate ?? 'Unavailable'} percent pass rate` : row.availability.percent === null ? 'No monitored data' : `${row.availability.percent.toFixed(2)}% availability; ${this.coverage(row.availability.coveragePercent)} monitored` }
  arc(kind: 'up' | 'down'): number { const a = this.row().availability, total = a.upSeconds + a.downSeconds + a.unknownSeconds; return total ? 100 * (kind === 'up' ? a.upSeconds : a.downSeconds) / total : 0 }
}
