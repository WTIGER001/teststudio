import { createServer } from 'node:http'
import { test, expect } from './fixtures'

test('Stacked result trends, selection gaps, and rankings', { tag: '@STUDIO-TRENDS-001' }, async ({ page, api }, testInfo) => {
  let mode = 'pass', calls = 0
  const target = createServer((request, response) => {
    const check = request.url === '/a'
    const status = check && (mode === 'fail' || (mode === 'retry' && calls++ === 0)) ? 500 : 200
    const timer = setTimeout(() => { response.statusCode = status; response.end('up') }, check && mode === 'timeout' ? 2000 : 0)
    response.on('close', () => clearTimeout(timer))
  })
  await new Promise<void>(resolve => target.listen(0, '127.0.0.1', resolve))
  try {
    const address = target.address()
    if (!address || typeof address === 'string') throw new Error('Trend target did not start')
    const { project, path } = await api.scaffold()
    await api.call('PUT', `${path}/environments/local`, { id: 'local', name: 'Trend target', applicationId: 'app', stage: 'local', endpoints: { api: `http://127.0.0.1:${address.port}` } }, 1)
    for (const [id, title, category, route] of [['CHECK-A', 'Availability check', 'Availability', '/a'], ['CHECK-B', 'Security check', 'Security', '/b']]) {
      await api.call('POST', `${path}/suites/health/tests`, { id, title, category, applicationId: 'app', type: 'https', runner: 'http', state: 'enabled', safety: 'safe', rationale: 'Reads an isolated loopback target', timeoutSeconds: 1, dependencies: [], requiredBindings: [], tags: [], check: { endpoint: 'api', path: route, method: 'GET', statusCodes: [200] } })
    }
    const draft = await api.call('POST', `${path}/suites/health/catalog-drafts`, {})
    await api.call('POST', `${path}/suites/health/catalog-drafts/${draft.id}/publish`, { activate: true }, draft.revision)
    const history: any[] = []
    await test.step('Create real passed, failed, timed-out, flaky, and incomplete history', async () => {
      for (const scenario of ['pass', 'fail', 'timeout', 'retry', 'deadline']) {
        mode = scenario === 'deadline' ? 'timeout' : scenario; calls = 0
        const full = scenario === 'pass' || scenario === 'deadline'
        const created = await api.call('POST', `${path}/runs`, { suiteId: 'health', environmentId: 'local', timeoutSeconds: scenario === 'deadline' ? 1 : 30, retries: scenario === 'retry' ? 1 : 0, ...(full ? {} : { testIds: ['CHECK-A'] }) })
        let result: any
        await expect.poll(async () => { result = (await api.call('GET', `${path}/runs/${created.id}`)).data; return ['completed', 'timed-out'].includes(result.state) }, { timeout: 10000 }).toBeTruthy()
        history.push(result)
      }
      expect(history.map(run => run.outcome)).toEqual(['passed', 'failed', 'timed-out', 'flaky', 'timed-out'])
    })

    await test.step('Read stacked state counts, keyboard values, and failure rankings', async () => {
      await page.goto(`/projects/${project}/trends`)
      await expect(page.getByRole('status')).toContainText('5 result-bearing runs · 7 logical results')
      await expect(page.locator('svg .trend-area')).toHaveCount(6)
      await page.locator('svg a').nth(2).focus()
      await expect(page.getByRole('tooltip')).toContainText('1 timed out')
      await page.getByText('View chart values', { exact: true }).click()
      const values = page.getByRole('table', { name: 'Accessible trend values, oldest first. Gaps have no measured counts.' })
      await expect(values.getByRole('row')).toHaveCount(6)
      await expect(values.getByRole('row').last().getByRole('cell').last()).toHaveText('2')
      await expect(page.getByRole('region', { name: 'Most failed tests' })).toContainText('Availability check')
      await expect(page.getByRole('region', { name: 'Most volatile tests' })).toContainText('4 / 4')
      await page.getByLabel('Color theme').selectOption('light')
      await testInfo.attach('trends-desktop-light', { body: await page.locator('ts-trend-chart').screenshot(), contentType: 'image/png' })
    })

    await test.step('Keep test and category gaps visible and preserve URL filters', async () => {
      await page.getByLabel('Suite identifier', { exact: true }).fill('health')
      await page.getByLabel('Permanent test identifier', { exact: true }).fill('CHECK-B')
      await page.getByRole('button', { name: 'Apply scope' }).click()
      await expect(page.getByRole('status')).toContainText('2 result-bearing runs · 2 logical results · 3 selection gaps')
      expect((await page.locator('svg .state-passed').getAttribute('d'))?.match(/M/g)?.length).toBe(2)
      await page.reload()
      await expect(page.getByLabel('Permanent test identifier', { exact: true })).toHaveValue('CHECK-B')
      await page.getByLabel('Permanent test identifier', { exact: true }).fill('')
      await page.getByLabel('Category', { exact: true }).fill('Security')
      await page.getByRole('button', { name: 'Apply scope' }).click()
      await expect(page.getByRole('heading', { name: 'Category: Security', exact: true })).toBeVisible()
      await expect(page.getByRole('status')).toContainText('3 selection gaps')
      await page.getByLabel('Color theme').selectOption('dark')
      await page.setViewportSize({ width: 390, height: 844 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy()
      await testInfo.attach('trends-mobile-dark', { body: await page.screenshot({ fullPage: true }), contentType: 'image/png' })
    })

    await test.step('Open test history from rankings and reach its original run', async () => {
      await page.getByLabel('Category', { exact: true }).fill('')
      await page.getByRole('button', { name: 'Apply scope' }).click()
      await page.getByRole('region', { name: 'Most failed tests' }).getByRole('link', { name: 'Availability check', exact: true }).click()
      await expect(page.getByLabel('Permanent test identifier', { exact: true })).toHaveValue('CHECK-A')
      await expect(page.getByRole('heading', { name: 'Test health / CHECK-A', exact: true })).toBeVisible()
      await page.locator('svg a').first().click()
      await expect(page.getByRole('heading', { name: 'Run details', exact: true })).toBeVisible()
      await page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'Trends', exact: true }).click()
    })

    await test.step('Keep unavailable history distinct from an empty scope', async () => {
      await page.evaluate(() => {
        const original = window.fetch.bind(window)
        window.fetch = (...args) => String(args[0]).includes('/trends?') ? Promise.reject(new Error('Trend service unavailable during verification.')) : original(...args)
      })
      await page.getByRole('button', { name: 'Refresh trends', exact: true }).click()
      await expect(page.getByRole('alert')).toContainText('Trend service unavailable during verification.')
      await expect(page.locator('ts-trend-chart')).toHaveCount(0)
      await expect(page.getByRole('heading', { name: 'No finished runs in this history page.', exact: true })).toHaveCount(0)
      await page.reload()
      await page.getByLabel('Environment identifier', { exact: true }).fill('unrecorded-environment')
      await page.getByRole('button', { name: 'Apply scope' }).click()
      await expect(page.getByRole('heading', { name: 'No finished runs in this history page.', exact: true })).toBeVisible()
      await expect(page.getByRole('alert')).toHaveCount(0)
      await page.getByRole('button', { name: 'Help', exact: true }).click()
      await expect(page.getByRole('dialog', { name: 'Trends help', exact: true })).toContainText('Logical counts and gaps')
      await page.keyboard.press('Escape')
    })
  } finally {
    target.closeAllConnections()
    await new Promise<void>((resolve, reject) => target.close(error => error ? reject(error) : resolve()))
  }
})
