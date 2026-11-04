import { ChangeDetectionStrategy, Component, inject, input, output } from '@angular/core'
import type { RecordValue } from '../../core/api.service'
import { ArtifactViewService } from './artifact-view.service'
import type { Artifact } from './run.models'

@Component({
  selector: 'ts-artifact-actions', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="row-actions">
      @if (service.retention.artifactState(record().data) === 'available') {
        @if (service.workspace.can('evidence.read')) { <button class="button small subtle" type="button" [disabled]="service.loading()" (click)="open.emit({ artifact: record().data, action: 'view', event: $event })">View</button> }
        @if (service.workspace.can('evidence.download')) { <button class="button small subtle" type="button" [disabled]="service.loading()" (click)="open.emit({ artifact: record().data, action: 'download', event: $event })">Download</button> }
      }
      @if (service.workspace.can('evidence.delete') && record().data.state !== 'deleted' && record().data.state !== 'expired') {
        <button class="button small subtle" type="button" [disabled]="!service.retention.view()?.finished || service.retention.busy() || service.retention.pinned() || record().data.state === 'pending' || record().data.state === 'deleting' || record().data.state === 'delete-pending'" (click)="service.retention.beginDelete(record())">Delete</button>
      }
    </div>
  `
})
export class ArtifactActionsComponent {
  readonly service = inject(ArtifactViewService)
  readonly record = input.required<RecordValue<Artifact>>()
  readonly open = output<{ artifact: Artifact; action: 'view' | 'download'; event: Event }>()
}
