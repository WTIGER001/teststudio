import type { RecordValue } from '../../core/api.service'
import type { Environment, Suite } from '../configuration/configuration.service'
import { validWebAddress } from '../../shared/wizard/creation.utils'

export interface ProbeAddress { origin: string; path: string; hostname: string; protocol: string }
export interface ProbeAddressMatch { key: string; environment: RecordValue<Environment>; endpoint: string; path: string }

export function parseProbeAddress(value: string): ProbeAddress | undefined {
  if (!validWebAddress(value)) return undefined
  const address = new URL(value)
  return { origin: address.origin, path: address.pathname + address.search, hostname: address.hostname, protocol: address.protocol }
}

export function matchProbeAddress(value: string, environments: RecordValue<Environment>[], suite?: Suite): ProbeAddressMatch[] {
  const address = parseProbeAddress(value)
  if (!address) return []
  const requested = new URL(value), matches: ProbeAddressMatch[] = []
  for (const environment of environments) {
    if (suite?.applications?.length && !suite.applications.includes(environment.data.applicationId)) continue
    for (const [endpoint, base] of Object.entries(environment.data.endpoints)) {
      try {
        const registered = new URL(base), prefix = registered.pathname.replace(/\/+$/, '')
        if (registered.origin !== address.origin || prefix && !requested.pathname.startsWith(prefix + '/')) continue
        matches.push({ key: `${environment.id}/${endpoint}`, environment, endpoint, path: (requested.pathname.slice(prefix.length) || '/') + requested.search })
      } catch {}
    }
  }
  return matches
}
