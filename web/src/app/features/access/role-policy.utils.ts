import type { PermissionChoice, RolePolicyDetail, RolePolicyMutation, RolePolicyRecord, RolePreview, RolePreviewInput, RoleRule } from '../../core/generated/access.types'
import { reasonValidation } from './access.utils'

export interface RoleRuleFields { permission: string; effect: 'allow' | 'deny'; object: string; expiresAt: string }
export interface RolePolicyFields { id: string; name: string; description: string; reason: string; rules: RoleRuleFields[]; inherits: string[] }
const identifier = /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/
const bytes = (value: string) => new TextEncoder().encode(value).length

export function roleRoute(project: string, id?: string): string[] { const route = project ? ['/projects', project, 'roles'] : ['/settings', 'roles']; return id ? [...route, id] : route }
export function roleRuleFields(value?: RoleRule): RoleRuleFields { return { permission: value?.permission ?? '', effect: value?.effect ?? 'allow', object: value?.object ?? '', expiresAt: value?.expiresAt ?? '' } }
export function rolePolicyDefinition(value: RolePolicyFields, project: string, selected: RolePolicyRecord | null): RolePolicyMutation {
  return { id: selected?.id ?? value.id.trim(), projectId: selected?.data.projectId ?? project, name: value.name.trim(), description: value.description.trim(), reason: value.reason.trim(), disabled: selected?.data.disabled ?? false, rules: value.rules.map(rule => ({ permission: rule.permission, effect: rule.effect, object: rule.object.trim(), ...(rule.expiresAt.trim() ? { expiresAt: rule.expiresAt.trim() } : {}) })), inherits: value.inherits.map(id => id.trim()) }
}
export function roleExpiryValid(value?: string | null): boolean {
  if (!value) return true
  return /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,9})?(?:Z|[+-]\d\d:\d\d)$/.test(value) && Number(value.slice(11, 13)) < 24 && Number.isFinite(Date.parse(value)) && new Date(`${value.slice(0, 10)}T00:00:00Z`).toISOString().slice(0, 10) === value.slice(0, 10)
}
export function roleObjectValid(value: string): boolean { const parts = value.split('/'); return !value || (parts[0] === 'test' ? parts.length === 3 : parts.length === 2 && ['suite', 'run', 'environment', 'application', 'group', 'user', 'role'].includes(parts[0])) && parts.slice(1).every(id => identifier.test(id)) }
export function rolePolicyValidation(value: RolePolicyMutation, permissions: PermissionChoice[], requireReason = true): string {
  if (!identifier.test(value.id) || value.id.startsWith('builtin.') || ['custom', 'choices'].includes(value.id) || !value.name || bytes(value.name) > 200 || bytes(value.description ?? '') > 4096) return 'Enter a valid custom identifier, a name up to 200 bytes, and a description up to 4096 bytes.'
  if ((value.rules?.length ?? 0) > 100 || (value.inherits?.length ?? 0) > 16) return 'Use at most 100 rules and 16 directly inherited roles.'
  const seen = new Set<string>()
  for (const rule of value.rules ?? []) {
    if (!permissions.some(p => p.id === rule.permission) || !['allow', 'deny'].includes(rule.effect) || !roleObjectValid(rule.object ?? '') || !roleExpiryValid(rule.expiresAt)) return 'Each rule needs a registered permission, allow or deny, an exact object or empty scope, and valid UTC expiry.'
    const key = JSON.stringify([rule.permission, rule.effect, rule.object ?? '']); if (seen.has(key)) return 'Remove duplicate permission rules.'; seen.add(key)
  }
  const inherits = value.inherits ?? []
  if (inherits.some(id => !identifier.test(id) || id === value.id || id.startsWith('builtin.') || ['custom', 'choices'].includes(id)) || new Set(inherits).size !== inherits.length) return 'Select different, unique managed roles in this scope.'
  return requireReason ? reasonValidation(value.reason) : ''
}
export function roleRights(capabilities: Record<string, boolean>) { return { read: capabilities['teststudio.roles.read'] === true } }
export function inheritanceChoices(rows: RolePolicyRecord[], selected: string[], root: string): { id: string; name: string }[][] { return selected.map(id => [...(id && !rows.some(row => row.id === id) ? [{ id, name: `${id} · Current selection` }] : []), ...rows.filter(row => row.id !== root).map(row => ({ id: row.id, name: `${row.data.name}${row.data.disabled ? ' · Disabled' : ''}` }))]) }
export function permissionDescription(choices: PermissionChoice[], id: string): string { return choices.find(p => p.id === id)?.description ?? id }
export function previewFingerprint(policy: RolePolicyMutation, revision: number, userId: string, object: string): string { return JSON.stringify({ ...policy, reason: undefined, revision, userId: userId.trim(), object: object.trim() }) }
export function rolePreviewInput(policy: RolePolicyMutation, revision: number, userId: string, object: string): RolePreviewInput { return { policy: { ...policy, reason: 'Preview proposed role policy' }, revision, userId: userId.trim(), object: object.trim() } }
export function previewValidation(user: string, object: string): string { return identifier.test(user.trim()) && roleObjectValid(object.trim()) ? '' : 'Enter a registered user identifier and an exact object or empty scope.' }
export function previewRows(preview: RolePreview | null, showUnchanged = false): { permission: string; before: boolean; after: boolean }[] { return (preview?.decisions ?? []).filter(row => showUnchanged || row.before !== row.after) }
export function roleStateTarget(record: RolePolicyDetail): string { return `${record.data.name} (${record.id}) · ${record.data.projectId || 'Installation'}` }
