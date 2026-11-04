import { execFile } from 'node:child_process'
import { promisify } from 'node:util'
import { createServer } from 'node:http'
import { test, expect, expectHTTPFailure } from './fixtures'

test('Investigate failure evidence and recover a lost rerun response', { tag: '@STUDIO-DIAGNOSTICS-001' }, async ({ page, api }, testInfo) => {
  const { project, path } = await api.scaffold()
  const media = `/tmp/teststudio-diagnostics-${project}`
  const generated = await promisify(execFile)(process.execPath, ['/suite/diagnostic-media.mjs', media], { timeout: 15000, maxBuffer: 4096 })
  expect(JSON.parse(generated.stdout).closed).toBe(true)
  const response = await page.request.post('http://127.0.0.1:8050/diagnostics/prepare', { data: { project } })
  expect(response.ok(), await response.text()).toBeTruthy()
  const { run } = await response.json()
  const base = `${path}/runs/${run}`
  await page.goto(`/projects/${project}/runs/${run}`)

  await test.step('Scope failed steps, timeout retries, logs and private screenshot access', async () => {
    const inspect = page.getByRole('button', { name: 'Inspect test BROWSER-1', exact: true })
    await inspect.click()
    await expect(page.getByRole('heading', { name: 'Investigate a test', exact: true })).toBeFocused()
    await expect(page.getByLabel('Test evidence scope', { exact: true })).toHaveValue('BROWSER-1')
    const history = page.getByRole('region', { name: 'Attempt history', exact: true })
    await expect(history).toContainText('Confirmation heading did not match')
    await expect(history).toContainText('Test timeout exceeded')
    expect(await history.locator('details.attempt-steps').first().evaluate(element => (element as HTMLDetailsElement).open)).toBeTruthy()
    await page.getByLabel('Attempt evidence scope', { exact: true }).selectOption({ label: 'Attempt 1' })
    await expect(history).toContainText('Verify confirmation heading')
    await expect(history).not.toContainText('Test timeout exceeded')
    const logs = page.getByRole('region', { name: 'Latest run log segments', exact: true })
    await expect(logs).toContainText('Checkout attempt one diagnostic')
    await expect(logs).not.toContainText('Checkout attempt two timeout')
    await expect(logs).not.toContainText('Unrelated account diagnostic')
    const artifacts = page.getByRole('region', { name: 'Run artifacts', exact: true })
    const screenshot = artifacts.getByRole('article', { name: 'checkout-one.png', exact: true })
    await screenshot.getByRole('button', { name: 'View', exact: true }).click()
    await expect(page.getByRole('region', { name: 'checkout-one.png', exact: true }).getByRole('heading', { name: 'checkout-one.png', exact: true })).toBeFocused()
    await expect(page.getByAltText('checkout-one.png', { exact: true })).toBeVisible()
    await page.getByRole('button', { name: 'Close preview', exact: true }).click()
    await expect(screenshot.getByRole('button', { name: 'View', exact: true })).toBeFocused()
    for (const [filename, state] of [['upload-failed.txt', 'upload-failed'], ['missing.txt', 'missing'], ['expired.txt', 'expired']]) {
      const row = artifacts.getByRole('article', { name: filename, exact: true })
      await expect(row).toContainText(state)
      await expect(row.getByRole('button', { name: 'View', exact: true })).toHaveCount(0)
    }
    await expect(page.getByRole('region', { name: 'Run summary', exact: true })).toContainText('incomplete')
    await expect(page.getByRole('table', { name: 'One final result for each completed test', exact: true })).toContainText('timed-out')
    await page.getByRole('button', { name: 'Refresh', exact: true }).click()
    await expect(page.getByLabel('Test evidence scope', { exact: true })).toHaveValue('BROWSER-1')
    await expect(page.getByLabel('Attempt evidence scope', { exact: true }).locator('option:checked')).toHaveText('Attempt 1')
    await page.getByLabel('Color theme').selectOption('dark')
    await page.setViewportSize({ width: 390, height: 844 })
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy()
    const scopeBox = await page.getByRole('region', { name: 'Investigate a test', exact: true }).boundingBox()
    const selectBox = await page.getByLabel('Test evidence scope', { exact: true }).boundingBox()
    const stepsBox = await history.locator('.attempt-step-list').boundingBox()
    expect(stepsBox?.width).toBeGreaterThan(200)
    expect(selectBox && scopeBox && selectBox.x + selectBox.width <= scopeBox.x + scopeBox.width).toBeTruthy()

    await testInfo.attach('scoped-failure-evidence-mobile', { body: await page.screenshot({ fullPage: true }), contentType: 'image/png' })
  })

  await test.step('Hide rejected evidence and keep selection while access recovers', async () => {
    const logsPath = `${base}/logs?test=BROWSER-1&attempt=1&tail=true&limit=50`
    await page.route(`**${base}/logs?**`, route => route.fulfill({ status: 403, contentType: 'application/json', body: JSON.stringify({ message: 'Fixture access revoked' }) }))
    await expectHTTPFailure(page, 'GET', logsPath, 403, () => page.getByRole('button', { name: 'Refresh logs', exact: true }).click())
    await expect(page.getByRole('alert').filter({ hasText: 'Access to this evidence is no longer allowed' })).toBeVisible()
    await expect(page.getByRole('region', { name: 'Latest run log segments', exact: true })).not.toContainText('Checkout attempt one diagnostic')
    await expect(page.getByLabel('Test evidence scope', { exact: true })).toHaveValue('BROWSER-1')
    await page.unroute(`**${base}/logs?**`)
    await page.getByRole('button', { name: 'Refresh logs', exact: true }).click()
    await expect(page.getByRole('region', { name: 'Latest run log segments', exact: true })).toContainText('Checkout attempt one diagnostic')
  })

  let slow = true
  const target = createServer((request, response) => { const timer = setTimeout(() => response.end('ready'), slow ? 1500 : 0); response.on('close', () => clearTimeout(timer)) })
  await new Promise<void>(resolve => target.listen(0, '127.0.0.1', resolve))
  try {
    const address = target.address()
    if (!address || typeof address === 'string') throw new Error('Local rerun fixture did not start')
    await api.call('PUT', `${path}/environments/local`, { id: 'local', name: 'Rerun response fixture', applicationId: 'app', stage: 'local', endpoints: { api: `http://127.0.0.1:${address.port}` } }, 1)
    await api.call('POST', `${path}/suites/health/tests`, { id: 'PROBE-1', title: 'Quick availability', category: 'Recovery', applicationId: 'app', type: 'https', runner: 'http', state: 'enabled', safety: 'safe', rationale: 'Reads the new local response fixture', timeoutSeconds: 1, dependencies: [], requiredBindings: [], tags: [], check: { endpoint: 'api', path: '/', method: 'GET', statusCodes: [200] } })
    const draft = await api.call('POST', `${path}/suites/health/catalog-drafts`, {})
    await api.call('POST', `${path}/suites/health/catalog-drafts/${draft.id}/publish`, { activate: true }, draft.revision)
    const parent = await api.call('POST', `${path}/runs`, { suiteId: 'health', environmentId: 'local', timeoutSeconds: 30 })
    await expect.poll(async () => (await api.call('GET', `${path}/runs/${parent.id}`)).data.outcome).toBe('timed-out')
    await page.goto(`/projects/${project}/runs/${parent.id}`)
    const rerunPath = `${path}/runs/${parent.id}/rerun-failures`
    slow = false
    let child = ''
    await page.route(`**${rerunPath}`, async route => {
      const actual = await route.fetch()
      expect(actual.ok()).toBeTruthy()
      child = (await actual.json()).id
      await route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ message: 'Fixture lost response after server commit' }) })
    })
    await expectHTTPFailure(page, 'POST', rerunPath, 503, () => page.getByRole('button', { name: 'Rerun failures', exact: true }).click())
    await expect(page.getByRole('alert').filter({ hasText: 'reuse the same request' })).toBeVisible()
    await page.getByRole('button', { name: 'Refresh', exact: true }).click()
    await expect(page.getByRole('alert').filter({ hasText: 'reuse the same request' })).toBeVisible()
    await page.unroute(`**${rerunPath}`)
    await page.getByRole('button', { name: 'Rerun failures', exact: true }).click()
    await expect(page).toHaveURL(new RegExp(`${child}$`))
    await expect(page.getByRole('link', { name: 'Original run', exact: true })).toBeVisible()
    await expect.poll(async () => (await api.call('GET', `${path}/runs/${child}`)).data.outcome).toBe('passed')
    expect((await api.call('GET', `${path}/runs`)).items).toHaveLength(3)
    expect((await api.call('GET', `${path}/runs/${parent.id}`)).data.outcome).toBe('timed-out')
  } finally { target.closeAllConnections(); await new Promise<void>((resolve, reject) => target.close(error => error ? reject(error) : resolve())) }
})
