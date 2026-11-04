import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core'
import { DomSanitizer } from '@angular/platform-browser'

@Component({
  selector: 'ts-private-trace', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<p class="small muted">Private viewer hosted by this TestStudio installation. No public upload is required. Access is checked again while the viewer is open.</p><iframe title="Private Playwright trace" [src]="source()" sandbox="allow-scripts allow-same-origin" referrerpolicy="no-referrer"></iframe>`,
  styles: [`:host { display: block; min-width: 0; } iframe { display: block; width: 100%; min-height: 600px; height: 75vh; border: 1px solid var(--border); border-radius: 8px; background: var(--surface); }`]
})
export class PrivateTraceComponent {
  readonly url = input.required<string>()
  private readonly sanitizer = inject(DomSanitizer)
  readonly source = computed(() => this.sanitizer.bypassSecurityTrustResourceUrl(this.url()))
}
