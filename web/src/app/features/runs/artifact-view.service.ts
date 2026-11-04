import { DestroyRef, effect, inject, Injectable, signal, untracked } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { ApiService } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { RunEvidenceService } from './run-evidence.service'
import { RunRetentionService } from './run-retention.service'
import type { Artifact } from './run.models'
import { privateTraceURL } from './trace-viewer.utils'
import { errorMessage, previewKind } from './run.utils'

interface Preview { artifact: Artifact; kind: 'text' | 'image' | 'video' | 'download' | 'trace'; viewerURL?: string; viewerUnavailable?: string; url: string; text: string; truncated: boolean; expiresAt: string }

@Injectable()
export class ArtifactViewService {
  private readonly api = inject(ApiService)
  readonly evidence = inject(RunEvidenceService)
  readonly workspace = inject(WorkspaceService)
  readonly retention = inject(RunRetentionService)
  readonly preview = signal<Preview | null>(null)
  readonly loading = signal(false)
  readonly error = signal('')
  private generation = 0

  constructor() {
    effect(() => { this.evidence.focusedTest(); this.evidence.focusedAttempt(); this.evidence.restricted(); untracked(() => { this.close(); this.error.set('') }) })
    const destroy = inject(DestroyRef)
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.close(); this.error.set('') })
    destroy.onDestroy(() => { this.generation++ })
  }

  close(): void { this.generation++; this.preview.set(null); this.loading.set(false) }

  async open(artifact: Artifact, action: 'view' | 'download'): Promise<void> {
    const generation = ++this.generation
    const base = this.evidence.base()
    this.loading.set(true); this.error.set(''); this.preview.set(null)
    try {
      const grant = await this.api.request<{ url: string; expiresAt: string; traceViewerUrl?: string; traceViewerUnavailable?: string }>(`${base}/artifacts/${encodeURIComponent(artifact.id)}/access`, 'POST', { action })
      const kind = action === 'download' ? 'download' : artifact.kind === 'trace' ? 'trace' : previewKind(artifact.contentType)
      const traceSupported = globalThis.isSecureContext && 'serviceWorker' in navigator
      const viewerURL = kind === 'trace' && traceSupported ? privateTraceURL(grant.traceViewerUrl, grant.url) : ''
      const viewerUnavailable = kind === 'trace' && !traceSupported ? 'Private trace viewing requires HTTPS or localhost and browser service-worker support. Download the trace for local inspection if permitted.' : grant.traceViewerUnavailable
      let text = '', truncated = false
      if (kind === 'text') {
        const response = await fetch(grant.url, { credentials: 'omit', cache: 'no-store' })
        if (!response.ok || !response.body) throw new Error('The artifact is no longer available. Refresh its access link.')
        const reader = response.body.getReader()
        const decoder = new TextDecoder()
        let remaining = 1 << 20
        try {
          for (;;) {
            const { value, done } = await reader.read()
            if (done) { text += decoder.decode(); break }
            const bytes = value.subarray(0, remaining)
            text += decoder.decode(bytes, { stream: true }); remaining -= bytes.length
            if (remaining === 0) { text += decoder.decode(); truncated = artifact.size > 1 << 20; break }
          }
        } finally { await reader.cancel() }
      }
      if (generation === this.generation && base === this.evidence.base()) this.preview.set({ artifact, kind, url: grant.url, expiresAt: grant.expiresAt, text, truncated, viewerURL, viewerUnavailable })
    } catch (error) { if (generation === this.generation) this.error.set(errorMessage(error, 'Artifact access could not be prepared.')) }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
}
