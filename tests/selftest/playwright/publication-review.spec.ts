import { test, expect, expectHTTPFailure } from './fixtures'

test('Review catalog changes, reject staleness, and recover a lost publication response', { tag: '@STUDIO-PUBLISH-001' }, async ({ page, api }, testInfo) => {
  const { project, path } = await api.scaffold()
  await api.createCheck(path)
  const original = await api.run(path)
  const suite = `${path}/suites/health`
  const current = (await api.call('GET', `${suite}/tests/SELF-CHECK`)).data
  await api.call('PUT', `${suite}/tests/SELF-CHECK`, { ...current, title: 'Reviewed health check', rationale: 'New read-only rationale' }, 1)
  await api.call('POST', `${path}/schedules`, { id: 'follow', name: 'Follow active catalog', enabled: false, kind: 'interval', intervalSeconds: 3600, timezone: 'UTC', overlap: 'skip', request: { suiteId: 'health', environmentId: 'local', timeoutSeconds: 60 } })
  await api.call('POST', `${path}/schedules`, { id: 'pinned', name: 'Keep original release', enabled: false, kind: 'interval', intervalSeconds: 3600, timezone: 'UTC', overlap: 'skip', request: { suiteId: 'health', environmentId: 'local', releaseId: original.release.id, timeoutSeconds: 60 } })
  await page.goto(`/projects/${project}/catalog`)
  const firstReview = page.waitForResponse(response => response.url().endsWith('/publication-reviews') && response.request().method() === 'POST')
  await page.getByRole('button', { name: 'Validate and preview', exact: true }).click()
  let reviewed = await (await firstReview).json()
  const preview = page.getByRole('region', { name: 'Release preview', exact: true })
  await expect(preview.getByRole('heading', { name: 'Release preview', exact: true })).toBeFocused()
  const change = preview.getByRole('article', { name: 'SELF-CHECK publication change', exact: true })
  await expect(change).toContainText('Reviewed health check')
  await expect(change).toContainText('changed')
  await expect(change).toContainText('rationale')
  await expect(change).toContainText('Safety: safe → safe')
  await expect(preview).toContainText('Follow active catalog · Disabled · Follows active release')
  await expect(preview).toContainText('Keep original release · Disabled · Pinned release')
  const before = (await api.call('GET', `${suite}/tests/SELF-CHECK`)).data
  await api.call('PUT', `${suite}/tests/SELF-CHECK`, { ...before, title: 'Concurrent health check' }, 2)
  let publicationPath = `${suite}/catalog-drafts/${reviewed.draftId}/publish`
  await expectHTTPFailure(page, 'POST', publicationPath, 409, () => preview.getByRole('button', { name: 'Publish and activate', exact: true }).click())
  await expect(preview.getByText(/This review no longer matches/)).toBeVisible()
  await expect(preview.getByRole('button', { name: 'Publish and activate', exact: true })).toBeDisabled()
  await expect(page.getByRole('alert').filter({ hasText: 'The review is outdated or no longer executable.' })).toBeVisible()
  const activeBefore = (await api.call('GET', `${suite}/release-registry`)).activeReleaseId
  expect(activeBefore).toBe(original.release.id)
  const renewedReview = page.waitForResponse(response => response.url().endsWith('/publication-reviews') && response.request().method() === 'POST')
  await page.getByRole('button', { name: 'Validate and preview', exact: true }).click()
  reviewed = await (await renewedReview).json()
  await expect(preview.getByRole('article', { name: 'SELF-CHECK publication change', exact: true })).toContainText('Concurrent health check')
  await page.route('**/catalog-drafts/*/publish', async route => {
    const response = await route.fetch()
    expect(response.status()).toBe(200)
    publicationPath = new URL(route.request().url()).pathname
    await route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ message: 'Simulated lost publication response after commit' }) })
  }, { times: 1 })
  const pendingRequest = page.waitForRequest(request => request.url().endsWith('/publish') && request.method() === 'POST')
  publicationPath = `${suite}/catalog-drafts/${reviewed.draftId}/publish`
  await expectHTTPFailure(page, 'POST', publicationPath, 503, () => preview.getByRole('button', { name: 'Publish and activate', exact: true }).click())
  const firstRequest = await pendingRequest
  const key = firstRequest.headers()['idempotency-key']
  const committed = await api.call('GET', `${suite}/release-registry`)
  expect(committed.items).toHaveLength(2)
  await expect(page.getByRole('button', { name: 'Validate and preview', exact: true })).toBeDisabled()
  await expect(preview.getByRole('button', { name: 'Publish release', exact: true })).toBeDisabled()
  const retryRequest = page.waitForRequest(request => request.url().endsWith('/publish') && request.method() === 'POST')
  await preview.getByRole('button', { name: 'Publish and activate', exact: true }).click()
  expect((await retryRequest).headers()['idempotency-key']).toBe(key)
  await expect(page.getByRole('status').filter({ hasText: /^Release published and activated/ })).toBeVisible()
  const registry = await api.call('GET', `${suite}/release-registry`)
  expect(registry.items).toHaveLength(2)
  expect(registry.activeReleaseId).toBe(committed.activeReleaseId)
  const retained = (await api.call('GET', `${path}/runs/${original.id}`)).data
  expect(retained.release.id).toBe(original.release.id)
  expect(retained.tests[0].title).toBe('TestStudio health')
  await page.getByRole('button', { name: 'Validate and preview', exact: true }).click()
  await expect(preview).toContainText('No test definition changes from the active release.')
  await page.setViewportSize({ width: 390, height: 844 })
  const width = await page.evaluate(() => ({ document: document.documentElement.scrollWidth, viewport: innerWidth }))
  expect(width.document, 'The complete review stays within the mobile viewport').toBeLessThanOrEqual(width.viewport)
  await testInfo.attach('publication-review-mobile', { body: await page.screenshot({ fullPage: true }), contentType: 'image/png' })
  await preview.getByRole('button', { name: 'Publish release', exact: true }).click()
  await expect(page.getByRole('status').filter({ hasText: /^Release published\. The active release stays unchanged/ })).toBeVisible()
  const unactivated = await api.call('GET', `${suite}/release-registry`)
  expect(unactivated.items).toHaveLength(3)
  expect(unactivated.activeReleaseId).toBe(registry.activeReleaseId)
})
