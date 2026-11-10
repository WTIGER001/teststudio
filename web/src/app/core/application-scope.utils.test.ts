import assert from 'node:assert/strict'
import { test } from 'node:test'
import { applicationFromURL, applicationSuites, applicationEnvironments, scopePath, switchApplicationPath } from './application-scope.utils.ts'

test('application switches preserve the working module and discard old detail context', () => {
  assert.equal(applicationFromURL('/projects/p/applications/shop/catalog?suite=one'), 'shop')
  assert.equal(applicationFromURL('/projects/p/applications'), '')
  assert.deepEqual(switchApplicationPath('p', 'warehouse', '/projects/p/applications/shop/runs/run-1?test=one'), ['/projects', 'p', 'applications', 'warehouse', 'runs'])
  assert.deepEqual(switchApplicationPath('p', '', '/projects/p/applications/shop/catalog?suite=one'), ['/projects', 'p', 'catalog'])
  assert.deepEqual(switchApplicationPath('p', 'shop', '/projects/p/configuration'), ['/projects', 'p', 'applications', 'shop', 'configuration'])
  assert.deepEqual(scopePath('p', 'shop', 'tests', 'new'), ['/projects', 'p', 'applications', 'shop', 'tests', 'new'])
  assert.deepEqual(scopePath('p', 'shop', 'configuration'), ['/projects', 'p', 'applications', 'shop', 'configuration'])
})

test('application scope excludes shared suites and only its environments', () => {
  const suites = [{ data: { applications: ['shop'] } }, { data: { applications: ['shop', 'warehouse'] } }, { data: { applications: [] } }]
  assert.deepEqual(applicationSuites(suites, 'shop'), suites.slice(0, 1))
  assert.deepEqual(applicationSuites(suites, ''), suites)
  const environments = [{ data: { applicationId: 'shop' } }, { data: { applicationId: 'warehouse' } }]
  assert.deepEqual(applicationEnvironments(environments, 'warehouse'), [environments[1]])
})
