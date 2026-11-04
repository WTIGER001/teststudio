import type { RecordValue } from '../../core/api.service'
import type { SuiteLifecycle } from '../../core/generated/access.types'
import type { Suite } from './configuration.service'

export interface CredentialReference { id: string; name: string; allowedOrigins?: string[] }
export interface CredentialBinding { binding: string; reference: string }

export function credentialBindingProblem(rows: CredentialBinding[], refs: CredentialReference[], address: string): { index: number; field: 'binding' | 'reference'; message: string } | undefined {
  const seen = new Set<string>()
  let origin = ''
  try { origin = new URL(address).origin } catch {}
  for (const [index, row] of rows.entries()) {
    if (!/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(row.binding) || seen.has(row.binding) || rows.length > 16) return { index, field: 'binding', message: 'Use up to 16 unique credential binding names.' }
    seen.add(row.binding)
    const ref = refs.find(ref => ref.id === row.reference)
    if (!ref || !ref.allowedOrigins?.includes(origin)) return { index, field: 'reference', message: 'Choose an approved credential reference for this exact target origin.' }
  }
  return undefined
}

export function credentialBindings(rows: CredentialBinding[]): Record<string, string> { return Object.fromEntries(rows.map(row => [row.binding, row.reference])) }

export function replaceConfiguredSuite(rows: RecordValue<Suite>[], state: SuiteLifecycle): RecordValue<Suite>[] {
  return rows.map(row => row.id === state.suite.id ? { ...row, revision: state.revision, updatedAt: state.updatedAt, data: { ...state.suite, applications: state.suite.applications ?? [] } } : row)
}
