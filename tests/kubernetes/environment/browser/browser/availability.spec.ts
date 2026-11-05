import { expect, test } from '@playwright/test'

test('inspect application availability @NET-BROWSER-001', async ({ page }) => {
  const origin = process.env['TESTSTUDIO_BASE_URL']
  if (!origin) throw new Error('A registered application origin is required')
  await page.goto(origin)
  await expect(page.getByRole('heading', { name: 'Gateway workspace' })).toBeVisible()
  await page.getByRole('button', { name: 'Load availability' }).click()
  await expect(page.locator('output')).toHaveText('ready: 3 items')
  console.log('The registered application returned three available items')
})
