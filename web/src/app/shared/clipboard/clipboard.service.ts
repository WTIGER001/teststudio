import { Injectable } from '@angular/core'

@Injectable({ providedIn: 'root' })
export class ClipboardService {
  async copy(value: string): Promise<boolean> {
    try { await navigator.clipboard.writeText(value); return true }
    catch { return false }
  }
}
