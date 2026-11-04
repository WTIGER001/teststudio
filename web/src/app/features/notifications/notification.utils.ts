import type { Channel, Delivery, NotificationDefinition, NotificationForm, SecretReference } from './notification.models'
import type { RecordValue } from '../../core/api.service'

const outcomeFields = [['passed', 'passed'], ['failed', 'failed'], ['timedOut', 'timed-out'], ['flaky', 'flaky'], ['unknown', 'unknown']] as const
const triggers = ['manual', 'schedule', 'rerun'] as const
const identifier = /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/
const bytes = (value: string) => new TextEncoder().encode(value).length

export function channelDefinition(channel: NotificationDefinition): NotificationDefinition {
  return { id: channel.id, name: channel.name, enabled: channel.enabled, kind: channel.kind, secretReference: channel.secretReference,
    suiteIds: channel.suiteIds ?? [], environmentIds: channel.environmentIds ?? [], stages: channel.stages ?? [], triggers: channel.triggers ?? [],
    outcomes: channel.outcomes ?? ['failed', 'timed-out', 'flaky', 'unknown'], includeEvidenceGaps: channel.includeEvidenceGaps, maxAttempts: channel.maxAttempts }
}

export function notificationForm(channel?: Channel): NotificationForm {
  const outcomes = channel?.outcomes ?? ['failed', 'timed-out', 'flaky', 'unknown']
  return { id: channel?.id ?? '', name: channel?.name ?? '', enabled: channel?.enabled ?? false, kind: channel?.kind ?? 'webhook', secretReference: channel?.secretReference ?? '',
    suites: (channel?.suiteIds ?? []).join(', '), environments: (channel?.environmentIds ?? []).join(', '), stages: (channel?.stages ?? []).join(', '),
    maxAttempts: channel?.maxAttempts ?? 6, includeEvidenceGaps: channel?.includeEvidenceGaps ?? false,
    ...Object.fromEntries(triggers.map(key => [key, !channel?.triggers?.length || channel.triggers.includes(key)])),
    ...Object.fromEntries(outcomeFields.map(([key, state]) => [key, outcomes.includes(state)])) } as NotificationForm
}

export function notificationDefinition(form: NotificationForm): NotificationDefinition {
  const selected = triggers.filter(key => form[key])
  return { id: form.id.trim(), name: form.name.trim(), enabled: form.enabled, kind: form.kind, secretReference: form.secretReference,
    suiteIds: form.suites.split(/[\s,]+/).filter(Boolean), environmentIds: form.environments.split(/[\s,]+/).filter(Boolean),
    stages: form.stages.split(',').map(value => value.trim()).filter(Boolean), triggers: selected.length === 3 ? [] : selected,
    outcomes: outcomeFields.filter(([key]) => form[key]).map(([, value]) => value), includeEvidenceGaps: form.includeEvidenceGaps, maxAttempts: form.maxAttempts }
}

export function notificationIdentityValidation(form: NotificationForm, references: SecretReference[]): string {
  const definition = notificationDefinition(form)
  if (!identifier.test(definition.id) || !definition.name || bytes(definition.name) > 256) return 'Enter a valid permanent identifier and a name of at most 256 bytes.'
  if (!references.some(ref => ref.id === definition.secretReference)) return 'Choose a registered secret reference for this project.'
  if (!['webhook', 'discord'].includes(definition.kind)) return 'Choose a supported transport.'
  return ''
}

export function notificationValidation(form: NotificationForm, references: SecretReference[]): string {
  const identity = notificationIdentityValidation(form, references)
  if (identity) return identity
  const definition = notificationDefinition(form)
  if (!Number.isInteger(definition.maxAttempts) || definition.maxAttempts < 1 || definition.maxAttempts > 10) return 'Select 1 through 10 delivery attempts.'
  if (!triggers.some(key => form[key])) return 'Select at least one run trigger.'
  if (!definition.outcomes?.length && !form.includeEvidenceGaps) return 'Select an outcome or include evidence gaps.'
  for (const [values, ids] of [[definition.suiteIds, true], [definition.environmentIds, true], [definition.stages, false]] as const) {
    if ((values?.length ?? 0) > 100 || new Set(values).size !== values?.length || values?.some(value => bytes(value) > 128 || (ids && !identifier.test(value)))) return 'Filters require unique values, at most 100 per field, with valid identifiers and at most 128 bytes per value.'
  }
  return ''
}

export function deliveryStatus(delivery: Delivery, now: number): string {
  return delivery.state === 'sending' && !!delivery.claimUntil && Date.parse(delivery.claimUntil) <= now ? 'Recovery pending' : delivery.state
}
export function canRetryDelivery(state: string): boolean { return ['failed', 'blocked', 'cancelled', 'retrying'].includes(state) }
export function canCancelDelivery(state: string): boolean { return ['pending', 'retrying'].includes(state) }
export function deliveryRows(items: RecordValue<Delivery>[], now: number) {
  return items.map(item => ({ ...item, status: deliveryStatus(item.data, now), retry: canRetryDelivery(item.data.state), cancel: canCancelDelivery(item.data.state) }))
}
