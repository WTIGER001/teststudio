import type { TimingPolicy } from '../timing/timing.models'
import type { RecordValue } from '../../core/api.service'

export type JSONValue = string | number | boolean | null | JSONValue[] | { [key: string]: JSONValue }
export interface HTTPCheck {
  endpoint: string; path: string; method: string; statusCodes: number[]
  requiredText?: string; forbiddenText?: string; headers?: Record<string, string>; json?: Record<string, JSONValue>
  maxResponseMs?: number; minCertificateDays?: number
}

export interface TestDefinition {
  definitionSourceId?: string
  id: string
  title: string
  description: string
  applicationId?: string
  category: string
  tags: string[] | null
  owner?: string
  conditions?: string
  type: string
  runner: string
  state: string
  safety: string
  rationale: string
  dependencies: string[] | null
  requiredBindings: string[] | null
  timing?: TimingPolicy
  timeoutSeconds: number
  source?: string
  selector?: string
  check?: HTTPCheck
}

export interface CatalogItem extends RecordValue<TestDefinition> { suiteId: string; suiteName: string }
export interface Draft { id: string; tests: TestDefinition[]; publishedReleaseId?: string }
export interface Issue { testId?: string; field: string; message: string }
