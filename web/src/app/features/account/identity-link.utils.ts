import type { IdentityLinkOperation, IdentityLinkStart } from '../../core/generated/access.types'

export function identityLinkDefinition(value: { providerId: string; reason: string; password: string; confirmLocal: boolean }, source: string): IdentityLinkStart {
  return { providerId: value.providerId, reason: value.reason.trim(), ...(source === 'password' ? { password: value.password } : {}), ...(source === 'local' ? { confirmLocal: value.confirmLocal } : {}) }
}

export function identityLinkValidation(body: IdentityLinkStart, source: string): string {
  if (!body.providerId || !body.reason || body.reason.length > 1024) return 'Choose a provider and enter a reason of at most 1024 characters.'
  if (source === 'password' && !body.password) return 'Enter your current password to verify this account.'
  if (source === 'local' && !body.confirmLocal) return 'Confirm control of this local operator account.'
  return ''
}

export function identityLinkAuthorizePath(operation: IdentityLinkOperation): string {
  return `/auth/identity-links/${encodeURIComponent(operation.id)}/authorize`
}

export function identityLinkTerminal(state: IdentityLinkOperation['state']): boolean {
  return ['complete', 'cancelled', 'failed', 'expired'].includes(state)
}
