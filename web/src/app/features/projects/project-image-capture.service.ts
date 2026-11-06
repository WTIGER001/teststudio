import { inject, Injectable, OnDestroy } from '@angular/core'
import { ProjectImageProcessingService } from './project-image-processing.service'

@Injectable()
export class ProjectImageCaptureService implements OnDestroy {
  private readonly images = inject(ProjectImageProcessingService)
  private stream?: MediaStream
  private generation = 0
  available(): boolean { return typeof navigator.mediaDevices?.getDisplayMedia === 'function' }

  async capture(): Promise<string> {
    const generation = ++this.generation
    const stream = await navigator.mediaDevices.getDisplayMedia({ video: { displaySurface: 'browser' }, audio: false })
    this.stream = stream
    const video = document.createElement('video')
    let timeout: ReturnType<typeof setTimeout> | undefined
    try {
      if (generation !== this.generation) throw new Error('Capture cancelled.')
      video.muted = true; video.srcObject = stream
      await Promise.race([video.play(), new Promise<never>((_, reject) => { timeout = setTimeout(() => reject(new Error('Capture did not produce a frame. Try again or upload a screenshot.')), 15000) })])
      if (generation !== this.generation) throw new Error('Capture cancelled.')
      return this.images.resize(video, video.videoWidth, video.videoHeight)
    } finally {
      clearTimeout(timeout)
      stream.getTracks().forEach(track => track.stop())
      video.pause(); video.srcObject = null
      if (this.stream === stream) this.stream = undefined
    }
  }

  stop(): void { this.generation++; this.stream?.getTracks().forEach(track => track.stop()); this.stream = undefined }
  ngOnDestroy(): void { this.stop() }
}
