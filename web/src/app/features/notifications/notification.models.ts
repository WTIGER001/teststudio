import type { Page, RecordValue } from '../../core/api.service'
import type { Counts } from '../runs/run.models'

export interface NotificationDefinition {
  id: string; name: string; enabled: boolean; kind: string; secretReference: string
  suiteIds: string[] | null; environmentIds: string[] | null; stages: string[] | null
  triggers: string[] | null; outcomes: string[] | null; includeEvidenceGaps: boolean; maxAttempts: number
}
export interface Channel extends NotificationDefinition { projectId: string; owner: string; archived: boolean; deliveryEpoch: number }
export interface SecretReference { id: string; name: string }
export interface Delivery {
  id: string; projectId: string; runId: string; channelId: string; channelName: string
  destination: { channel: Channel; revision: number }
  message: { suiteId: string; environmentId?: string; stage?: string; trigger: string; state: string; outcome: string; evidenceState: string; counts: Counts; finishedAt: string; url: string }
  state: string; attempts: number; attemptLimit: number; nextAt: string; httpStatus?: number; error?: string
  acceptanceUnknown: boolean; claimUntil?: string; createdAt: string; updatedAt: string; deliveredAt?: string
}
export interface DeliveryPage extends Page<RecordValue<Delivery>> { partial: boolean; scanned: number }
export interface NotificationAttempt { number: number; state: string; startedAt: string; finishedAt?: string; httpStatus?: number; error?: string; acceptanceUnknown: boolean }
export interface NotificationForm {
  id: string; name: string; enabled: boolean; kind: string; secretReference: string
  suites: string; environments: string; stages: string; maxAttempts: number; includeEvidenceGaps: boolean
  manual: boolean; schedule: boolean; rerun: boolean
  passed: boolean; failed: boolean; timedOut: boolean; flaky: boolean; unknown: boolean
}
