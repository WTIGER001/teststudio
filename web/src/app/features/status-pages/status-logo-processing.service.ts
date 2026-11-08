import { Injectable } from '@angular/core'

@Injectable({ providedIn: 'root' })
export class StatusLogoProcessingService {
  async fromFile(file: File): Promise<string> {
    if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type) || file.size > 10 * 1024 * 1024) throw new Error('Choose a PNG, JPEG, or WebP logo up to 10 MB.')
    const image = await createImageBitmap(file)
    try {
      if (!image.width || !image.height || image.width * image.height > 40_000_000) throw new Error('Choose a logo smaller than 40 megapixels.')
      const canvas = document.createElement('canvas'), context = canvas.getContext('2d')
      if (!context) throw new Error('Image processing is unavailable in this browser. Use an image URL instead.')
      let scale = Math.min(1, 800 / image.width, 320 / image.height)
      for (let attempt = 0; attempt < 8; attempt++, scale *= .75) {
        canvas.width = Math.max(1, Math.round(image.width * scale)); canvas.height = Math.max(1, Math.round(image.height * scale))
        context.drawImage(image, 0, 0, canvas.width, canvas.height)
        const result = canvas.toDataURL('image/png')
        if (result.length <= 256 * 1024) return result
      }
      throw new Error('This logo is too detailed. Choose a simpler image.')
    } finally { image.close() }
  }
}
