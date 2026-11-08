import { DatePipe, NgTemplateOutlet } from '@angular/common'
import { ChangeDetectionStrategy, Component, computed, input, output, signal, TemplateRef } from '@angular/core'
import { StatusBucket, StatusRow, StatusSelection, statusLabel } from './models'

@Component({
  selector: 'teststudio-status-bar', standalone: true, imports: [DatePipe, NgTemplateOutlet], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section [attr.aria-label]="row().title">
      @if (showHeading()) { @if (titleTemplate(); as template) { <ng-container [ngTemplateOutlet]="template" [ngTemplateOutletContext]="{ $implicit: row() }" /> } @else { <h2>{{ row().title }}</h2> } }
      @if (descriptionTemplate(); as template) { <ng-container [ngTemplateOutlet]="template" [ngTemplateOutletContext]="{ $implicit: row() }" /> } @else if (row().description) { <p class="description">{{ row().description }}</p> }
      <p class="current">{{ label(row().state) }}</p>
      <div class="pills" aria-label="Check history">
        @for (bucket of buckets(); track bucket.since) {
          <button class="pill" type="button" [class.passed]="bucket.state === 'passed'" [class.failed]="bucket.state === 'failed'" [class.timeout]="bucket.state === 'timed-out'" [class.flaky]="bucket.state === 'flaky'" [class.gaps]="hasGap(bucket)" [class.partial]="bucket.partial" [attr.aria-label]="details(bucket)" (click)="select(bucket)">
            <span class="tooltip" role="tooltip">{{ label(bucket.state) }} · {{ bucket.since | date:'medium':timezone():locale() }} – {{ bucket.until | date:'medium':timezone():locale() }}<br>{{ details(bucket, false) }}</span>
          </button>
        } @empty { <span>No observations in this period</span> }
      </div>
      @if (selected(); as bucket) {
        <div class="selected" role="status">@if (detailTemplate(); as template) { <ng-container [ngTemplateOutlet]="template" [ngTemplateOutletContext]="{ $implicit: bucket, row: row() }" /> } @else { {{ details(bucket) }} }
          <button type="button" (click)="drilldownRequested.emit({ row: row(), bucket })">Inspect period</button>
        </div>
      }
    </section>`,
  styleUrl: './status-bar.css'
})
export class TestStudioStatusBarComponent {
  readonly row = input.required<StatusRow>()
  readonly showHeading = input(true)
  readonly locale = input('en-US')
  readonly timezone = input<string | undefined>(undefined)
  readonly titleTemplate = input<TemplateRef<unknown> | null>(null)
  readonly descriptionTemplate = input<TemplateRef<unknown> | null>(null)
  readonly detailTemplate = input<TemplateRef<unknown> | null>(null)
  readonly bucketSelected = output<StatusSelection>()
  readonly drilldownRequested = output<StatusSelection>()
  private readonly selection = signal<{ row: StatusRow; bucket: StatusBucket } | null>(null)
  readonly selected = computed(() => this.selection()?.row === this.row() ? this.selection()!.bucket : null)
  readonly buckets = computed<StatusBucket[]>(() => this.row().buckets ?? this.row().history.map(point => ({ since: point.at, until: point.at, state: point.state, partial: false, totals: { counts: { passed: 0, failed: 0, timedOut: 0, flaky: 0, skipped: 0, unknown: 0 }, upSeconds: 0, downSeconds: 0, unknownSeconds: 0, durationSumMs: 0, durationSampleCount: 0, censoredCount: 0, durationUnavailableCount: 0 }, availability: null, passRate: null })))
  label(state: string): string { return statusLabel(state, this.row().metric) }
  hasGap(bucket: StatusBucket): boolean { return bucket.availability ? bucket.availability.unknownSeconds > 0 && bucket.availability.coveragePercent > 0 : bucket.totals.counts.unknown > 0 && bucket.totals.counts.unknown < Object.values(bucket.totals.counts).reduce((a, b) => a + b, 0) }
  details(bucket: StatusBucket, boundaries = true): string {
    const counts = bucket.totals.counts
    const outcomes = `${counts.passed} passed · ${counts.failed} failed · ${counts.timedOut} timed out · ${counts.flaky} flaky · ${counts.skipped} skipped · ${counts.unknown} unknown`
    const value = bucket.availability ? `${bucket.availability.downSeconds.toFixed(0)} seconds unavailable · ${bucket.availability.unknownSeconds.toFixed(0)} seconds unknown · ${outcomes}` : outcomes
    return `${boundaries ? this.label(bucket.state) + ' · ' + bucket.since + ' – ' + bucket.until + ' · ' : ''}${value}${bucket.partial ? ' · Partial period' : ''}`
  }
  select(bucket: StatusBucket): void { this.selection.set({ row: this.row(), bucket }); this.bucketSelected.emit({ row: this.row(), bucket }) }
}
