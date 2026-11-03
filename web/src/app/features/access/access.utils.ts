import type { Group, GroupDefinition, Membership, User, UserDefinition } from '../../core/generated/access.types'

const identifier = /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/
const bytes = (value: string) => new TextEncoder().encode(value).length

export function userDefinition(value: Pick<User, 'id' | 'name' | 'email'>): UserDefinition { return { id: value.id.trim(), name: value.name.trim(), email: value.email?.trim() ?? '' } }
export function groupDefinition(value: Pick<Group, 'id' | 'name' | 'description' | 'projectId'>): GroupDefinition { return { id: value.id.trim(), name: value.name.trim(), description: value.description, projectId: value.projectId ?? '' } }
export function groupRoute(group: Pick<Group, 'id' | 'projectId'>): string[] { return group.projectId ? ['/projects', group.projectId, 'groups', group.id] : ['/settings', 'groups', group.id] }
export function userValidation(value: UserDefinition): string {
  if (!identifier.test(value.id) || !value.name || bytes(value.name) > 256) return 'Enter a valid identifier and a name of at most 256 bytes.'
  if ((value.email && !/^[^\s@<>,;]+@[^\s@<>,;]+$/.test(value.email)) || bytes(value.email ?? '') > 254) return 'Enter one email address of at most 254 bytes, or leave it empty.'
  return ''
}
export function groupValidation(value: GroupDefinition): string {
  if (!identifier.test(value.id) || !value.name || bytes(value.name) > 256 || bytes(value.description ?? '') > 4096) return 'Enter a valid identifier, a name of at most 256 bytes, and a description of at most 4096 bytes.'
  return ''
}
export function reasonValidation(value: string): string { return !value.trim() || bytes(value) > 1024 ? 'Enter a reason of 1 through 1024 bytes.' : '' }
export function membershipExpiry(value: string, now: number): { expiresAt: string | null; error: string } {
  if (!value.trim()) return { expiresAt: null, error: '' }
  const local = /^\d{4}-\d\d-\d\dT\d\d:\d\d(?::\d\d)?$/.test(value)
  const timestamp = Date.parse(local ? `${value}Z` : value)
  const valid = Number.isFinite(timestamp) && timestamp > now && (local ? new Date(timestamp).toISOString().startsWith(value) : /T.*(?:Z|[+-]\d\d:\d\d)$/.test(value))
  return valid ? { expiresAt: new Date(timestamp).toISOString(), error: '' } : { expiresAt: null, error: 'Enter a future UTC timestamp, or leave expiry empty.' }
}
export function expiryForm(value?: string | null): string { const time = value ? Date.parse(value) : NaN; return Number.isFinite(time) ? new Date(time).toISOString().slice(0, 19) : '' }
export function canEditMembership(group: Group | null, manage: boolean, membership?: Membership): boolean { return !!group && manage && !group.archived && !group.managedBy && (!membership || (!membership.readOnly && membership.source === 'local')) }
export function membershipValidation(user: string): string { return identifier.test(user) ? '' : 'Enter a valid registered user identifier.' }
export function accessRights(capabilities: Record<string, boolean>, project: string) {
  const can = (permission: string) => capabilities[`teststudio.${permission}`] === true
  return { readUsers: can('users.read'), manageUsers: can('users.manage'), readGroups: can('groups.read'), manageGroups: can('groups.manage'), readMembers: can('groups.read') && (!project || can('members.read')), manageMembers: can('groups.manage') && (!project || can('members.manage')) }
}
export function accessFocus(id: string): void { setTimeout(() => document.getElementById(id)?.focus(), 0) }
