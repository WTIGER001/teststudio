import type { PasswordChange, PasswordLogin } from '../../core/generated/access.types'

export function passwordPolicy(value: string): string {
  const normalized = value.normalize('NFC'), size = Array.from(normalized).length
  if (size < 15 || size > 256 || new TextEncoder().encode(value).length > 1024 || new TextEncoder().encode(normalized).length > 1024) return 'Use 15 through 256 characters and at most 1024 bytes. Spaces are significant.'
  if (/[\u0000-\u001f\u007f-\u009f]/u.test(normalized)) return 'Passwords cannot contain control characters.'
  return ''
}
export function loginDefinition(value: PasswordLogin): PasswordLogin { return { userId: value.userId.trim(), password: value.password } }
export function loginValidation(value: PasswordLogin): string { return !/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(value.userId) || !value.password || new TextEncoder().encode(value.password).length > 1024 ? 'Enter your user identifier and password.' : '' }
export function passwordChangeDefinition(value: PasswordChange): PasswordChange { return { currentPassword: value.currentPassword, newPassword: value.newPassword } }
export function passwordChangeValidation(value: PasswordChange, confirmation: string): string {
  if (!value.currentPassword) return 'Enter your current password.'
  if (value.newPassword !== confirmation) return 'The new passwords must match.'
  if (value.currentPassword.normalize('NFC') === value.newPassword.normalize('NFC')) return 'Choose a different new password.'
  return passwordPolicy(value.newPassword)
}
