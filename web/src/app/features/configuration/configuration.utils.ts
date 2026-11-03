import type { RecordValue } from '../../core/api.service'
import type { SuiteLifecycle } from '../../core/generated/access.types'
import type { Suite } from './configuration.service'

export function replaceConfiguredSuite(rows: RecordValue<Suite>[], state: SuiteLifecycle): RecordValue<Suite>[] {
  return rows.map(row => row.id === state.suite.id ? { ...row, revision: state.revision, updatedAt: state.updatedAt, data: { ...state.suite, applications: state.suite.applications ?? [] } } : row)
}
