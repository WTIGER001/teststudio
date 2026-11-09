import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { BrowserRecorderService } from './browser-recorder.service'

@Component({
  selector: 'ts-recorder-connection', standalone: true, imports: [DatePipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <p>Connect the desktop recorder on the computer where you want to record. It opens a dedicated browser and Playwright Inspector.</p>
    <ol>
      <li><p>Open a terminal in your installed TestStudio recorder directory. Run this command.</p><div class="field"><label for="desktop-recorder-command">Desktop recorder command</label><input id="desktop-recorder-command" readonly [value]="service.command" spellcheck="false" autocomplete="off"></div><button type="button" class="button secondary" (click)="service.copy('command')">Copy desktop command</button></li>
      <li><p>Paste the connection code into the recorder prompt, then press Enter.</p>
        @if (service.pairing(); as pairing) {
          @if (!service.pairingExpired()) {
            <div class="field"><label for="desktop-recording-code">Connection code</label><textarea id="desktop-recording-code" readonly rows="3" spellcheck="false" autocomplete="off" [value]="pairing.id + ':' + pairing.token"></textarea></div>
            <button type="button" class="button secondary" (click)="service.copy('code')">Copy connection code</button>
            <p class="small muted">Expires {{ pairing.expiresAt | date:'shortTime':'UTC' }} UTC. Keep this code private. Paste it only into your desktop recorder prompt.</p>
          } @else { <p class="notice">The connection code expired. Get a new code to connect this recording session.</p> }
        } @else { <p class="small muted">Connection codes are temporary and are not saved by this page. Get a new code after reopening the page.</p> }
        <button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.connect()">Get new connection code</button>
      </li>
    </ol>
    @if (service.copied()) { <p role="status" class="small">{{ service.copied() }}</p> }
    <p class="notice">This page updates when the recorder connects. Closing this page does not stop the recorder. Use Cancel recording to end this session.</p>
    <details><summary>First-time desktop setup</summary><p>Get the trusted recorder archive for your computer from your TestStudio operator. Extract it and open a terminal in that directory. Install Node.js 22 or later, then run:</p><pre>node setup-recorder.mjs</pre><p>Setup installs the locked Playwright dependencies and Chromium. Go is not required. Keep the helper and dependency directories beside the command. The recorder opens its own browser and preserves your existing browser windows.</p><p class="small muted">Linux desktops can need operator-installed browser system libraries. Windows packages and a one-click installer are not available yet.</p></details>
    <details><summary>Advanced connection and recovery</summary><p>The server address must use HTTPS. Numeric loopback HTTP is available for local testing. The recorder does not follow redirects.</p><p>Use the session identifier printed by the command with <code>--resume</code> after an interruption. Recovery retains captured steps and does not open another recording browser. Expired sessions need a new recording.</p><p>Connection credentials remain in private files on the recording computer. The device can submit source for this session only. Saving a test, saving, and running tests remain separate actions.</p></details>
  `,
  styles: `ol { padding-left: 1.5rem; } li { padding: .25rem 0 1rem; } .field { max-width: 46rem; } input, textarea { box-sizing: border-box; width: 100%; font-family: monospace; } textarea { resize: vertical; overflow-wrap: anywhere; } details { margin: 1rem 0; } pre, code { white-space: pre-wrap; overflow-wrap: anywhere; }`
})
export class RecorderConnectionComponent { readonly service = inject(BrowserRecorderService) }
