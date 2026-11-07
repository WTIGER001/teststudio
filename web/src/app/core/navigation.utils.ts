import { applicationFromURL } from './application-scope.utils'
export interface NavigationLink { label: string; path: string; icon: string; exact?: boolean; active: boolean; queryParams?: Record<string, string> }
export interface NavigationSection { id: string; title: string; collapsible: boolean; active: boolean; links: NavigationLink[] }
type Capabilities = Record<string, boolean>

export function navigationSections(project: string, capabilities: Capabilities, installation: Capabilities, url: string, projectCapabilities: Capabilities = capabilities): NavigationSection[] {
  const path = url.split(/[?#]/)[0]
  const allowed = (scope: Capabilities, ...permissions: string[]) => permissions.every(permission => scope[`teststudio.${permission}`] === true)
  const link = (label: string, target: string, icon: string, exact = false, related: string[] = []): NavigationLink => ({ label, path: target, icon, exact, active: path === target || !exact && path.startsWith(target + '/') || related.some(prefix => path === prefix || path.startsWith(prefix + '/')) })
  const sections: NavigationSection[] = []
  const add = (id: string, title: string, collapsible: boolean, links: NavigationLink[]) => {
    if (links.length) sections.push({ id, title, collapsible, links, active: links.some(item => item.active) })
  }
  if (project) {
    const base = `/projects/${project}`
    const application = applicationFromURL(url)
    const testing = application ? `${base}/applications/${application}` : base
    add('testing', application ? 'Application testing' : 'All application testing', false, [
      ...(application && allowed(capabilities, 'projects.read') ? [link('Application overview', testing, '▦', true)] : []),
      ...(allowed(capabilities, 'suites.read') ? [link('Suites', `${testing}/suites`, '▤')] : []),
      link('Test catalog', `${testing}/catalog`, '≡', false, [`${testing}/tests`]), link('Test runs', `${testing}/runs`, '▷'),
      link('Trends', `${testing}/trends`, '▥')
    ])
    add('overviews', 'Project overviews', false, [
      ...(allowed(capabilities, 'projects.read') ? [link('All applications', `${base}/applications`, '▦', true)] : []),
      link('HTTPS probes', `${base}/probes`, '◎', true)
    ])
    add('automation', 'Automation', true, [
      ...(allowed(capabilities, 'schedules.read') ? [link('Schedules', `${base}/schedules`, '◷')] : []),
      ...(allowed(capabilities, 'notifications.read') ? [link('Notifications', `${base}/notifications`, '♧')] : [])
    ])
    add('advanced', 'Advanced testing', true, [
      ...(allowed(capabilities, 'sources.read') ? [link('Import and sync tests', `${base}/sources`, '⇧')] : []),
      ...(allowed(capabilities, 'catalog.read') ? [link('Suite releases', `${base}/releases`, '◇')] : [])
    ])
    if (application) add('application', 'Application administration', true, [
      ...(allowed(capabilities, 'members.read', 'permissions.read') ? [link('Application members', `${testing}/members`, '♧')] : []),
      link('Explain application access', `${testing}/explain-access`, '♧')
    ])
    add('project', 'Project administration', true, [
      ...(allowed(projectCapabilities, 'projects.read') ? [link('Project settings', `${base}/configuration`, '⚙')] : []),
      ...(allowed(projectCapabilities, 'members.read', 'permissions.read') ? [link('Project members', `${base}/members`, '♧'), link('Project permissions', `${base}/permissions`, '♧')] : []),
      ...(allowed(projectCapabilities, 'groups.read') ? [link('Project groups', `${base}/groups`, '♙')] : []),
      ...(allowed(projectCapabilities, 'roles.read') ? [link('Project roles', `${base}/roles`, '♧')] : []),
      link('Explain project access', `${base}/explain-access`, '♧')
    ])
  }
  add('installation', 'System administration', true, [
    ...(allowed(installation, 'users.read') ? [link('Users', '/settings/users', '♙')] : []),
    ...(allowed(installation, 'providers.read') ? [link('Identity providers', '/settings/providers', '♧')] : []),
    ...(allowed(installation, 'members.read', 'permissions.read') ? [link('System access', '/settings/access', '♧'), link('System permissions', '/settings/permissions', '♧')] : []),
    ...(allowed(installation, 'groups.read') ? [link('System groups', '/settings/groups', '♙')] : []),
    ...(allowed(installation, 'roles.read') ? [link('System roles', '/settings/roles', '♧')] : []),
    link('Explain installation access', '/account/access', '♧')
  ])
  return sections
}

export function projectNavigation(url: string): NavigationLink[] {
  return [{ label: 'Projects', path: '/projects', icon: '▦', exact: true, active: url.split(/[?#]/)[0] === '/projects' }]
}
