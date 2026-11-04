import { execFile } from 'node:child_process'
import { promisify } from 'node:util'
import { test, expect, expectHTTPFailure } from './fixtures'

test.use({ serviceWorkers: 'allow' })

test('Inspect a private installation-hosted trace without external requests', { tag: '@STUDIO-TRACE-001' }, async ({ page, api }, testInfo) => {
  const { project } = await api.scaffold()
  const media = `/tmp/teststudio-diagnostics-${project}`
  const generated = await promisify(execFile)(process.execPath, ['/suite/diagnostic-media.mjs', media, 'website'], { timeout: 15000, maxBuffer: 4096 })
  expect(JSON.parse(generated.stdout).closed).toBe(true)
  const response = await page.request.post('http://127.0.0.1:8050/diagnostics/prepare', { data: { project } })
  expect(response.ok(), await response.text()).toBeTruthy()
  const { run } = await response.json()
  const external: string[] = []
  page.context().on('response', response => {
    const url = new URL(response.url())
    if (url.protocol.startsWith('http') && !['http://127.0.0.1:8040', 'http://127.0.0.1:8041', 'http://127.0.0.1:8050'].includes(url.origin) && !response.fromServiceWorker()) external.push(url.origin)
  })
  await page.goto(`/projects/${project}/runs/${run}`)
  await page.getByRole('button', { name: 'Inspect test BROWSER-1', exact: true }).click()
  await page.getByLabel('Attempt evidence scope', { exact: true }).selectOption({ label: 'Attempt 1' })
  await expect(page.getByRole('region', { name: 'Execution snapshot', exact: true })).toContainText('diagnostics')
  await expect(page.getByRole('region', { name: 'Attempt history', exact: true })).toContainText('Confirmation heading did not match')
  await expect(page.getByRole('region', { name: 'Latest run log segments', exact: true })).toContainText('Checkout attempt one diagnostic')
  const screenshot = page.getByRole('article', { name: 'checkout-one.png', exact: true })
  await screenshot.getByRole('button', { name: 'View', exact: true }).click()
  await expect(page.getByAltText('checkout-one.png', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Close preview', exact: true }).click()
  const attachment = page.getByRole('article', { name: 'run-context.json', exact: true })
  await attachment.getByRole('button', { name: 'View', exact: true }).click()
  await expect(page.getByRole('region', { name: 'run-context.json', exact: true })).toContainText('local trace inspection')
  await page.getByRole('button', { name: 'Close preview', exact: true }).click()
  const artifact = page.getByRole('article', { name: 'checkout-trace.zip', exact: true })
  await artifact.getByRole('button', { name: 'View', exact: true }).click()
  const iframe = page.locator('iframe[title="Private Playwright trace"]')
  await expect(iframe).toBeVisible()
  const viewer = page.frameLocator('iframe[title="Private Playwright trace"]')
  await expect(viewer.getByText('Actions', { exact: true }).first()).toBeVisible({ timeout: 15000 })
  await testInfo.attach('private-viewer-body', { body: Buffer.from(await viewer.locator('body').innerText()), contentType: 'text/plain' })
  await expect(viewer.getByText(/Navigate|goto/).first()).toBeVisible()
  await viewer.getByText(/Navigate|goto/).first().click()
  await expect.poll(async () => {
    for (const frame of page.frames()) {
      if (frame.url().startsWith('http://127.0.0.1:8041/trace/') && await frame.getByRole('heading', { name: 'Diagnostic fixture', exact: true }).count()) {
        const heading = frame.getByRole('heading', { name: 'Diagnostic fixture', exact: true })
        if (await heading.evaluate(element => getComputedStyle(element).color) !== 'rgb(25, 125, 75)') continue
        if (await frame.getByAltText('Fixture mark', { exact: true }).evaluate(element => (element as HTMLImageElement).naturalWidth) !== 80) continue
        return true
      }
    }
    return false
  }).toBe(true)
  expect(external).toEqual([])
  await testInfo.attach('private-trace-viewer', { body: await page.screenshot({ fullPage: true }), contentType: 'image/png' })
  const source = await iframe.getAttribute('src')
  expect(source).toMatch(/^http:\/\/127\.0\.0\.1:8041\/trace\//)
  const token = new URL(source!).pathname.split('/')[2]
  const authorization = `http://127.0.0.1:8041/trace/${token}/authorization`
  await expectHTTPFailure(page, 'GET', authorization, 401, async () => {
    const expired = await page.request.post('http://127.0.0.1:8050/trace/expire', { data: { project, token } })
    expect(expired.ok()).toBeTruthy()
    expect(await page.evaluate(async url => (await fetch(url, { credentials: 'omit', cache: 'no-store' })).status, authorization)).toBe(401)
  }, { allowRepeated: true })
  await expect(viewer.getByRole('heading', { name: 'Trace access ended', exact: true })).toBeVisible({ timeout: 10000 })
  await page.getByRole('button', { name: 'Refresh trace access', exact: true }).click()
  await expect(viewer.getByText('Actions', { exact: true }).first()).toBeVisible({ timeout: 15000 })
  await page.getByRole('button', { name: 'Download trace', exact: true }).click()
  const download = page.getByRole('link', { name: 'Download checkout-trace.zip', exact: true })
  await expect(download).toHaveAttribute('href', /^http:\/\/127\.0\.0\.1:8041\/access\//)
  await page.getByRole('button', { name: 'Close preview', exact: true }).click()
  await expect(artifact.getByRole('button', { name: 'View', exact: true })).toBeFocused()
  expect(external).toEqual([])
})
