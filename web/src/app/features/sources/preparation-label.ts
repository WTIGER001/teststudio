import type { SourcePreparationJob } from '../../core/generated/access.types'

export function preparationStateLabel(job: Pick<SourcePreparationJob, 'state' | 'assemblyComplete'>): string {
  return job.state === 'received' && job.assemblyComplete ? 'Waiting for discovery' : job.state
}
