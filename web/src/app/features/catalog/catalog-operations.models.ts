import type { TestDefinition, Issue } from './catalog.models'
import type { RecordValue } from '../../core/api.service'

export interface CatalogDocument { schemaVersion: string; originProjectId?: string; originSuiteId?: string; exportedAt?: string; tests: TestDefinition[] }
export interface CatalogPlan {
  id: string; projectId: string; suiteId: string; actor: string; kind: string; catalogRevision: number; suiteRevision: number
  changes: { test: TestDefinition; before?: TestDefinition; expectedRevision: number; action: string }[]
  issues: Issue[]; createdAt: string; expiresAt: string; appliedAt?: string
}
export interface TestRevision { test: TestDefinition; actor: string; at: string }
export interface CatalogRevision extends RecordValue<TestRevision> { number: number }
