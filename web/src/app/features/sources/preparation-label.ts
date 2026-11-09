import type { SourcePreparationJob } from '../../core/generated/access.types'

export function preparationStateLabel(job: Pick<SourcePreparationJob, 'state' | 'assemblyComplete'>): string {
  return job.state === 'received' && job.assemblyComplete ? 'Waiting for discovery' : ({ published: 'Ready', ready: 'Ready', received: 'Setting up', preparing: 'Setting up', discovering: 'Checking tests' } as Record<string, string>)[job.state] ?? job.state
}
