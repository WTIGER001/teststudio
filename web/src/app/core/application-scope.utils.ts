export function applicationFromURL(url: string): string {
  const match = url.split(/[?#]/)[0].match(/^\/projects\/[^/]+\/applications\/([^/]+)/)
  return match ? decodeURIComponent(match[1]) : ''
}

export function scopePath(project: string, application: string, ...parts: string[]): string[] {
  const scoped = ['suites', 'catalog', 'runs', 'trends', 'tests', 'members', 'explain-access'].includes(parts[0])
  return ['/projects', project, ...(application && scoped ? ['applications', application] : []), ...parts]
}

export function switchApplicationPath(project: string, application: string, url: string): string[] {
  const module = url.split(/[?#]/)[0].replace(/^\/projects\/[^/]+(?:\/applications\/[^/]+)?\/?/, '').split('/')[0]
  const destination = ['suites', 'catalog', 'runs', 'trends', 'members', 'explain-access'].includes(module) ? module : ''
  return destination ? scopePath(project, application, destination) : ['/projects', project, 'applications', ...(application ? [application] : [])]
}

export function applicationSuites<T extends { data: { applications: string[] } }>(rows: T[], application: string): T[] {
  return application ? rows.filter(row => row.data.applications.includes(application)) : rows
}

export function applicationEnvironments<T extends { data: { applicationId: string } }>(rows: T[], application: string): T[] {
  return application ? rows.filter(row => row.data.applicationId === application) : rows
}
