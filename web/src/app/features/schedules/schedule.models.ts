import type { RunRequest } from '../runs/run.models'
import type { TimingEstimate } from '../timing/timing.models'

export interface ScheduleDefinition {
  id: string; name: string; enabled: boolean; kind: 'interval' | 'cron'; intervalSeconds?: number; cron?: string; timezone: string
  overlap: 'skip' | 'allow'; request: RunRequest
}
export interface Schedule extends ScheduleDefinition {
  projectId: string; owner: string; generation: number; archived: boolean; nextAt?: string; lastAt?: string
  lastRunId?: string; lastState?: string; lastError?: string; claimUntil?: string
}
export interface Firing {
  id: string; scheduleId: string; generation: number; owner: string; dueAt: string; at: string
  state: 'triggered' | 'skipped' | 'blocked'; runId?: string; error?: string; coalesced: boolean
}
export interface SchedulePreview { next: string[]; timing: TimingEstimate }
export interface ScheduleForm {
  id: string; name: string; enabled: boolean; kind: string; interval: number; cron: string; timezone: string; overlap: string
  suite: string; environment: string; location: string; release: string; ids: string; timeout: number; retries: number
}
