import { test, expect } from './fixtures'

test('Theme persistence and mobile layout', { tag: '@STUDIO-UI-001' }, async ({ page, api }, testInfo) => {
  await test.step('Use dark mode on a narrow viewport', async () => {
    await page.getByLabel('Color theme').selectOption('dark')
    await page.setViewportSize({ width: 390, height: 844 })
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy()
    await testInfo.attach('mobile-dark', { body: await page.screenshot(), contentType: 'image/png' })
  })
  await test.step('Keep light mode after reload', async () => {
    await page.getByLabel('Color theme').selectOption('light')
    await page.reload()
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
    await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
  })
})

test('Publish a catalog and inspect an executed check', { tag: '@STUDIO-CATALOG-001' }, async ({ page, api }, testInfo) => {
  const { project } = await api.scaffold()
  await test.step('Create and publish an HTTP test from the catalog UI', async () => {
    await page.goto(`/projects/${project}/catalog`)
    await page.getByRole('button', { name: '+ Create test' }).click()
    await page.getByLabel('Permanent identifier').fill('SELF-CHECK')
    await page.getByLabel('Title', { exact: true }).fill('TestStudio health')
    await page.getByLabel('Production-safety label').selectOption('safe')
    await page.getByLabel('Safety rationale').fill('Reads this isolated TestStudio instance')
    await page.getByRole('button', { name: 'Save draft changes' }).click()
    await expect(page.getByRole('status')).toContainText('Draft changes saved')
    await page.getByRole('button', { name: 'Validate and preview' }).click()
    await page.getByRole('button', { name: 'Publish and activate' }).click()
    await expect(page.getByRole('status')).toContainText('Release published')
  })
  await test.step('Start a run and inspect retained evidence', async () => {
    await page.goto(`/projects/${project}/runs`)
    await page.getByRole('button', { name: '+ New run' }).click()
    const form = page.getByRole('region', { name: 'Start a run' })
    await form.getByLabel('Suite', { exact: true }).selectOption('health')
    await form.getByLabel('Target environment').selectOption('local')
    await form.getByLabel('Run time limit (seconds)').fill('60')
    await form.getByRole('button', { name: 'Estimate timing' }).click()
    await expect(form.getByRole('status')).toContainText('0 / 1 timing baselines ready')
    await form.getByRole('button', { name: 'Start run', exact: true }).click()
    const results = page.getByRole('region', { name: 'Final test results' })
    await expect(results.getByRole('cell', { name: 'passed', exact: true })).toBeVisible({ timeout: 20000 })
    await expect(results).toContainText('Functional: passed')
    await expect(page.getByRole('heading', { name: 'Attempt history' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Run artifacts' })).toBeVisible()
    await testInfo.attach('executed-native-check', { body: await page.screenshot({ fullPage: true }), contentType: 'image/png' })
  })
})

test('Reset timing history without changing the captured result', { tag: '@STUDIO-TIMING-001' }, async ({ page, api }) => {
  const { project, path } = await api.scaffold()
  await api.createCheck(path)
  let run: any
  await test.step('Build a comparable baseline through normal runs', async () => { for (let i = 0; i < 6; i++) run = await api.run(path) })
  const original = run.timing
  expect(original.tests['SELF-CHECK'].sampleCount).toBe(5)
  await page.goto(`/projects/${project}/runs/${run.id}`)
  await page.getByText('Manage comparable timing history', { exact: true }).click()
  await page.getByRole('button', { name: 'SELF-CHECK baseline' }).click()
  await expect(page.getByRole('heading', { name: 'SELF-CHECK · current baseline' })).toBeFocused()
  await page.getByLabel('Reason to reset timing history').fill('Verify a deliberate baseline reset')
  await page.getByRole('button', { name: 'Reset timing history', exact: true }).click()
  await expect(page.getByRole('status').filter({ hasText: /^Baseline reset\./ })).toBeVisible()
  await expect(page.getByText(/0 stored clean observations/)).toBeVisible()
  const after = (await api.call('GET', `${path}/runs/${run.id}`)).data
  expect(after.timing).toEqual(original)
  expect(after.outcome).toBe('passed')
  const estimate = await api.call('POST', `${path}/runs/estimate`, { suiteId: 'health', environmentId: 'local' })
  expect(estimate.readyTests).toBe(0)
  expect(estimate.tests['SELF-CHECK'].sampleCount).toBe(0)
})
