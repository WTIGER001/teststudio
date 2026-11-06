import { ChangeDetectionStrategy, Component, input } from '@angular/core'
@Component({
 selector:'ts-schedule-review',standalone:true,changeDetection:ChangeDetectionStrategy.OnPush,
 template:`<dl><dt>Schedule</dt><dd>{{ name() }}</dd><dt>Suite</dt><dd>{{ suite() }}</dd><dt>Target</dt><dd>{{ target() }}</dd><dt>Cadence</dt><dd>{{ cadence() }}</dd><dt>Tests</dt><dd>{{ tests() || 'All eligible tests and dependencies' }}</dd><dt>Release</dt><dd>{{ release() || 'Follow the active release at each firing' }}</dd><dt>Run options</dt><dd>{{ options() }}</dd></dl>`,
 styles:`dt{font-weight:600;}dd{margin:.25rem 0 1rem;overflow-wrap:anywhere;}`
})
export class ScheduleReviewComponent { readonly name=input.required<string>();readonly suite=input.required<string>();readonly target=input.required<string>();readonly cadence=input.required<string>();readonly tests=input('');readonly release=input('');readonly options=input(''); }
