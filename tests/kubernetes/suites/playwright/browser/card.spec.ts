import { test, expect } from '@playwright/test'

test('Inspect a local result card', { tag: '@K3S-BROWSER-001' }, async ({ page }, testInfo) => {
  await page.setContent('<main><h1>Worker result</h1><p role="status">Ready for review</p><button type="button" onclick="document.querySelector(\'p\').textContent=\'Reviewed\'">Review result</button></main>')
  await expect(page.getByRole('heading', { name: 'Worker result' })).toBeVisible()
  await page.getByRole('button', { name: 'Review result' }).click()
  await expect(page.getByRole('status')).toHaveText('Reviewed')
  await testInfo.attach('worker-result-card', { body: await page.screenshot(), contentType: 'image/png' })
})
