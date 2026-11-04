import { test, expect } from '@playwright/test'

test('Intentional runner assertion failure', { tag: '@ADAPTER-FAIL' }, async ({ page }) => {
  await page.setContent('<main><h1>Fresh runner diagnostic</h1><p>Observed state: unavailable</p></main>')
  await test.step('Compare an intentionally different state', async () => {
    await expect(page.getByRole('heading', { name: 'Expected available state', exact: true })).toBeVisible({ timeout: 250 })
  })
})

test('Intentional runner timeout', { tag: '@ADAPTER-TIMEOUT' }, async ({ page }) => {
  await page.setContent('<main><h1>Fresh timeout diagnostic</h1><p>This isolated fixture waits for its test deadline.</p></main>')
  await test.step('Wait for a deliberately unavailable action', async () => {
    await page.getByRole('button', { name: 'Unavailable action', exact: true }).click({ timeout: 0 })
  })
})
