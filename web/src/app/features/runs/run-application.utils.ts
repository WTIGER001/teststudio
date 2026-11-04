import type { Run } from './run.models'

export function runApplications(run: Run): string[] {
  if (run.request.applicationId) return [run.request.applicationId]
  if (run.environment?.applicationId) return [run.environment.applicationId]
  const applications = new Set<string>()
  for (const test of run.tests) {
    if (test.applicationId) applications.add(test.applicationId)
    else for (const application of run.applicationIds ?? []) applications.add(application)
  }
  if (!run.tests.length) for (const application of run.applicationIds ?? []) applications.add(application)
  return [...applications].sort()
}
