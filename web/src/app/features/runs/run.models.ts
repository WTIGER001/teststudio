import type { TimingEstimate, TimingAssessment } from '../timing/timing.models'
import type { TestDefinition } from '../catalog/catalog.models'
import type { Environment } from '../configuration/configuration.service'

export interface Counts { total: number; passed: number; failed: number; timedOut: number; flaky: number; skipped: number; completed: number }
export interface RunRequest { suiteId: string; environmentId?: string; releaseId?: string; testIds?: string[]; executionLocation: string; timeoutSeconds: number; retries?: number; parentRunId?: string }
export interface RunSummary {
  id: string; projectId: string; suiteId: string; environmentId?: string; environmentName?: string; stage?: string; protected: boolean
  state: string; outcome: string; phase: string; counts: Counts; createdAt: string; finishedAt?: string; evidenceState: string; actor: string; trigger: string; selection: string
}
export interface TestStep { title: string; category: string; durationMs: number; error?: string }
export interface HTTPObservation { status: number; url: string; durationMs: number; timingsMs?: Record<string, number>; certificateExpires?: string }
export interface GoObservation {
  package: string; test: string; toolchain: string; os: string; architecture: string
  buildDurationMs: number; buildExitCode?: number; exitCode?: number; outputTruncated?: boolean
}
export interface Attempt {
  executionId?: string; recovered?: boolean; durationUnavailable?: boolean; workerDurationMs?: number; steps?: TestStep[]; runnerStatus?: string; expectedStatus?: string
  testId: string; attempt: number; state: string; failureKind?: string; error?: string; startedAt: string; finishedAt: string; durationMs: number
  observedState?: string; totalDurationMs?: number; timingRegressions?: number; timing?: TimingAssessment
  http?: HTTPObservation
  go?: GoObservation
}
export interface Run {
  id: string; projectId: string; request: RunRequest; release: { id: string }; environment?: Environment; tests: TestDefinition[]
  timing?: TimingEstimate; currentTestStartedAt?: string; currentAttempt?: number
  actor: string; trigger: string; state: string; outcome: string; phase: string; currentTestId?: string; counts: Counts
  results: Attempt[]; createdAt: string; updatedAt: string; startedAt?: string; finishedAt?: string; deadline: string; workerId?: string
  cancelRequested: boolean; error?: string; evidenceState: string; evidenceErrors: string[]
}
export interface Artifact {
  id: string; projectId: string; runId: string; testId?: string; attempt?: number; workerId?: string; sequence?: number; stream?: string
  kind: string; filename: string; contentType: string; size: number; checksum: string; state: string; error?: string; createdAt: string; expiresAt: string
}
export interface RunEvent { id: string; sequence: number; kind: string; testId?: string; attempt?: number; message: string; at: string }
export interface LogPage { segments: { artifact: Artifact; text: string }[]; cursor?: string; truncated: boolean }
