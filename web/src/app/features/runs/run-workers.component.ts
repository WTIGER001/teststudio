import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RunWorkersService } from './run-workers.service'

@Component({
  selector: 'ts-run-workers', standalone: true, imports: [DatePipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.records().length || service.error()) {
      <section class="panel" aria-labelledby="worker-resources-title"><div class="section-top"><h2 id="worker-resources-title">Worker resources</h2><button class="button secondary" type="button" [disabled]="service.loading()" (click)="service.load()">Refresh workers</button></div>
        <p class="small muted">Observations are saved before resource removal. Recovery retains available diagnostics and keeps the original run outcome.</p>
        @if (service.error()) { <p class="notice warning" role="alert">{{ service.error() }}</p> }
        @if (service.pending()) { <p class="notice" role="status">Worker cleanup or evidence recovery is pending. This page checks again every five seconds.</p> }
        <div class="table-scroll"><table class="worker-table"><caption class="sr-only">Worker execution and recovery state on this page</caption><thead><tr><th scope="col">Test / attempt</th><th scope="col">Resources</th><th scope="col">Observation</th><th scope="col">Details</th></tr></thead><tbody>
          @for (item of service.records(); track item.id) {
            <tr><td><code>{{ item.data.reference.testId }}</code><span class="table-subtitle muted">Attempt {{ item.data.reference.attempt }} · {{ item.data.reference.profile }}</span></td>
              <td><span class="badge" [class.success]="item.data.state === 'cleaned'" [class.warning]="item.data.state !== 'cleaned'">{{ item.data.state }}</span><span class="table-subtitle muted">{{ item.data.cleanupAttempts }} cleanup {{ item.data.cleanupAttempts === 1 ? 'attempt' : 'attempts' }}</span></td>
              <td>{{ item.data.resultSaved ? 'Checkpoint saved' : 'Awaiting checkpoint' }}@if (item.data.recoveredResult; as result) { <span class="table-subtitle">Recovered observation: {{ result.observedState || result.state }}</span> }@if (item.data.recovered) { <span class="table-subtitle muted">Recovered after interruption or incomplete capture.</span> }</td>
              <td class="wrap-anywhere">@if (item.data.error) { <p>{{ item.data.error }}</p> }@if (item.data.recoveredResult?.error) { <p class="small muted">{{ item.data.recoveredResult.error }}</p> }@if (item.data.evidenceAbandoned) { <p class="small">Capture ended with gaps. Available files and failure records remain readable.</p> }@for (error of item.data.evidenceErrors; track $index) { <p class="small muted">{{ error }}</p> }
                <span class="small muted">Updated {{ item.data.updatedAt | date:'medium' }}</span>
                <details><summary class="small">Resource references</summary><p class="small wrap-anywhere"><code>{{ item.data.reference.container }}</code></p><p class="small wrap-anywhere"><code>{{ item.data.reference.volume }}</code></p></details>
              </td></tr>
          }
        </tbody></table></div>
        @if (service.canGoBack() || service.nextCursor()) { <div class="row-actions"><button class="button secondary" type="button" [disabled]="!service.canGoBack() || service.loading()" (click)="service.back()">Previous workers</button><button class="button secondary" type="button" [disabled]="!service.nextCursor() || service.loading()" (click)="service.next()">Next workers</button></div> }
      </section>
    }
  `
})
export class RunWorkersComponent { readonly service = inject(RunWorkersService) }
