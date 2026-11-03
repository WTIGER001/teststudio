import { test, expect } from '@playwright/test'

function requireExecution() {
 if (!JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.id) throw new Error('Discovery must not execute hooks or test bodies')
}
test.beforeAll(requireExecution)
test('First independent discovery check', { tag: '@PREBUILT-BROWSER-1' }, async ({ page }) => {
 requireExecution()
 await page.setContent('<main><h1>Independent browser suite</h1></main>')
 await expect(page.getByRole('heading', { name: 'Independent browser suite' })).toBeVisible()
})
test.describe('Nested discovery context', { tag: '@PREBUILT-BROWSER-2' }, () => {
 test('Second independent discovery check', () => { requireExecution(); expect(13 + 19).toBe(32) })
})
