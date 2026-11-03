import { inject, Injectable } from '@angular/core'
import { ApiService } from './api.service'
import { accessOperations, type AccessRequests, type AccessResponses } from './generated/access.types'
import { accessPath } from './access-path.utils'

@Injectable({ providedIn: 'root' })
export class AccessClientService {
  private readonly api = inject(ApiService)

  call<K extends keyof AccessRequests>(operation: K, parameters: AccessRequests[K]): Promise<AccessResponses[K]> {
    const route = accessOperations[operation]
    const args = parameters as Record<string, unknown> & { query?: Record<string, string | number | boolean | undefined>; body?: unknown; revision?: number }
    return this.api.request<AccessResponses[K]>(accessPath(route.path, args, args.query), route.method, args.body, args.revision)
  }
}
