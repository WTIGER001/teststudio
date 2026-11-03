import assert from 'node:assert/strict'
import { test } from 'node:test'
import { applicationResources, findApplication, summarizeApplications } from './applications.utils.ts'

test('application views retain shared suites and keep environments assigned to one application', () => {
  const row = (id: string, data: object) => ({ id, revision: 1, updatedAt: '', data })
  const applications = [row('shop', { id: 'shop', name: 'Shop', description: '' }), row('billing', { id: 'billing', name: 'Billing', description: '' })]
  const environments = [row('shop-local', { id: 'shop-local', applicationId: 'shop' }), row('billing-local', { id: 'billing-local', applicationId: 'billing' })]
  const suites = [row('checkout', { applications: ['shop', 'billing'] }), row('unit', { applications: [] }), row('legacy', { applications: null, archived: true })]
  const before = JSON.stringify({ applications, environments, suites })
  const summary = summarizeApplications(applications as never, environments as never, suites as never)
  assert.deepEqual(summary.map(app => [app.id, app.environments, app.suites]), [['shop', 1, 1], ['billing', 1, 1]])
  assert.deepEqual(applicationResources('shop', environments as never, suites as never).environments.map(item => item.id), ['shop-local'])
  assert.deepEqual(applicationResources('billing', environments as never, suites as never).suites.map(item => item.id), ['checkout'])
  assert.equal(findApplication(applications as never, 'missing'), null)
  assert.equal(JSON.stringify({ applications, environments, suites }), before)
})
