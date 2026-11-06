import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component } from '@angular/core'
import { buildInfo } from './generated/build-info'

@Component({
  selector: 'ts-build-info', standalone: true, imports: [DatePipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section aria-label="Application build information">
      <strong>TestStudio <span>{{ info.displayVersion }}</span></strong>
      <span class="built">Built <time [attr.datetime]="info.builtAt">{{ info.builtAt | date:"yyyy-MM-dd HH:mm 'UTC'":'UTC' }}</time></span>
      <details><summary>Build <code>{{ info.buildId }}</code></summary><dl>
        <dt>Semantic version</dt><dd>{{ info.version }}</dd>
        <dt>Source commit</dt><dd><code>{{ info.commit }}</code></dd>
        <dt>Source state</dt><dd>{{ info.modified ? 'Includes working changes' : 'Committed source' }}</dd>
        <dt>Source fingerprint</dt><dd><code>{{ info.sourceHash }}</code></dd>
      </dl></details>
    </section>
  `,
  styles: [`:host { display: block; min-width: 0; } section { display: grid; gap: .4rem; font-size: 11px; } strong { color: var(--text); font-weight: 600; } strong span { margin-left: .35rem; } .built { font-size: 10px; } summary { cursor: pointer; padding: .25rem 0; } code { font-size: 10px; overflow-wrap: anywhere; } dl { margin: .5rem 0 0; } dt { font-weight: 600; margin-top: .5rem; } dd { margin: .2rem 0 0; overflow-wrap: anywhere; }`]
})
export class BuildInfoComponent { readonly info = buildInfo }
