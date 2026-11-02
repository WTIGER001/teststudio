export interface CapturePolicy {
  logs: boolean; screenshots: string; trace: string; video: string; maxBytes: number; maxFileBytes?: number
  retentionDays: number; kindRetentionDays?: Record<string, number>; failureRetentionDays?: number
}
export interface CaptureLimits { projectBytes: number; runBytes: number; fileBytes: number; retentionDays: number }
export interface EvidencePin { projectId: string; runId: string; pinned: boolean; until?: string; reason: string; actor: string; updatedAt: string; deletionsInProgress: number }
export interface RetentionView {
	finished: boolean
  pin: EvidencePin; revision: number; status: { state: string; error?: string; updatedAt: string }
  runQuota: { reservedBytes: number; limitBytes: number }; projectQuota: { reservedBytes: number; limitBytes: number }
  capture: CapturePolicy; limits: CaptureLimits
}
