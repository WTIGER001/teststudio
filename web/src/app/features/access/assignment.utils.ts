import type { AssignmentDefinition, AssignmentRecord, Role } from '../../core/generated/access.types'
import { reasonValidation } from './access.utils'

export function assignmentRights(capabilities: Record<string, boolean>) {
  const can = (permission: string) => capabilities[`teststudio.${permission}`] === true
  return { read: can('members.read') && can('permissions.read'), manage: can('members.manage') && can('permissions.grant'), readRoles: can('roles.read') }
}
export function assignableRoles(roles: Role[]): Role[] { return roles.filter(role => role.delegatable && !role.disabled) }
export function selectedRole(roles: Role[], id: string): Role | null { return roles.find(role => role.id === id) ?? null }
export function assignmentDefinition(value: AssignmentDefinition): AssignmentDefinition { return { principalKind: value.principalKind, principalId: value.principalId.trim(), roleId: value.roleId, reason: value.reason.trim() } }
export function assignmentValidation(value: AssignmentDefinition, roles: Role[]): string {
  if (!['user', 'group'].includes(value.principalKind) || !/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(value.principalId)) return 'Enter a registered user or group identifier.'
  if (!assignableRoles(roles).some(role => role.id === value.roleId)) return 'Choose an available role that you can assign.'
  return reasonValidation(value.reason)
}
export function assignmentTarget(row: AssignmentRecord): string { return `${row.data.principalName} (${row.data.principalId}) · ${row.data.roleName} · ${row.projectId || 'Installation'}` }
