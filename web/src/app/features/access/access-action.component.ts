import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { AccessActionService } from './access-action.service'

@Component({
  selector: 'ts-access-action', standalone: true, imports: [FormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (action.message()) { <p class="notice success" role="status">{{ action.message() }}</p> }
    @if (action.pending(); as pending) {
      <section class="panel" aria-labelledby="access-action-heading">
        <h2 id="access-action-heading" tabindex="-1">{{ pending.title }}</h2><p class="small wrap-anywhere">{{ pending.target }}</p><p class="muted small impact">{{ pending.impact }}</p>
        <form (ngSubmit)="action.submit()"><div class="field"><label for="access-action-reason">Reason</label><textarea id="access-action-reason" name="reason" [ngModel]="action.reason()" (ngModelChange)="action.reason.set($event)" required maxlength="1024" [disabled]="action.busy()"></textarea></div>
          @if (action.error()) { <p class="notice error" role="alert">{{ action.error() }}</p> }
          @if (action.conflict()) { <p class="small">The record changed or the action could not be confirmed. Cancel this action, refresh the record, and open the action again.</p> }
          <div class="row-actions wrap-actions"><button type="submit" class="button primary" [disabled]="action.busy() || action.conflict()">Confirm action</button><button type="button" class="button secondary" [disabled]="action.busy()" (click)="action.close()">Cancel action</button></div>
        </form>
      </section>
    }
  `,
  styles: `.impact { margin: 12px 0 18px; }`
})
export class AccessActionComponent { readonly action = inject(AccessActionService) }
