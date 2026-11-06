import { Injectable } from '@angular/core'

@Injectable()
export class ProjectImageProcessingService {
  async fromFile(file: File): Promise<string> {
    if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type) || file.size > 10 * 1024 * 1024) throw new Error('Choose a PNG, JPEG, or WebP image up to 10 MB.')
    const image = await createImageBitmap(file)
    try { return this.resize(image, image.width, image.height) }
    finally { image.close() }
  }

  resize(image: CanvasImageSource, width: number, height: number): string {
    if (!width || !height) throw new Error('The image has no visible frame. Try again after the page loads.')
    const scale = Math.min(1, 1200 / Math.max(width, height))
    const canvas = document.createElement('canvas')
    canvas.width = Math.max(1, Math.round(width * scale)); canvas.height = Math.max(1, Math.round(height * scale))
    const context = canvas.getContext('2d')
    if (!context) throw new Error('Image processing is unavailable in this browser.')
    context.fillStyle = '#ffffff'; context.fillRect(0, 0, canvas.width, canvas.height)
    context.drawImage(image, 0, 0, canvas.width, canvas.height)
    for (const quality of [.8, .65, .45, .25]) {
      const result = canvas.toDataURL('image/jpeg', quality)
      if (result.length <= 256 * 1024) return result
    }
    throw new Error('This image is too detailed. Choose a smaller image.')
  }

}
