import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core'
import { TestStudioStatusPageComponent } from '@wtiger001/teststudio-angular'
import type { StatusPageReport } from '../../core/generated/access.types'
export { statusWindows, statusTextColor, monitoredLabel } from './status.utils'

@Component({
  selector: 'ts-status-view', standalone: true, imports: [TestStudioStatusPageComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<teststudio-status-page [report]="report()" [useReportBackground]="true" [periodLabel]="periodLabel()" (windowChanged)="windowChanged.emit($event)" />`
})
export class StatusViewComponent {
  readonly report = input.required<StatusPageReport>()
  readonly periodLabel = computed(() => this.report().probes.some(row => row.metric === 'outcomes') ? 'Report period' : 'Availability period')
  readonly windowChanged = output<string>()
}
