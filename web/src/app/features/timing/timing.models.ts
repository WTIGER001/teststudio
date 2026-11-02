export interface TimingPolicy {
  mode: 'off' | 'warn' | 'fail'; slowdownFactor: number; minimumIncreaseMs: number
  minimumSamples: number; window: number; maxAgeDays: number; timeoutMode: 'fixed' | 'adaptive'
  timeoutFactor: number; timeoutMarginMs: number; minimumTimeoutMs: number
}
export interface TimingContext { projectId: string; suiteId: string; testId: string; environmentId?: string; executionLocation: string; runner: string; profile: string }
export interface TimingSample { runId: string; durationMs: number; at: string }
export interface TimingWindow { context: TimingContext; generation: number; samples: TimingSample[]; resetAt?: string; resetReason?: string }
export interface TimingPrediction {
  baselineId: string; context: TimingContext; generation: number; baselineRevision: number; policy: TimingPolicy
  sampleCount: number; expectedMs?: number; p95Ms?: number; slowThresholdMs?: number; suggestedTimeoutMs?: number
  effectiveTimeoutMs: number; ready: boolean; asOf: string
}
export interface TimingAssessment { baseline: TimingPrediction; status: string; ratio?: number; censored: boolean; timeoutMs: number }
export interface TimingEstimate {
  tests: Record<string, TimingPrediction>; expectedExecutionMs?: number; highExecutionMs?: number
  readyTests: number; measuredTests: number; totalTests: number; provisional: boolean; testTimeoutBudgetMs: number; exceedsRunLimit: boolean
}
