import type { RecordValue } from '../../core/api.service'
import type { WorkerExecution } from './run-workers.models'

export function pendingWorkers(items: RecordValue<WorkerExecution>[]): boolean {
  return items.some(item => item.data.state !== 'cleaned')
}
