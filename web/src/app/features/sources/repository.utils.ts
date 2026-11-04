import type { SourceRepositoryConfig } from '../../core/generated/access.types'

export interface RepositoryFormValue {
  id: string; name: string; repository: string; referenceKind: string; reference: string; credential: string; enabled: boolean; autoActivate: boolean; push: boolean; webhookCredential: string; intervalSeconds: number
  bindings: { suite: string; manifest: string; runner: string }[]
}
export function repositoryWizardProblem(value: RepositoryFormValue, step: number): { field: 'suite' | 'intervalSeconds' | 'webhookCredential'; binding?: number; message: string } | null {
  if (step === 1) {
    const suites = new Set<string>()
    for (let index = 0; index < value.bindings.length; index++) {
      const suite = value.bindings[index].suite
      if (suites.has(suite)) return { field: 'suite', binding: index, message: 'Choose each suite only once.' }
      suites.add(suite)
    }
  }
  if (step === 2 && (!Number.isInteger(value.intervalSeconds) || value.intervalSeconds !== 0 && value.intervalSeconds < 60)) return { field: 'intervalSeconds', message: 'Use 0 to disable periodic sync, or a whole number from 60 through 86400 seconds.' }
  if (step === 2 && value.push && !value.webhookCredential) return { field: 'webhookCredential', message: 'Choose an approved webhook secret for signed push synchronization.' }
  return null
}
export function repositoryConfig(value: RepositoryFormValue): SourceRepositoryConfig {
  const reference = value.referenceKind === 'commit' ? value.reference : `${value.referenceKind === 'tag' ? 'tags' : 'heads'}/${value.reference}`
  return { pushEnabled: value.push, webhookCredentialReference: value.webhookCredential || undefined, syncIntervalSeconds: value.intervalSeconds || undefined, name: value.name, provider: 'github', repository: value.repository, reference, credentialReference: value.credential || undefined, enabled: value.enabled, autoActivate: value.autoActivate, bindings: value.bindings.map(item => ({ suiteId: item.suite, manifestPath: item.manifest, runner: item.runner, buildProfile: item.runner === 'playwright' ? 'playwright-node' : 'go-module' })) }
}
export function repositoryReference(reference: string): { kind: string; value: string } {
  if (reference.startsWith('heads/')) return { kind: 'branch', value: reference.slice(6) }
  if (reference.startsWith('tags/')) return { kind: 'tag', value: reference.slice(5) }
  return { kind: 'commit', value: reference }
}
export function repositoryPreparations(values: Record<string, string>): { suite: string; id: string }[] {
  return Object.entries(values).map(([suite, id]) => ({ suite, id })).sort((a, b) => a.suite.localeCompare(b.suite))
}
export function repositorySyncActive(state: string | undefined): boolean { return state === 'received' || state === 'fetching' }

export function repositoryWebhookAddress(origin: string, project: string, source: string): string { return `${origin}/api/v1/projects/${encodeURIComponent(project)}/repository-sources/${encodeURIComponent(source)}/webhooks/github` }
