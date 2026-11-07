import type { PermissionDefinition, PermissionExplanation, PermissionSource } from '../../core/generated/access.types'
import { previewValidation } from './role-policy.utils'

export interface ExplanationFields { userId: string; object: string; permission: string }
export interface SourceDetail { label: string; value: string }

export function explanationPermissions(permissions: PermissionDefinition[], project: string): PermissionDefinition[] { return permissions.filter(permission => !project || !permission.installationOnly) }
export function explanationValidation(fields: ExplanationFields, permissions: PermissionDefinition[]): string {
  return previewValidation(fields.userId, fields.object) || (permissions.some(permission => permission.id === fields.permission) ? '' : 'Choose a registered permission for this scope.')
}
export function explanationReason(reason: PermissionExplanation['reason']): string {
  return { allowed: 'At least one active allow applies, with no overriding deny.', 'explicit-deny': 'An active deny overrides the matching allows.', 'no-matching-allow': 'No active allow grants this permission in this context.', 'user-disabled': 'This account is disabled. Its requests are denied.' }[reason]
}
export function sourceTitle(source: PermissionSource): string { return source.restricted ? 'Restricted source' : source.kind === 'direct' ? 'Direct grant' : source.assignedRoleId !== source.roleId ? 'Inherited role grant' : 'Assigned role grant' }
export function sourceDetails(source: PermissionSource): SourceDetail[] {
  if (source.restricted) return []
  const details: SourceDetail[] = [{ label: 'Principal', value: `${source.principalName || source.principalId || 'Unknown principal'} · ${source.principalKind}/${source.principalId}` }]
  if (source.kind === 'role') {
    details.push({ label: 'Assigned role', value: `${source.assignedRoleName || source.assignedRoleId} · ${source.assignedRoleId}` })
    if (source.assignedRoleId !== source.roleId) details.push({ label: 'Inherited grant owner', value: `${source.roleName || source.roleId} · ${source.roleId}` })
    if (source.assignmentApplication) details.push({ label: 'Application binding', value: source.assignmentApplication })
    details.push({ label: 'Assignment binding', value: source.assignmentProject ? `Project · ${source.assignmentProject}` : 'No disclosed project binding' })
  }
  details.push({ label: 'Source ownership', value: source.scopeKnown ? source.projectId ? `Project · ${source.projectId}` : 'System' : 'Native scope · ownership not managed by TestStudio' })
  details.push({ label: 'Configured project scope', value: source.teamScope || 'Unspecified' }, { label: 'Configured object scope', value: source.objectScope || 'Whole scope' }, { label: 'Expiry', value: source.expiresAt || 'No expiry' })
  if (source.restrictedFields?.length) details.push({ label: 'Field restrictions', value: source.restrictedFields.join(', ') })
  if (source.grantId) details.push({ label: 'Managed grant identifier', value: source.grantId })
  return details
}
export function appendExplanation(previous: PermissionExplanation, next: PermissionExplanation): PermissionExplanation {
  if (!previous.cursor || ['userId', 'projectId', 'applicationId', 'object', 'permission', 'allowed', 'reason'].some(key => previous[key as keyof PermissionExplanation] !== next[key as keyof PermissionExplanation])) throw new Error('The explanation changed during the scan. Check access again to load current sources.')
  const ids = new Set(previous.items.map(source => source.id))
  return { ...next, items: [...previous.items, ...next.items.filter(source => !ids.has(source.id))] }
}
