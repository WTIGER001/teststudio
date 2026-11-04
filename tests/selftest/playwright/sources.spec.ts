import { test, expect, expectHTTPFailure } from './fixtures'
import { createHash } from 'node:crypto'

function nativeYAML(title = 'Uploaded health check') {
  return `schemaVersion: teststudio.catalog/v1
tests:
  - id: UPLOADED-HEALTH
    title: ${title}
    description: An independently written native check
    applicationId: app
    category: Availability
    tags: [upload]
    type: https
    runner: http
    state: enabled
    safety: safe
    rationale: Reads the isolated TestStudio health endpoint
    dependencies: []
    requiredBindings: []
    timeoutSeconds: 10
    check:
      endpoint: api
      path: /healthz
      method: GET
      statusCodes: [200]
`
}

test('Native source upload, atomic publication, immutable snapshots, and stale review', { tag: '@STUDIO-SOURCES-001' }, async ({ page, api }, testInfo) => {
  const { project, path } = await api.scaffold()
  const base = `${path}/suites/health`
  const readSuite = async () => (await api.call('GET', `${path}/suites`)).items.find((item: { id: string }) => item.id === 'health')
  await page.goto(`/projects/${project}/sources`)
  await page.getByLabel('Color theme').selectOption('light')
  await page.getByLabel('Suite for sources', { exact: true }).selectOption('health')
  await expect(page.getByRole('heading', { name: 'No uploads in this suite.' })).toBeVisible()
  await page.getByRole('button', { name: '+ Upload tests', exact: true }).click()
  await page.getByLabel('Source file', { exact: true }).setInputFiles({ name: 'health.yaml', mimeType: 'application/yaml', buffer: Buffer.from(nativeYAML()) })
  await expect(page.getByLabel('Source format', { exact: true })).toHaveValue('yaml')
  const staged = page.waitForResponse(response => response.request().method() === 'POST' && response.url().endsWith('/source-uploads'))
  await page.getByRole('button', { name: 'Validate source and preview', exact: true }).click()
  const first = await (await staged).json()
  const session = await (await page.request.get('/auth/session')).json()
  const detached = await page.request.post(`${base}/catalog-imports/${first.plan.id}/apply`, { data: {}, headers: { Origin: 'http://127.0.0.1:8040', 'X-CSRF-Token': session.session.csrfToken, 'If-Match': '"1"' } })
  expect(detached.status()).toBe(403)
  expect(first.upload.data.digest).toBe(`sha256:${createHash('sha256').update(nativeYAML()).digest('hex')}`)
  const review = page.getByRole('region', { name: 'Source review', exact: true })
  await expect(review.getByRole('cell', { name: 'UPLOADED-HEALTH', exact: false })).toBeVisible()
  await expect(review.getByRole('cell', { name: 'create', exact: true })).toBeVisible()
  expect((await page.request.get(`${base}/tests/UPLOADED-HEALTH`)).status()).toBe(404)
  await review.getByText('Inspect changes', { exact: true }).click()
  await expect(review.getByRole('heading', { name: 'After', exact: true })).toBeVisible()
  await expect(review.getByRole('checkbox', { name: 'Activate this release for future runs' })).not.toBeChecked()
  await testInfo.attach('source-review-light', { body: await page.screenshot({ fullPage: true }), contentType: 'image/png' })
  await review.getByRole('button', { name: 'Publish source release', exact: true }).click()
  await expect(page.getByRole('status').filter({ hasText: 'Source release published. No tests were started.' })).toBeVisible()
  const published = await api.call('GET', `${base}/source-uploads/${first.upload.id}`)
  expect(published.upload.data.state).toBe('published')
  const firstRelease = published.upload.data.releaseId
  expect((await api.call('GET', `${base}/releases/${firstRelease}`)).data.catalog.artifact.sourceSnapshotId).toBe(first.upload.id)
  expect((await readSuite()).data.activeReleaseId).toBeUndefined()
  expect((await api.call('GET', `${path}/runs`)).items).toEqual([])
  await review.getByRole('button', { name: 'View retained source', exact: true }).click()
  await expect(page.getByLabel('Retained source document', { exact: true })).toHaveValue(nativeYAML())

  await test.step('Preserve the active release after an intervening catalog edit', async () => {
    const suite = await readSuite()
    await api.call('POST', `${base}/releases/${firstRelease}/activate`, {}, suite.revision)
    await page.getByRole('button', { name: '+ Upload tests', exact: true }).click()
    await page.getByLabel('Source document', { exact: true }).fill(nativeYAML('Revised source health'))
    const response = page.waitForResponse(response => response.request().method() === 'POST' && response.url().endsWith('/source-uploads'))
    await page.getByRole('button', { name: 'Validate source and preview', exact: true }).click()
    const second = await (await response).json()
    await expect(review.getByRole('cell', { name: 'update', exact: true })).toBeVisible()
    const test = await api.call('GET', `${base}/tests/UPLOADED-HEALTH`)
    await api.call('PUT', `${base}/tests/UPLOADED-HEALTH`, { ...test.data, title: 'Concurrent catalog edit' }, test.revision)
    await expectHTTPFailure(page, 'POST', `${base}/source-uploads/${second.upload.id}/publish`, 409, () => review.getByRole('button', { name: 'Publish source release', exact: true }).click())
    await expect(page.getByRole('alert')).toContainText('Review current data and validate the source again.')
    expect((await readSuite()).data.activeReleaseId).toBe(firstRelease)
    expect((await api.call('GET', `${base}/tests/UPLOADED-HEALTH`)).data.title).toBe('Concurrent catalog edit')
    expect((await api.call('GET', `${base}/releases/${firstRelease}`)).data.catalog.tests[0].title).toBe('Uploaded health check')
  })

  await test.step('Publish a fresh preview and execute it through normal run contracts', async () => {
    await page.getByRole('button', { name: 'Validate source and preview', exact: true }).click()
    await review.getByRole('checkbox', { name: 'Activate this release for future runs' }).check()
    await review.getByRole('button', { name: 'Publish source release', exact: true }).click()
    await expect(page.getByRole('status').filter({ hasText: 'Source release published.' })).toBeVisible()
    expect((await api.call('GET', `${path}/runs`)).items).toEqual([])
    const run = await api.run(path)
    expect(run.tests[0].id).toBe('UPLOADED-HEALTH')
    const source = await api.call('GET', `${base}/source-uploads/${first.upload.id}/content`)
    expect(source.content).toBe(nativeYAML())
  })

  await test.step('Keep malformed source diagnostics, mobile layout, and contextual help', async () => {
    await page.getByRole('button', { name: '+ Upload tests', exact: true }).click()
    await page.getByLabel('Source document', { exact: true }).fill('schemaVersion: teststudio.catalog/v1\ntests: []\ntests: []\n')
    await page.getByRole('button', { name: 'Validate source and preview', exact: true }).click()
    await expect(review.getByText(/source contains duplicate object keys/)).toBeVisible()
    await expect(review.getByRole('button', { name: 'Publish source release', exact: true })).toHaveCount(0)
    await page.getByLabel('Color theme').selectOption('dark')
    await page.setViewportSize({ width: 390, height: 844 })
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy()
    await page.getByRole('button', { name: 'Help', exact: true }).click()
    const help = page.getByRole('dialog', { name: 'Import and sync tests help', exact: true })
    await expect(help).toBeVisible()
    for (const title of ['Introduction', 'Concepts', 'Tutorials']) await expect(help.getByRole('heading', { name: title, exact: true })).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(page.getByRole('button', { name: 'Help', exact: true })).toBeFocused()
    await testInfo.attach('sources-mobile-dark', { body: await page.screenshot({ fullPage: true }), contentType: 'image/png' })
    await page.reload()
    await page.getByLabel('Suite for sources', { exact: true }).selectOption('health')
    await expect(page.getByRole('region', { name: 'Upload history', exact: true }).getByRole('cell', { name: 'published', exact: true })).toHaveCount(2)
  })
})
