import type { RecordValue } from '../../core/api.service'
import type { Application, Environment } from './configuration.service'

export interface Requirement { name: string; kind: 'endpoint' | 'parameter' | 'secret'; valueType?: string; description?: string; required: boolean; defaultValue?: string }
export interface ConfigurationItem { name: string; kind: string; status: string; message?: string }
export interface EnvironmentConfiguration extends RecordValue<Environment> { readiness: { ready: boolean; items: ConfigurationItem[] } }
export interface ApplicationConfiguration { application: RecordValue<Application>; requirements: Requirement[]; environments: EnvironmentConfiguration[] }
