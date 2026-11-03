import { test, expect, expectHTTPFailure } from './fixtures'

test('bulk metadata previews preserve definitions and reject stale partial writes', { tag: '@STUDIO-BULK-METADATA-001' }, async ({ page, api }) => {
  const { project, path } = await api.scaffold()
  await api.createCheck(path)
  const run = await api.run(path)
  await api.call('POST', `${path}/suites/health/tests/SELF-CHECK/actions/duplicate`, { id: 'SECOND-CHECK' }, 1)
  const catalog = `${path}/suites/health`
  const initial = (await api.call('GET', `${catalog}/tests/SELF-CHECK`)).data
  const setup = async () => {
    await page.getByRole('button', { name: 'Select visible tests', exact: true }).click()
    await page.getByLabel('Bulk action', { exact: true }).selectOption('metadata')
  }
  const field = async (name: string, value: string) => {
    await page.getByRole('checkbox', { name: `Change ${name}`, exact: true }).check()
    await page.getByLabel(`Bulk ${name}`, { exact: true }).fill(value)
  }
  await page.goto(`/projects/${project}/catalog`)
  await setup()
  await expect(page.getByRole('button', { name: 'Preview bulk changes', exact: true })).toBeDisabled()
  await field('Category', 'Availability / Local')
  await field('Tags', 'reviewed, local')
  await field('Owner', 'Example team')
  await page.getByRole('button', { name: 'Preview bulk changes', exact: true }).click()
  const preview = page.getByRole('region', { name: 'Review metadata changes', exact: true })
  await expect(preview.getByRole('table', { name: 'Proposed catalog changes', exact: true })).toContainText('Availability → Availability / Local')
  expect((await api.call('GET', `${catalog}/tests/SELF-CHECK`)).data.category).toBe(initial.category)
  await preview.getByRole('button', { name: 'Apply reviewed changes', exact: true }).click()
  await expect(page.getByRole('status').filter({ hasText: 'Catalog changes applied' })).toBeVisible()
  for (const id of ['SELF-CHECK', 'SECOND-CHECK']) {
    const test = (await api.call('GET', `${catalog}/tests/${id}`)).data
    expect(test.category).toBe('Availability / Local'); expect(test.tags).toEqual(['reviewed', 'local']); expect(test.owner).toBe('Example team')
    expect(test.check).toEqual(initial.check)
  }
  await page.getByRole('checkbox', { name: 'Change Category', exact: true }).uncheck()
  await field('Owner', '')
  await field('Tags', '')
  const [response] = await Promise.all([
    page.waitForResponse(response => response.url().endsWith('/tests/bulk-preview') && response.request().method() === 'POST'),
    page.getByRole('button', { name: 'Preview bulk changes', exact: true }).click()
  ])
  const stalePlan = await response.json()
  await expect(preview).toBeVisible()
  const concurrent = await api.call('GET', `${catalog}/tests/SELF-CHECK`)
  await api.call('PUT', `${catalog}/tests/SELF-CHECK`, { ...concurrent.data, title: 'Concurrent title' }, concurrent.revision)
  await expectHTTPFailure(page, 'POST', `${catalog}/catalog-imports/${stalePlan.id}/apply`, 409, () => preview.getByRole('button', { name: 'Apply reviewed changes', exact: true }).click())
  expect((await api.call('GET', `${catalog}/tests/SECOND-CHECK`)).data.owner).toBe('Example team')
  await page.reload()
  await setup()
  await field('Owner', '')
  await field('Tags', '')
  await page.getByRole('button', { name: 'Preview bulk changes', exact: true }).click()
  await preview.getByRole('button', { name: 'Apply reviewed changes', exact: true }).click()
  await expect(page.getByRole('status').filter({ hasText: 'Catalog changes applied' })).toBeVisible()
  for (const id of ['SELF-CHECK', 'SECOND-CHECK']) {
    const test = (await api.call('GET', `${catalog}/tests/${id}`)).data
    expect(test.owner ?? '').toBe(''); expect(test.tags).toEqual([]); expect(test.category).toBe('Availability / Local')
  }
  const retained = (await api.call('GET', `${path}/runs/${run.id}`)).data
  expect(retained.tests[0]).toEqual(run.tests[0])
  const active = await api.call('GET', `${catalog}/release-registry`)
  expect(active.activeReleaseId).toBe(run.release.id)
})
