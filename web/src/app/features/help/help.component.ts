import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core'
import { HelpService } from './help.service'

@Component({
  selector: 'ts-help', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <button type="button" class="button secondary small" aria-controls="page-help" aria-haspopup="dialog" [attr.aria-expanded]="opened()" (click)="open(drawer, content)"><span class="help-symbol" aria-hidden="true">?</span> Help</button>
    <dialog #drawer id="page-help" class="help-drawer" aria-labelledby="help-title" aria-describedby="help-description" (close)="opened.set(false)" (click)="dismissBackdrop($event, drawer)">
      <header class="help-header">
        <div><span class="eyebrow">Page guide</span><h2 id="help-title">{{ help.page().title }} help</h2><p id="help-description" class="small muted">Help for the page you are viewing.</p></div>
        <button #closeHelp type="button" class="button secondary small" autofocus (click)="drawer.close()" (keydown.shift.tab)="wrapFocus($event, content)" aria-label="Close help">Close <span aria-hidden="true">×</span></button>
      </header>
      <nav class="help-sections" aria-label="Help sections">
        <button type="button" (click)="jumpTo(introduction)">Introduction</button>
        <button type="button" (click)="jumpTo(concepts)">Concepts</button>
        <button type="button" (click)="jumpTo(tutorials)">Tutorials</button>
      </nav>
      <div #content class="help-content" role="region" aria-label="Help content" tabindex="0" (keydown.tab)="returnToHeader($event, closeHelp)">
        <section aria-labelledby="help-introduction"><h3 #introduction id="help-introduction" tabindex="-1">Introduction</h3>@for (paragraph of help.page().introduction; track $index) { <p>{{ paragraph }}</p> }</section>
        <section aria-labelledby="help-concepts"><h3 #concepts id="help-concepts" tabindex="-1">Concepts</h3><dl>@for (concept of help.page().concepts; track concept.title) { <dt>{{ concept.title }}</dt><dd>{{ concept.description }}</dd> }</dl></section>
        <section aria-labelledby="help-tutorials"><h3 #tutorials id="help-tutorials" tabindex="-1">Tutorials</h3>@for (tutorial of help.page().tutorials; track tutorial.title) { <article class="help-tutorial"><h4>{{ tutorial.title }}</h4><ol>@for (step of tutorial.steps; track $index) { <li>{{ step }}</li> }</ol></article> }</section>
      </div>
    </dialog>
  `,
  styleUrl: './help.component.css'
})
export class HelpComponent {
  readonly help = inject(HelpService)
  readonly opened = signal(false)

  open(drawer: HTMLDialogElement, content: HTMLElement): void {
    drawer.showModal(); content.scrollTop = 0; this.opened.set(true)
  }

  jumpTo(section: HTMLElement): void { section.scrollIntoView({ block: 'start' }); section.focus({ preventScroll: true }) }

  wrapFocus(event: Event, target: HTMLElement): void { event.preventDefault(); target.focus({ preventScroll: true }) }

  returnToHeader(event: Event, target: HTMLElement): void {
    if (event instanceof KeyboardEvent && !event.shiftKey) this.wrapFocus(event, target)
  }

  dismissBackdrop(event: MouseEvent, drawer: HTMLDialogElement): void {
    if (event.target !== drawer) return
    const bounds = drawer.getBoundingClientRect()
    if (event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom) drawer.close()
  }
}
