import { ChangeDetectionStrategy, Component, input, output } from '@angular/core'
import { RouterLink } from '@angular/router'
import type { NavigationLink } from './navigation.utils'

@Component({
  selector: 'ts-sidebar-links', standalone: true, imports: [RouterLink], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `@for (link of links(); track link.path) { <a [attr.aria-label]="link.label" [routerLink]="link.path" [queryParams]="link.queryParams" [class.active]="link.active" [attr.aria-current]="link.active ? 'page' : null" (click)="navigated.emit()"><span aria-hidden="true">{{ link.icon }}</span>{{ link.label }}</a> }`,
  styles: [`:host { display: grid; gap: 3px; } a { display: flex; align-items: center; gap: 10px; color: var(--muted); border-radius: 7px; padding: 8px 10px; font-size: 12px; font-weight: 600; min-width: 0; } a span { width: 17px; font-size: 18px; line-height: 1; text-align: center; flex: none; } a:hover { background: var(--surface-muted); text-decoration: none; } a.active { color: var(--accent); background: var(--accent-soft); } @media (max-width: 700px) { :host { grid-template-columns: repeat(2, minmax(0, 1fr)); } a { padding: 8px; font-size: 11px; } }`]
})
export class SidebarLinksComponent {
  readonly links = input.required<NavigationLink[]>()
  readonly navigated = output<void>()
}
