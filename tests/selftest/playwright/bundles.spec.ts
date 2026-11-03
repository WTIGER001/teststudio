import { test, expect } from './fixtures'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import { createHash } from 'node:crypto'

const sourceFile = path.join(__dirname, 'fixtures/bundle-go.zip')

test('ZIP staging retains source and queues cancellable preparation', { tag: '@STUDIO-BUNDLES-001' }, async ({ page, api }) => {
  const { project, path } = await api.scaffold()
  await api.call('PUT', `${path}/suites/bundles`, { id: 'bundles', name: 'Bundled Go checks', applications: [], requiresTarget: false }, 0)
  await page.goto(`/projects/${project}/sources`)
  await page.getByLabel('Suite for sources', { exact: true }).selectOption('bundles')
  await page.getByLabel('Go source ZIP', { exact: true }).setInputFiles(sourceFile)
  await page.getByLabel('Suite manifest path', { exact: true }).fill('suites/totals.json')
  const response = page.waitForResponse(response => response.request().method() === 'POST' && response.url().includes('/source-bundles?'))
  await page.getByRole('button', { name: 'Upload and prepare bundle', exact: true }).click()
  expect((await response).status()).toBe(200)
  await expect(page.getByRole('status').filter({ hasText: 'Waiting for a preparation worker' })).toBeVisible()
  const prepared = await api.call('GET', `${path}/suites/bundles/source-preparations`)
  expect(prepared.items).toHaveLength(1)
  const created = prepared.items[0]
  const original = await readFile(sourceFile)
  expect(created.data.state).toBe('received')
  expect(created.data.sourceDigest).toBe(`sha256:${createHash('sha256').update(original).digest('hex')}`)
  await expect(page.getByRole('status').filter({ hasText: 'Waiting for a preparation worker' })).toBeVisible()
  await page.getByRole('button', { name: 'Get original source ZIP', exact: true }).click()
  await expect(page.getByRole('link', { name: 'Save original source ZIP', exact: true })).toHaveAttribute('href', /^blob:/)
  const retained = await page.request.get(`${path}/suites/bundles/source-preparations/${created.id}/bundle`)
  expect(retained.ok()).toBeTruthy()
  expect(await retained.body()).toEqual(original)
  await page.getByRole('button', { name: 'Upload and prepare bundle', exact: true }).click()
  expect((await api.call('GET', `${path}/suites/bundles/source-preparations`)).items).toHaveLength(1)
  expect((await api.call('GET', `${path}/runs`)).items).toEqual([])
  await page.getByRole('button', { name: 'Cancel preparation', exact: true }).click()
  await expect(page.getByRole('table', { name: 'Source preparation history' }).getByRole('cell', { name: 'cancelled', exact: true })).toBeVisible()
  await page.getByLabel('Color theme').selectOption('dark')
  await page.setViewportSize({ width: 390, height: 844 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy()
  await page.reload()
  await page.getByLabel('Suite for sources', { exact: true }).selectOption('bundles')
  await expect(page.getByRole('cell', { name: 'cancelled', exact: true })).toBeVisible()
})
