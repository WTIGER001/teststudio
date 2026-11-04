export interface NavigationLink { label: string; path: string; icon: string; exact?: boolean; active: boolean }
export interface NavigationSection { id: string; title: string; collapsible: boolean; active: boolean; links: NavigationLink[] }
type Capabilities = Record<string, boolean>

export function navigationSections(project: string, capabilities: Capabilities, installation: Capabilities, url: string): NavigationSection[] {
  const path = url.split(/[?#]/)[0]
  const allowed = (scope: Capabilities, ...permissions: string[]) => permissions.every(permission => scope[`teststudio.${permission}`] === true)
  const link = (label: string, target: string, icon: string, exact = false, related: string[] = []): NavigationLink => ({ label, path: target, icon, exact, active: path === target || !exact && path.startsWith(target + '/') || related.some(prefix => path === prefix || path.startsWith(prefix + '/')) })
  const sections: NavigationSection[] = []
  const add = (id: string, title: string, collapsible: boolean, links: NavigationLink[]) => {
    if (links.length) sections.push({ id, title, collapsible, links, active: links.some(item => item.active) })
  }
  add('workspace', 'Workspace', false, [link('Projects', '/projects', '▦', true), link('Your account', '/account', '♙', true)])
  if (project) {
    const base = `/projects/${project}`
    add('testing', 'Testing', false, [
      ...(allowed(capabilities, 'projects.read') ? [link('Applications', `${base}/applications`, '▦')] : []),
      ...(allowed(capabilities, 'suites.read') ? [link('Suites', `${base}/suites`, '▤')] : []),
      link('Test catalog', `${base}/catalog`, '≡', false, [`${base}/tests`]), link('Test runs', `${base}/runs`, '▷'),
      link('Trends', `${base}/trends`, '▥')
    ])
    add('automation', 'Automation', true, [
      ...(allowed(capabilities, 'schedules.read') ? [link('Schedules', `${base}/schedules`, '◷')] : []),
      ...(allowed(capabilities, 'notifications.read') ? [link('Notifications', `${base}/notifications`, '♧')] : [])
    ])
    add('advanced', 'Advanced testing', true, [
      ...(allowed(capabilities, 'sources.read') ? [link('Import and sync tests', `${base}/sources`, '⇧')] : []),
      ...(allowed(capabilities, 'catalog.read') ? [link('Suite releases', `${base}/releases`, '◇')] : [])
    ])
    add('project', 'Project administration', true, [
      link('Project settings', `${base}/configuration`, '⚙'),
      ...(allowed(capabilities, 'members.read', 'permissions.read') ? [link('Project access', `${base}/members`, '♧'), link('Project permissions', `${base}/permissions`, '♧')] : []),
      ...(allowed(capabilities, 'groups.read') ? [link('Project groups', `${base}/groups`, '♙')] : []),
      ...(allowed(capabilities, 'roles.read') ? [link('Project roles', `${base}/roles`, '♧')] : []),
      link('Explain project access', `${base}/explain-access`, '♧')
    ])
  }
  add('installation', 'Installation administration', true, [
    ...(allowed(installation, 'users.read') ? [link('Users', '/settings/users', '♙')] : []),
    ...(allowed(installation, 'providers.read') ? [link('Identity providers', '/settings/providers', '♧')] : []),
    ...(allowed(installation, 'members.read', 'permissions.read') ? [link('Installation access', '/settings/access', '♧'), link('Installation permissions', '/settings/permissions', '♧')] : []),
    ...(allowed(installation, 'groups.read') ? [link('Installation groups', '/settings/groups', '♙')] : []),
    ...(allowed(installation, 'roles.read') ? [link('Installation roles', '/settings/roles', '♧')] : []),
    link('Explain installation access', '/account/access', '♧')
  ])
  return sections
}
