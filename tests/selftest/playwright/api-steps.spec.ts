import { test, expect } from './fixtures'

test('API workflow steps retain one registered logical result', { tag: '@STUDIO-API-STEPS-001' }, async ({ api }) => {
  const workspace = await test.step('Register an independent project and target through the API', () => api.scaffold())
  await test.step('Create and publish one native check', () => api.createCheck(workspace.path))
  const run = await test.step('Execute that registered check', () => api.run(workspace.path))
  await test.step('Read its result and evidence through separate API operations', async () => {
    const stored = (await api.call('GET', `${workspace.path}/runs/${run.id}`)).data
    const attempts = await api.call('GET', `${workspace.path}/runs/${run.id}/attempts`)
    const artifacts = await api.call('GET', `${workspace.path}/runs/${run.id}/artifacts`)
    expect(stored.counts.total).toBe(1)
    expect(stored.counts.completed).toBe(1)
    expect(stored.results).toHaveLength(1)
    expect(attempts.items).toHaveLength(1)
    expect(artifacts.items.length).toBeGreaterThan(0)
    expect((await api.call('GET', `${workspace.path}/runs`)).items).toHaveLength(1)
  })
})
