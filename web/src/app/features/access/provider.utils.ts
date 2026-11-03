import type { ProviderMutation, ProviderRecord } from '../../core/generated/access.types'
import { reasonValidation } from './access.utils'

export interface ProviderMappingFields { external: string; groupId: string }
export interface ProviderFields { id: string; name: string; issuer: string; clientId: string; secretRef: string; scopes: string; disabled: boolean; autoProvision: boolean; groupClaim: string; membershipTtlSeconds: number; reason: string; mappings: ProviderMappingFields[] }
const identifier = /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/
const bytes = (value: string) => new TextEncoder().encode(value).length

export function providerFields(record?: ProviderRecord): ProviderFields {
  const p = record?.data
  return { id: record?.id ?? '', name: p?.name ?? '', issuer: p?.issuer ?? '', clientId: p?.clientId ?? '', secretRef: p?.secretRef ?? '', scopes: p?.scopes.join(' ') ?? 'openid profile email', disabled: p?.disabled ?? true, autoProvision: p?.autoProvision ?? false, groupClaim: p?.groupClaim ?? '', membershipTtlSeconds: p?.membershipTtlSeconds ?? 1800, reason: '', mappings: providerMappings(p?.groupMappings ?? {}) }
}
export function providerMappings(mappings: Record<string, string>): ProviderMappingFields[] { return Object.entries(mappings).sort(([a], [b]) => a.localeCompare(b)).map(([external, groupId]) => ({ external, groupId })) }
export function providerDefinition(fields: ProviderFields): ProviderMutation { return { id: fields.id.trim(), name: fields.name.trim(), issuer: fields.issuer.trim(), clientId: fields.clientId.trim(), secretRef: fields.secretRef, scopes: fields.scopes.trim().split(/\s+/).filter(Boolean), disabled: fields.disabled, autoProvision: fields.autoProvision, groupClaim: fields.groupClaim, groupMappings: Object.fromEntries(fields.mappings.map(mapping => [mapping.external, mapping.groupId.trim()])), membershipTtlSeconds: fields.membershipTtlSeconds, reason: fields.reason.trim() } }
export function providerIssuerValid(value: string): boolean {
  try { const url = new URL(value); return !url.username && !url.password && !value.includes('?') && !value.includes('#') && (url.protocol === 'https:' || url.protocol === 'http:' && (['localhost', '[::1]'].includes(url.hostname) || /^127(?:\.\d{1,3}){3}$/.test(url.hostname))) } catch { return false }
}
export function providerValidation(fields: ProviderFields, refs: string[], selected: ProviderRecord | null): string {
  const p = providerDefinition(fields)
  if (!identifier.test(p.id) || !p.name || bytes(p.name) > 200 || !p.clientId || bytes(p.clientId) > 512 || bytes(p.issuer) > 2048 || !providerIssuerValid(p.issuer)) return 'Enter a valid identifier, bounded name and client identifier, and an HTTPS issuer or loopback HTTP issuer.'
  if (p.secretRef && !refs.includes(p.secretRef) && !(selected?.data.secretRef === p.secretRef && p.disabled)) return 'Choose an operator-approved credential reference, or no client secret.'
  if (!Number.isInteger(p.membershipTtlSeconds) || p.membershipTtlSeconds < 0 || p.membershipTtlSeconds > 3600) return 'Use a membership lifetime from 0 through 3600 seconds. Zero uses 1800 seconds.'
  if (p.scopes.length > 32 || new Set(p.scopes).size !== p.scopes.length || p.scopes.some(scope => bytes(scope) > 256 || !/^[\x21\x23-\x5b\x5d-\x7e]+$/.test(scope))) return 'Use at most 32 unique printable scope tokens.'
  if (bytes(p.groupClaim ?? '') > 256 || fields.mappings.length > 40 || fields.mappings.length && !p.groupClaim) return 'Group mappings require a claim name and at most 40 rows.'
  if (new Set(fields.mappings.map(mapping => mapping.external)).size !== fields.mappings.length || fields.mappings.some(mapping => !mapping.external.trim() || bytes(mapping.external) > 512 || !identifier.test(mapping.groupId.trim()))) return 'Each mapping needs a unique exact external value and an existing provider-owned group identifier.'
  return reasonValidation(p.reason)
}
export function providerCredentialChoices(refs: string[], current: string): { id: string; name: string }[] { return [{ id: '', name: 'No client secret · public client' }, ...refs.map(id => ({ id, name: id })), ...current && !refs.includes(current) ? [{ id: current, name: `${current} · no longer approved` }] : []] }
export function providerRoute(id?: string): string[] { return id ? ['/settings', 'providers', id] : ['/settings', 'providers'] }
export function providerGroupValidation(value: { id: string; name: string; projectId: string }): string { return identifier.test(value.id.trim()) && value.name.trim() && bytes(value.name.trim()) <= 256 && (!value.projectId.trim() || identifier.test(value.projectId.trim())) ? '' : 'Enter a valid group identifier, a name up to 256 bytes, and an optional registered project identifier.' }
