import type { RecordValue } from '../../core/api.service'
import type { Environment, Suite } from '../configuration/configuration.service'

export function probeTargets(environments: RecordValue<Environment>[], suites: RecordValue<Suite>[], suiteId: string): RecordValue<Environment>[] {
  const applications = suites.find(row => row.id === suiteId)?.data.applications ?? []
  return environments.filter(row => !row.data.disabled && (!applications.length || applications.includes(row.data.applicationId)))
}

export function probeURL(base: string | undefined, path: string): string {
  if (!base) return 'Endpoint will be resolved from the run environment.'
  try {
    const address = new URL(base), query = path.indexOf('?')
    address.pathname = address.pathname.replace(/\/+$/, '') + (query < 0 ? path : path.slice(0, query))
    address.search = query < 0 ? '' : path.slice(query)
    address.hash = ''
    return address.href
  } catch { return 'Check the target address and relative path.' }
}
