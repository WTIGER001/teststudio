import { ChangeDetectionStrategy, Component, forwardRef, input, output } from '@angular/core'
import type { CatalogCategory } from './catalog-browse.utils'

@Component({
  selector: 'ts-catalog-category-nodes', standalone: true, imports: [forwardRef(() => CatalogCategoryNodesComponent)], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<ul>@for (node of nodes(); track node.path) { <li>@if (node.children.length) { <details open><summary>{{ node.label }} <span class="count-label">{{ node.count }}</span></summary><button class="button small subtle category-button" type="button" [attr.aria-current]="selected() === node.path ? 'true' : null" (click)="choose.emit(node.path)">View all in {{ node.label }}</button><ts-catalog-category-nodes [nodes]="node.children" [selected]="selected()" (choose)="choose.emit($event)" /></details> } @else { <button class="button small subtle category-button" type="button" [attr.aria-current]="selected() === node.path ? 'true' : null" (click)="choose.emit(node.path)">{{ node.label }} <span class="count-label">{{ node.count }}</span></button> }</li> }</ul>`,
  styles: [`:host { display: block; min-width: 0; } ul { list-style: none; margin: 0; padding: 0 0 0 12px; } li { margin: 4px 0; min-width: 0; } summary { cursor: pointer; padding: 8px 0; overflow-wrap: anywhere; } .category-button { text-align: left; white-space: normal; overflow-wrap: anywhere; max-width: 100%; } [aria-current=true] { background: var(--accent-soft); color: var(--accent); outline: 1px solid var(--accent); }`]
})
export class CatalogCategoryNodesComponent {
  readonly nodes = input.required<CatalogCategory[]>()
  readonly selected = input('')
  readonly choose = output<string>()
}
