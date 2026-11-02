import { createServer } from 'node:http'
import { test, expect } from './fixtures'

test('Timeout status and failure rerun', { tag: '@STUDIO-TIMEOUT-001' }, async ({ page, api }, testInfo) => {
  let slow = true
  const target = createServer((request, response) => {
    const timer = setTimeout(() => response.end('up'), slow ? 1500 : 0)
    response.on('close', () => clearTimeout(timer))
  })
  await new Promise<void>(resolve => target.listen(0, '127.0.0.1', resolve))
  try {
    const address = target.address()
    if (!address || typeof address === 'string') throw new Error('Local timeout target did not start')
    const { project, path } = await api.scaffold()
    await api.call('PUT', `${path}/environments/local`, { id: 'local', name: 'Timeout test target', applicationId: 'app', stage: 'local', endpoints: { api: `http://127.0.0.1:${address.port}` } }, 1)
    await api.call('POST', `${path}/suites/health/tests`, {
      id: 'SLOW-1', title: 'Bounded response', category: 'Timing', applicationId: 'app', type: 'https', runner: 'http', state: 'enabled', safety: 'safe', rationale: 'Reads an isolated loopback target', timeoutSeconds: 1,
      dependencies: [], requiredBindings: [], tags: [], check: { endpoint: 'api', path: '/health', method: 'GET', statusCodes: [200] }
    })
    const draft = await api.call('POST', `${path}/suites/health/catalog-drafts`, {})
    await api.call('POST', `${path}/suites/health/catalog-drafts/${draft.id}/publish`, { activate: true }, draft.revision)
    const parent = await api.call('POST', `${path}/runs`, { suiteId: 'health', environmentId: 'local', timeoutSeconds: 30 })

    await test.step('Show timeout counts and retained attempts separately from failures', async () => {
      await expect.poll(async () => (await api.call('GET', `${path}/runs/${parent.id}`)).data.outcome).toBe('timed-out')
      const result = (await api.call('GET', `${path}/runs/${parent.id}`)).data
      expect(result.state).toBe('completed')
      expect(result.counts).toMatchObject({ timedOut: 1, failed: 0, completed: 1 })
      expect(result.results[0]).toMatchObject({ state: 'timed-out', failureKind: 'timeout', timing: { censored: true } })
      await page.goto(`/projects/${project}/runs/${parent.id}`)
      await expect(page.getByRole('region', { name: 'Run summary' })).toContainText('0 failed · 1 timed out')
      await expect(page.getByRole('table', { name: 'One final result for each completed test' }).getByRole('cell', { name: 'timed-out', exact: true })).toBeVisible()
      await expect(page.getByRole('button', { name: 'Rerun failures', exact: true })).toBeVisible()
      await page.getByLabel('Color theme').selectOption('dark')
      await page.setViewportSize({ width: 390, height: 844 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy()
      await testInfo.attach('timeout-mobile-dark', { body: await page.screenshot({ fullPage: true }), contentType: 'image/png' })
    })

    await test.step('Rerun the timed-out test and keep its original history', async () => {
      slow = false
      await page.getByRole('button', { name: 'Rerun failures', exact: true }).click()
      await expect(page).not.toHaveURL(new RegExp(`${parent.id}$`))
      const childID = new URL(page.url()).pathname.split('/').at(-1)
      await expect.poll(async () => (await api.call('GET', `${path}/runs/${childID}`)).data.outcome).toBe('passed')
      const child = (await api.call('GET', `${path}/runs/${childID}`)).data
      expect(child.request).toMatchObject({ parentRunId: parent.id, testIds: ['SLOW-1'] })
      expect(child.release.id).toBe(parent.data.release.id)
      expect(child.counts).toMatchObject({ timedOut: 0, failed: 0, passed: 1 })
      const original = (await api.call('GET', `${path}/runs/${parent.id}`)).data
      expect(original.results[0].state).toBe('timed-out')
      await page.getByRole('link', { name: '← Run history' }).click()
      const row = page.getByRole('row').filter({ has: page.getByRole('link', { name: parent.id, exact: true }) })
      await expect(row).toContainText('1 timed out')
      await expect(row).toContainText('timed-out')
    })
  } finally {
    target.closeAllConnections()
    await new Promise<void>((resolve, reject) => target.close(error => error ? reject(error) : resolve()))
  }
})
