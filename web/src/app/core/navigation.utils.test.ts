import assert from 'node:assert/strict'
import { test } from 'node:test'
import { navigationSections } from './navigation.utils.ts'

test('navigation separates daily testing from administration and opens the current scope', () => {
  const permissions = Object.fromEntries(['projects.read', 'catalog.read', 'sources.read', 'schedules.read', 'notifications.read', 'members.read', 'permissions.read', 'groups.read', 'roles.read', 'users.read', 'providers.read'].map(permission => [`teststudio.${permission}`, true]))
  const catalog = navigationSections('demo', permissions, permissions, '/projects/demo/catalog')
  assert.deepEqual(catalog.map(section => section.title), ['Workspace', 'Testing', 'Project administration', 'Installation administration'])
  assert.deepEqual(catalog.filter(section => section.collapsible).map(section => section.active), [false, false])
  assert(catalog.find(section => section.id === 'testing')?.links.some(link => link.label === 'Test runs'))
  assert(catalog.find(section => section.id === 'testing')?.links.some(link => link.label === 'Applications' && link.path === '/projects/demo/applications'))
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
  assert(!navigationSections('demo', {}, {}, '/projects/demo/applications').flatMap(section => section.links).some(link => link.label === 'Applications'))
  assert.equal(viewer.find(section => section.id === 'workspace')?.active, false)
  assert.equal(viewer.find(section => section.id === 'installation')?.active, true)
})
