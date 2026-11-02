import type { Attempt } from './run.models'

export interface WorkerExecution {
  reference: { id: string; projectId: string; runId: string; testId: string; attempt: number; workerId: string; profile: string; container: string; volume: string }
  state: string; resultSaved: boolean; recovered: boolean; evidenceComplete: boolean; evidenceAbandoned: boolean
  evidenceErrors: string[]; error?: string; cleanupAttempts: number; reconcileAttempts: number
  createdAt: string; updatedAt: string; nextCheck: string; claimUntil?: string; recoveredResult?: Attempt
}
