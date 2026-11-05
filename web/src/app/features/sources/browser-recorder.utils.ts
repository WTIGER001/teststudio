import type { BrowserRecordingPairing, BrowserRecordingRecord } from '../../core/generated/access.types'

export function recordingActive(state: string | undefined): boolean {
  return state === 'pairing' || state === 'starting' || state === 'recording' || state === 'stopping'
}

export function recordingFlow(mode: string, state: string | undefined): { steps: string[]; current: number; title: string } {
  const hosted = mode === 'companion'
  const steps = hosted ? ['Configure', 'Connect desktop', 'Record', 'Review'] : ['Configure', 'Record', 'Review']
  const current = !state ? 0 : state === 'pairing' ? 1 : recordingActive(state) ? hosted ? 2 : 1 : steps.length - 1
  const titles = hosted ? ['Configure the browser test', 'Connect your desktop recorder', 'Record actions and assertions', 'Review the captured test'] : ['Configure the browser test', 'Record actions and assertions', 'Review the captured test']
  return { steps, current, title: titles[current] }
}

export function recordingStateLabel(state: string): string {
  return ({ pairing: 'Waiting for desktop', starting: 'Opening recording browser', recording: 'Recording', stopping: 'Closing recorder', ready: 'Ready for review', cancelled: 'Cancelled', failed: 'Recording stopped with a problem' } as Record<string, string>)[state] ?? state
}

export function recordingPairingExpired(pairing: BrowserRecordingPairing | null, now: number): boolean {
  return !pairing || !Number.isFinite(Date.parse(pairing.expiresAt)) || now >= Date.parse(pairing.expiresAt)
}

export function recordingRows(items: BrowserRecordingRecord[], row: BrowserRecordingRecord): BrowserRecordingRecord[] {
  return [row, ...items.filter(item => item.id !== row.id)]
}

export function desktopRecorderCommand(origin: string): string {
  const address = new URL(origin).origin
  return `teststudio-recorder --server '${address.replaceAll("'", "'\\''")}'`
}
