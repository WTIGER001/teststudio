import assert from 'node:assert/strict'
import { test } from 'node:test'
import { navigationSections, projectNavigation } from './navigation.utils.ts'

test('navigation separates daily testing from administration and opens the current scope', () => {
  const permissions = Object.fromEntries(['projects.read', 'catalog.read', 'sources.read', 'schedules.read', 'suites.read', 'notifications.read', 'members.read', 'permissions.read', 'groups.read', 'roles.read', 'users.read', 'providers.read'].map(permission => [`teststudio.${permission}`, true]))
  const catalog = navigationSections('demo', permissions, permissions, '/projects/demo/catalog')
  assert.deepEqual(catalog.map(section => section.title), ['All application testing', 'Project overviews', 'Automation', 'Advanced testing', 'Project administration', 'Installation administration'])
  assert.deepEqual(catalog.filter(section => section.collapsible).map(section => section.active), [false, false, false, false])
  assert.deepEqual(catalog.find(section => section.id === 'testing')?.links.map(link => link.label), ['Suites', 'Test catalog', 'Test runs', 'Trends'])
  assert.deepEqual(catalog.find(section => section.id === 'automation')?.links.map(link => link.label), ['Schedules', 'Notifications'])
  assert.deepEqual(catalog.find(section => section.id === 'advanced')?.links.map(link => link.label), ['Import and sync tests', 'Suite releases'])
  assert(catalog.find(section => section.id === 'overviews')?.links.some(link => link.label === 'All applications' && link.path === '/projects/demo/applications'))
  assert(!catalog.find(section => section.id === 'testing')?.links.some(link => link.path.startsWith('/settings/')))
  const settings = navigationSections('demo', permissions, permissions, '/projects/demo/roles/reviewer?tab=rules')
  assert.equal(settings.find(section => section.id === 'project')?.active, true)
  assert.equal(settings.find(section => section.id === 'installation')?.active, false)
  const installation = navigationSections('', permissions, permissions, '/settings/users/alex')
  assert.equal(installation.find(section => section.id === 'installation')?.active, true)
  assert(!installation.some(section => ['testing', 'project'].includes(section.id)))
})

test('navigation retains capability gates and exact account route matching', () => {
  const viewer = navigationSections('demo', { 'teststudio.catalog.read': true, 'teststudio.projects.read': true }, {}, '/account/access')
  const paths = viewer.flatMap(section => section.links.map(link => link.path))
  assert(paths.includes('/projects/demo/applications'))
  assert(paths.includes('/projects/demo/catalog'))
  assert(paths.includes('/projects/demo/releases'))
  for (const path of ['/settings/users', '/settings/access', '/settings/roles', '/projects/demo/groups', '/projects/demo/notifications', '/projects/demo/sources']) assert(!paths.includes(path))
  assert(!navigationSections('demo', {}, {}, '/projects/demo/applications').flatMap(section => section.links).some(link => link.label === 'All applications'))
  assert(!viewer.some(section => section.id === 'workspace'))
  assert(!paths.includes('/account'))
  assert.equal(viewer.find(section => section.id === 'installation')?.active, true)
})

test('creation routes retain catalog context and advanced destinations open their own group', () => {
  const permissions = { 'teststudio.catalog.read': true, 'teststudio.sources.read': true, 'teststudio.schedules.read': true }
  const created = navigationSections('demo', permissions, {}, '/projects/demo/tests/new/https?suite=health')
  assert.equal(created.find(section => section.id === 'testing')?.links.find(link => link.label === 'Test catalog')?.active, true)
  const sources = navigationSections('demo', permissions, {}, '/projects/demo/sources')
  assert.equal(sources.find(section => section.id === 'advanced')?.active, true)
  assert.equal(sources.find(section => section.id === 'automation')?.active, false)
  assert.equal(navigationSections('demo', {}, {}, '/projects/demo/suites').flatMap(section => section.links).some(link => link.label === 'Suites'), false)
})


test('application navigation stays inside its scope and separates project overviews', () => {
  const sections = navigationSections('demo', { 'teststudio.projects.read': true, 'teststudio.suites.read': true }, {}, '/projects/demo/applications/shop/runs/run-1')
  const testing = sections.find(section => section.id === 'testing')!
  assert.equal(testing.title, 'Application testing')
  assert.deepEqual(testing.links.map(link => link.label), ['Application overview', 'Suites', 'Test catalog', 'Test runs', 'Trends'])
  assert(testing.links.every(link => link.path.startsWith('/projects/demo/applications/shop')))
  assert.equal(testing.links.find(link => link.label === 'Test runs')?.active, true)
  assert.deepEqual(sections.find(section => section.id === 'overviews')?.links.map(link => link.path), ['/projects/demo/applications', '/projects/demo/probes'])
  const probes = navigationSections('demo', {}, {}, '/projects/demo/probes')
  assert.equal(probes.find(section => section.id === 'overviews')?.active, true)
  assert.equal(probes.find(section => section.id === 'testing')?.active, false)
})


test('project management stands above scope and account access stays in the global header', () => {
  assert.equal(projectNavigation('/projects')[0].active, true)
  assert.equal(projectNavigation('/projects/demo/catalog')[0].active, false)
  assert.deepEqual(projectNavigation('/account').map(link => link.path), ['/projects'])
  const sections = navigationSections('demo', {}, {}, '/projects/demo/catalog')
  assert(!sections.some(section => section.title === 'Workspace'))
  assert(!sections.flatMap(section => section.links).some(link => link.path === '/account'))
})
