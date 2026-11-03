import type { GrantDefinition, GrantRecord, GrantUpdate, PermissionChoice } from '../../core/generated/access.types'
import { reasonValidation } from './access.utils'
import { roleExpiryValid, roleObjectValid } from './role-policy.utils'

export interface GrantFields { principalKind: 'user' | 'group'; principalId: string; permission: string; effect: 'allow' | 'deny'; object: string; expiresAt: string; reason: string }
export function grantFields(row?: GrantRecord): GrantFields { return { principalKind: row?.data.principalKind ?? 'user', principalId: row?.data.principalId ?? '', permission: row?.data.permission ?? '', effect: row?.data.effect ?? 'allow', object: row?.data.object ?? '', expiresAt: row?.data.expiresAt ?? '', reason: '' } }
export function grantDefinition(value: GrantFields): GrantDefinition { return { principalKind: value.principalKind, principalId: value.principalId.trim(), permission: value.permission, effect: value.effect, object: value.object.trim(), expiresAt: value.expiresAt.trim() || null, reason: value.reason.trim() } }
export function grantUpdate(value: GrantFields): GrantUpdate { return { expiresAt: value.expiresAt.trim() || null, reason: value.reason.trim() } }
export function grantValidation(value: GrantFields, selected: GrantRecord | null, permissions: PermissionChoice[]): string {
  if (!roleExpiryValid(value.expiresAt.trim())) return 'Enter a complete UTC expiry timestamp, or leave expiry empty.'
  if (!selected && (!['user', 'group'].includes(value.principalKind) || !/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(value.principalId.trim()) || !['allow', 'deny'].includes(value.effect) || !roleObjectValid(value.object.trim()) || !permissions.some(p => p.id === value.permission && p.canGrant))) return 'Enter a registered principal and an exact object or empty scope. Select a permission you can grant.'
  return reasonValidation(value.reason)
}
export function grantTarget(row: GrantRecord): string { return `${row.data.principalName} (${row.data.principalKind}/${row.data.principalId}) · ${row.data.effect} ${row.data.permission} · ${row.data.projectId || 'Installation'} · ${row.data.object || 'Whole scope'}` }
export function grantReadRights(capabilities: Record<string, boolean>): boolean { return capabilities['teststudio.members.read'] === true && capabilities['teststudio.permissions.read'] === true }
