import type { RecordValue } from '../../core/api.service'
import type { Schedule, ScheduleDefinition, ScheduleForm } from './schedule.models'

export function scheduleDefinition(value: ScheduleDefinition): ScheduleDefinition {
  const { id, name, enabled, kind, intervalSeconds, cron, timezone, overlap, request } = value
  return { id, name, enabled, kind, intervalSeconds, cron, timezone, overlap, request }
}
export function scheduleRows(items: RecordValue<Schedule>[]) {
  return items.map(item => ({ ...item,
    cadence: item.data.kind === 'interval' ? `Every ${item.data.intervalSeconds} seconds` : `${item.data.cron} · ${item.data.timezone}`,
    status: item.data.archived ? 'Archived' : item.data.enabled ? 'Enabled' : 'Paused'
  }))
}
export function scheduleForm(value?: ScheduleDefinition): ScheduleForm {
  return {
    id: value?.id ?? '', name: value?.name ?? '', enabled: value?.enabled ?? false, kind: value?.kind ?? 'interval',
    interval: value?.intervalSeconds ?? 60, cron: value?.cron ?? '0 2 * * *', timezone: value?.timezone ?? 'UTC', overlap: value?.overlap ?? 'skip',
    suite: value?.request.suiteId ?? '', environment: value?.request.environmentId ?? '', location: value?.request.executionLocation || 'local',
    release: value?.request.releaseId ?? '', ids: value?.request.testIds?.join(', ') ?? '', timeout: value?.request.timeoutSeconds ?? 1800, retries: value?.request.retries ?? 0
  }
}
export function definitionFromForm(value: ScheduleForm, original?: ScheduleDefinition): ScheduleDefinition {
  const kind = value.kind === 'cron' ? 'cron' : 'interval'
  return {
    id: value.id.trim(), name: value.name.trim(), enabled: value.enabled, kind,
    intervalSeconds: kind === 'interval' ? Number(value.interval) : undefined,
    cron: kind === 'cron' ? value.cron.trim() : undefined, timezone: kind === 'cron' ? value.timezone.trim() : 'UTC',
    overlap: value.overlap === 'allow' ? 'allow' : 'skip',
    request: { ...original?.request, suiteId: value.suite, environmentId: value.environment || undefined, executionLocation: value.location.trim(),
      releaseId: value.release.trim() || undefined, testIds: value.ids.split(/[\s,]+/).filter(Boolean), timeoutSeconds: Number(value.timeout), retries: Number(value.retries)
    }
  }
}

export function scheduleWizardIssue(step: number, value: ScheduleForm): string {
  if (step === 0) {
    if (!value.name.trim() || value.name.length > 256 || !/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(value.id)) return 'Enter a name and a valid identifier under Advanced selection options.'
    if (!value.suite || !value.location.trim()) return 'Choose a suite and an execution location.'
  }
  if (step === 1) {
    if (value.kind === 'interval' && (!Number.isInteger(Number(value.interval)) || value.interval < 5 || value.interval > 2592000)) return 'Enter an interval from 5 seconds through 30 days.'
    if (value.kind === 'cron' && (!value.cron.trim() || !value.timezone.trim())) return 'Enter a cron expression and timezone.'
    if (!Number.isInteger(Number(value.timeout)) || value.timeout < 1 || value.timeout > 3600 || !Number.isInteger(Number(value.retries)) || value.retries < 0 || value.retries > 2) return 'Check the run limit and retry count under Advanced run options.'
  }
  return ''
}
