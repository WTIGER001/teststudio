import type { RecordValue } from '../../core/api.service'
import type { Artifact } from './run.models'
import type { EvidencePin } from './run-retention.models'

export function activePin(pin: EvidencePin | undefined, now: number): boolean { return !!pin?.pinned && (!pin.until || Date.parse(pin.until) > now) }
export function effectiveArtifactState(artifact: Artifact, pinned: boolean, now: number): string { return artifact.state === 'available' && !pinned && Date.parse(artifact.expiresAt) <= now ? 'expired' : artifact.state }
export function pendingArtifacts(items: RecordValue<Artifact>[]): boolean { return items.some(item => ['pending', 'delete-pending', 'deleting'].includes(item.data.state)) }
export function localDateTime(value: string | undefined): string {
  if (!value) return ''
  const date = new Date(value)
  const part = (value: number) => String(value).padStart(2, '0')
  return `${date.getFullYear()}-${part(date.getMonth() + 1)}-${part(date.getDate())}T${part(date.getHours())}:${part(date.getMinutes())}`
}
export function pinInput(pinned: boolean, reason: string, until: string, now: number): { pinned: boolean; reason: string; until?: string } {
  const value: { pinned: boolean; reason: string; until?: string } = { pinned, reason: reason.trim() }
  if (!value.reason || value.reason.length > 500) throw new Error('Enter a reason of up to 500 characters.')
  if (pinned && until) {
    const date = new Date(until)
    if (!Number.isFinite(date.getTime()) || date.getTime() <= now) throw new Error('Choose a future pin expiry or leave it empty.')
    value.until = date.toISOString()
  }
  return value
}
export function bytesLabel(bytes: number): string {
  if (bytes === 0) return '0 B'
  const unit = Math.min(5, Math.floor(Math.log(Math.abs(bytes)) / Math.log(1024)))
  return `${new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(bytes / 1024 ** unit)} ${['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB'][unit]}`
}
