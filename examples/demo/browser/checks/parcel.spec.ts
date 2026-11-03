import { test, expect } from '@playwright/test'

test('a parcel summary shows its delivery details', { tag: '@DEMO-BROWSER-001' }, async ({ page }, testInfo) => {
  await page.setContent(`<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Parcel desk</title><style>body{font:18px system-ui;background:#f2f4f7;color:#20293c;margin:0;padding:64px}main{max-width:720px;margin:auto;background:white;padding:40px;border-radius:18px}small{color:#576781}h1{font-size:40px;margin-bottom:12px}.status{display:inline-block;padding:10px 16px;border-radius:30px;color:#166449;background:#dcf7eb}dl{display:grid;grid-template-columns:140px 1fr;gap:20px}dd{margin:0}</style></head><body><main><small>TESTSTUDIO LOCAL EXAMPLE</small><h1>Parcel desk</h1><p class="status">Ready for collection</p><dl><dt>Reference</dt><dd>PARCEL-1042</dd><dt>Recipient</dt><dd>Alex Example</dd><dt>Destination</dt><dd>Collection point 7</dd></dl></main></body></html>`)
  await expect(page.getByRole('heading', { name: 'Parcel desk' })).toBeVisible()
  await expect(page.getByText('Ready for collection', { exact: true })).toBeVisible()
  await testInfo.attach('Parcel summary', { body: await page.screenshot({ fullPage: true }), contentType: 'image/png' })
})
