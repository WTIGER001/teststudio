import type { RecordValue } from '../../core/api.service'
import type { Application, Environment, Suite } from '../configuration/configuration.service'

export interface ApplicationSummary { id: string; name: string; description: string; environments: number; suites: number }

export function summarizeApplications(applications: RecordValue<Application>[], environments: RecordValue<Environment>[], suites: RecordValue<Suite>[]): ApplicationSummary[] {
  const environmentCounts = new Map<string, number>(), suiteCounts = new Map<string, number>()
  for (const row of environments) environmentCounts.set(row.data.applicationId, (environmentCounts.get(row.data.applicationId) ?? 0) + 1)
  for (const row of suites.filter(row => row.data.applications?.length === 1)) for (const id of new Set(row.data.applications ?? [])) suiteCounts.set(id, (suiteCounts.get(id) ?? 0) + 1)
  return applications.map(row => ({ id: row.id, name: row.data.name, description: row.data.description, environments: environmentCounts.get(row.id) ?? 0, suites: suiteCounts.get(row.id) ?? 0 }))
}

export function applicationResources(id: string, environments: RecordValue<Environment>[], suites: RecordValue<Suite>[]) {
  return { environments: environments.filter(row => row.data.applicationId === id), suites: suites.filter(row => row.data.applications?.length === 1 && row.data.applications[0] === id) }
}

export function findApplication(applications: RecordValue<Application>[], id: string): Application | null { return applications.find(row => row.id === id)?.data ?? null }
