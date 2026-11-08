import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { EmbeddedRecorderService } from './embedded-recorder.service'

@Component({
  selector: 'ts-embedded-recorder', standalone: true, imports: [FormsModule], providers: [EmbeddedRecorderService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section aria-label="Live recording browser">
      <p>Use the website below as you normally would. To add a check, choose <strong>Check something appears</strong>, then click the item you expect to see.</p>
      <div class="row-actions wrap-actions"><button type="button" class="button" [class.primary]="!service.checking()" [class.secondary]="service.checking()" [disabled]="!service.view() || service.uncertain()" (click)="service.send({ action: 'assert-visible' })">Check something appears</button>@if (service.checking()) { <button type="button" class="button secondary" (click)="service.send({ action: 'record' })">Continue recording</button> }<button type="button" class="button secondary" [disabled]="service.pending()" (click)="service.refresh()">Refresh browser view</button></div>
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }} Review the captured steps, then use Refresh browser view to reconnect. Actions are not repeated automatically.</p> }
      @if (service.checking()) { <p role="status" class="notice">Click the item that should appear. This adds a visibility check.</p> }
      @if (service.view(); as view) {
        <p class="small muted browser-address">{{ view.url }}</p>
        <img class="browser-view" [src]="'data:image/jpeg;base64,' + view.image" [width]="view.width" [height]="view.height" tabindex="0" role="group" aria-label="Recording browser" aria-describedby="recording-browser-keyboard" draggable="false" (click)="service.point($event)" (keydown)="service.key($event)" (paste)="service.paste($event)">
        <p id="recording-browser-keyboard" class="small muted">Click inside the browser to type. Tab moves to TestStudio controls. Use Next website field below to move within the website.</p>
        <div class="row-actions wrap-actions"><button type="button" class="button secondary" (click)="service.send({ action: 'scroll', y: -500 })">Scroll up</button><button type="button" class="button secondary" (click)="service.send({ action: 'scroll', y: 500 })">Scroll down</button><button type="button" class="button secondary" (click)="service.send({ action: 'key', key: 'Tab' })">Next website field</button><button type="button" class="button secondary" (click)="service.send({ action: 'key', key: 'Enter' })">Press Enter on website</button></div>
        <details><summary>Keyboard entry</summary><div class="field"><label for="recording-text">Text for the selected website field</label><input id="recording-text" maxlength="2048" [ngModel]="service.text()" (ngModelChange)="service.text.set($event)"><button type="button" class="button secondary" [disabled]="!service.text()" (click)="service.enterText()">Enter text on website</button></div></details>
        <h4>Captured steps</h4><ol aria-label="Captured steps">@for (step of view.steps; track $index) { <li>{{ step.description }}</li> }</ol>
        @if (!view.steps.length) { <p class="muted">Your steps will appear here as you use the website.</p> }
      } @else { <p role="status">Connecting to the recording browser…</p> }
    </section>
  `,
  styles: [`.browser-view { display: block; width: 100%; height: auto; border: 2px solid var(--border); border-radius: .5rem; cursor: pointer; } .browser-view:focus-visible { outline: 3px solid var(--accent); outline-offset: 3px; } .browser-address { overflow-wrap: anywhere; } .row-actions { margin: 1rem 0; } li { margin: .5rem 0; } details { margin: 1rem 0; }`]
})
export class EmbeddedRecorderComponent { readonly service = inject(EmbeddedRecorderService) }
