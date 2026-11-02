import { afterRenderEffect, ChangeDetectionStrategy, Component, ElementRef, inject, output, viewChild } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RunRetentionService } from './run-retention.service'

@Component({
  selector: 'ts-artifact-deletion', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.deleting(); as item) {
      <section class="artifact-deletion" aria-labelledby="delete-evidence-title"><div class="section-top"><h3 #heading tabindex="-1" id="delete-evidence-title">Delete {{ item.data.filename }}</h3><button class="button subtle" type="button" (click)="service.closeDelete()">Cancel deletion</button></div>
        @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
        <p class="small muted">This removes the retained file. Its manifest and test outcome remain in history.</p>
        <form [formGroup]="service.deleteForm" (ngSubmit)="confirm()"><div class="field"><label for="artifact-delete-reason">Deletion reason</label><textarea id="artifact-delete-reason" formControlName="reason" rows="2" maxlength="500" required></textarea></div><button class="button primary" type="submit" [disabled]="service.deleteForm.invalid || service.busy()">Delete artifact</button></form>
      </section>
    }
  `
})
export class ArtifactDeletionComponent {
  readonly service = inject(RunRetentionService)
  readonly deleted = output<void>()
  private readonly heading = viewChild<ElementRef<HTMLHeadingElement>>('heading')
  constructor() { afterRenderEffect(() => { if (this.service.deleting()) this.heading()?.nativeElement.focus() }) }
  async confirm(): Promise<void> { if (await this.service.deleteArtifact()) this.deleted.emit() }
}
