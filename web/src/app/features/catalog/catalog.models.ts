import type { RecordValue } from '../../core/api.service'

export interface TestDefinition {
  id: string
  title: string
  description: string
  applicationId?: string
  category: string
  tags: string[]
  type: string
  runner: string
  state: string
  safety: string
  rationale: string
  dependencies: string[]
  requiredBindings: string[]
  timeoutSeconds: number
  source?: string
  selector?: string
  check?: { endpoint: string; path: string; method: string; statusCodes: number[] }
}

export interface CatalogItem extends RecordValue<TestDefinition> { suiteId: string; suiteName: string }
export interface Draft { id: string; tests: TestDefinition[]; publishedReleaseId?: string }
export interface Issue { testId?: string; field: string; message: string }
