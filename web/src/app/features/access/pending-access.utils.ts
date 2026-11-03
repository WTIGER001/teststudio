import type { PendingAccessInput, PendingAccessRecord, PendingProviderChoice, Role } from '../../core/generated/access.types'
import { assignableRoles } from './assignment.utils'
import { reasonValidation } from './access.utils'

export interface PendingAccessFields { providerId: string; email: string; roleId: string; reason: string }

export function pendingAccessDefinition(value: PendingAccessFields, providers: PendingProviderChoice[]): PendingAccessInput {
  return { providerId: value.providerId, providerRevision: providers.find(provider => provider.id === value.providerId)?.revision ?? 0, email: value.email.trim(), roleId: value.roleId, reason: value.reason.trim() }
}

export function pendingAccessValidation(body: PendingAccessInput, roles: Role[]): string {
  if (!body.providerId || body.providerRevision < 1) return 'Choose a currently enabled provider from the reviewed choices.'
  if (!body.email || body.email.length > 254 || !body.email.includes('@')) return 'Enter one complete email address of at most 254 characters.'
  if (!assignableRoles(roles).some(role => role.id === body.roleId)) return 'Choose an available project role that you can assign.'
  return reasonValidation(body.reason)
}

export function pendingAccessTarget(row: PendingAccessRecord): string { return `${row.data.email} · ${row.data.providerName} · ${row.data.roleName} · ${row.data.projectId}` }

export function pendingBlockReason(reason: PendingAccessRecord['data']['blockReason']): string {
  switch (reason) {
    case 'provider-changed': return 'Provider configuration changed. Review the current provider before preparing access again.'
    case 'role-changed': return 'The approved role or its inherited policies changed. Review the current permissions before preparing access again.'
    case 'authority-changed': return 'The approving owner no longer has the required delegation authority. A current access manager must review access again.'
    case 'project-unavailable': return 'The project is unavailable or archived. Restore it before preparing access again.'
    default: return ''
  }
}
