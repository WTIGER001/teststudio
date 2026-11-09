import { inject, Injectable } from '@angular/core'
import { ApiError, ApiService, Page, RecordValue } from './api.service'
import { creationMatches } from '../shared/wizard/creation.utils'

@Injectable()
export class RecordCreationService {
  private readonly api = inject(ApiService)
  private readonly uncertain = new Map<string, string>()

  async create<T>(path: string, data: T, method: 'POST' | 'PUT', recordPath: string, revision?: number): Promise<RecordValue<T>> {
    const fingerprint = JSON.stringify(data)
    if (this.uncertain.has(recordPath)) {
      if (this.uncertain.get(recordPath) !== fingerprint) throw new Error('A previous save has an unknown outcome. Restore its entries or refresh and review the saved record before changing them.')
      try {
        const separator = recordPath.lastIndexOf('/')
        const record = method === 'PUT'
          ? (await this.api.request<Page<RecordValue<T>>>(recordPath.slice(0, separator))).items.find(row => row.id === recordPath.slice(separator + 1))
          : await this.api.request<RecordValue<T>>(recordPath)
        if (record) {
          if (!creationMatches(data, record.data)) throw new Error('The saved record differs from these entries. Open it and review the current version before continuing.')
          this.uncertain.delete(recordPath)
          return record
        }
      } catch (error) { if (!(error instanceof ApiError && error.status === 404)) throw error }
    }
    try {
      const record = await this.api.request<RecordValue<T>>(path, method, data, revision ?? (method === 'PUT' ? 0 : undefined))
      this.uncertain.delete(recordPath)
      return record
    } catch (error) {
      if (!(error instanceof ApiError) || error.status >= 500) this.uncertain.set(recordPath, fingerprint)
      throw error
    }
  }
}
