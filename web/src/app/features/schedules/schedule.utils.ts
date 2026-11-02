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
