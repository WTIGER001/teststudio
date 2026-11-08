import { DatePipe, isPlatformBrowser } from '@angular/common'
import { ChangeDetectionStrategy, Component, computed, effect, inject, input, output, PLATFORM_ID, signal, TemplateRef } from '@angular/core'
import { StatusLoadError, StatusReport, StatusSelection, StatusSource, statusPeriods } from './models'
import { TestStudioStatusBarComponent } from './status-bar.component'
import { TestStudioStatusSummaryComponent } from './status-summary.component'
import { TestStudioStatusClient } from './status-client.service'
import { TestStudioStatusLogoComponent } from './status-logo.component'

@Component({
  selector: 'teststudio-status-page', standalone: true, imports: [DatePipe, TestStudioStatusBarComponent, TestStudioStatusSummaryComponent, TestStudioStatusLogoComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="status-page" [style.background-color]="background()" [style.color]="foreground()" aria-label="Status page">
      @if (current(); as value) {
        @if (value.logo) { <teststudio-status-logo class="page-logo" [source]="value.logo" [description]="value.logoAlt || value.title + ' logo'" /> }
        <header><h1>{{ value.title }}</h1><label class="range">{{ periodLabel() }}<select [attr.aria-label]="periodLabel()" [value]="value.window" (change)="selectPeriod($any($event.target).value)">@for (period of periods; track period.id) { <option [value]="period.id" [selected]="period.id === value.window">{{ period.label }}</option> }</select></label></header>
        <p class="as-of">Updated {{ value.asOf | date:'medium':timezone():locale() }}</p>
        @for (row of value.probes; track $index) {
          <article class="status-row" [attr.aria-label]="row.title">
            <teststudio-status-bar [row]="row" [locale]="locale()" [timezone]="timezone()" [titleTemplate]="titleTemplate()" [descriptionTemplate]="descriptionTemplate()" [detailTemplate]="detailTemplate()" (bucketSelected)="bucketSelected.emit($event)" (drilldownRequested)="drilldownRequested.emit($event)" />
            @if (showSummaries()) { <teststudio-status-summary [row]="row" [locale]="locale()" /> }
          </article>
        }
        <footer><span class="legend up-key">Passed / available</span><span class="legend down-key">Unsuccessful / unavailable</span><span class="legend unknown-key">Unknown</span><p>Availability estimates monitored time. Pass rates count completed outcomes and exclude skipped and unknown results. Missing or stale monitoring is unknown.</p></footer>
      } @else { <h1>Service status</h1>@if (loading()) { <p role="status">Loading status…</p> } @else if (error(); as value) { <p role="alert">{{ value.message }}</p> } @else { <p>No report supplied.</p> } }
    </section>`,
  styleUrl: './status-page.css'
})
export class TestStudioStatusPageComponent {
  readonly report = input<StatusReport | null>(null)
  readonly source = input<StatusSource | null>(null)
  readonly period = input('')
  readonly periodLabel = input('Report period')
  readonly locale = input('en-US')
  readonly timezone = input<string | undefined>(undefined)
  readonly showSummaries = input(true)
  readonly useReportBackground = input(false)
  readonly titleTemplate = input<TemplateRef<unknown> | null>(null)
  readonly descriptionTemplate = input<TemplateRef<unknown> | null>(null)
  readonly detailTemplate = input<TemplateRef<unknown> | null>(null)
  readonly windowChanged = output<string>()
  readonly bucketSelected = output<StatusSelection>()
  readonly drilldownRequested = output<StatusSelection>()
  readonly loadFailed = output<StatusLoadError>()
  readonly loadingChanged = output<boolean>()
  private readonly client = inject(TestStudioStatusClient)
  private readonly browser = isPlatformBrowser(inject(PLATFORM_ID))
  private readonly remote = signal<StatusReport | null>(null)
  private readonly requestedPeriod = signal('')
  readonly loading = signal(false)
  readonly error = signal<StatusLoadError | null>(null)
  readonly current = computed(() => this.source() ? this.remote() : this.report())
  readonly periods = statusPeriods
  readonly background = computed(() => this.useReportBackground() ? this.current()?.background || null : null)
  readonly foreground = computed(() => {
    const background = this.background()
    if (!background || !/^#[a-fA-F0-9]{6}$/.test(background)) return null
    const channels = [1, 3, 5].map(index => parseInt(background.slice(index, index + 2), 16) / 255).map(value => value <= .04045 ? value / 12.92 : ((value + .055) / 1.055) ** 2.4)
    return channels[0] * .2126 + channels[1] * .7152 + channels[2] * .0722 > .179 ? '#111827' : '#ffffff'
  })
  constructor() {
    effect(onCleanup => {
      const source = this.source(), period = this.period() || this.requestedPeriod()
      this.remote.set(null); this.error.set(null)
      if (!source || !this.browser) return
      const subscription = this.client.watch(source, period).subscribe(state => { this.remote.set(state.report); this.loading.set(state.loading); this.loadingChanged.emit(state.loading); this.error.set(state.error); if (state.error) this.loadFailed.emit(state.error) })
      onCleanup(() => subscription.unsubscribe())
    })
  }
  selectPeriod(period: string): void { this.requestedPeriod.set(period); this.windowChanged.emit(period) }
}
