import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core'
import type { PermissionSource } from '../../core/generated/access.types'
import { sourceDetails, sourceTitle } from './permission-explanation.utils'

@Component({
  selector: 'ts-permission-source', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <article class="source-card" [attr.aria-label]="title()">
      <div class="source-heading"><h3>{{ title() }}</h3><div class="row-actions wrap-actions"><span class="badge" [class.warning]="source().effect === 'deny'">{{ source().effect === 'deny' ? 'Deny' : 'Allow' }}</span><span class="badge" [class.warning]="source().status === 'expired'">{{ source().status === 'expired' ? 'Expired · no effect' : 'Active matching source' }}</span></div></div>
      @if (source().restricted) { <p class="small muted">This source applies to the context, but its administrative details require additional read authority.</p> }
      @else { <dl class="source-details">@for (detail of details(); track detail.label) { <dt>{{ detail.label }}</dt><dd class="wrap-anywhere">{{ detail.value }}</dd> }</dl> }
    </article>
  `,
  styles: `.source-card { padding: 20px; border: 1px solid var(--border); border-radius: 10px; } .source-heading { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; justify-content: space-between; margin-bottom: 16px; } h3 { margin: 0; font-size: 1rem; } .source-details { display: grid; grid-template-columns: minmax(130px, 1fr) minmax(0, 3fr); gap: 8px 20px; margin: 0; font-size: .875rem; } dt { color: var(--muted); } dd { margin: 0; } @media (max-width: 640px) { .source-details { grid-template-columns: 1fr; gap: 4px; } dd { margin-bottom: 10px; } .source-card { padding: 14px; } }`
})
export class PermissionSourceComponent { readonly source = input.required<PermissionSource>(); readonly details = computed(() => sourceDetails(this.source())); readonly title = computed(() => sourceTitle(this.source())) }
