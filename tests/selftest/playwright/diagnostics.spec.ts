import { test, expect } from './fixtures'

test('Intentional failure verifies evidence ingestion', { tag: '@ADAPTER-FAIL' }, async ({ page, api }) => {
  await expect(page.getByRole('heading', { name: 'This heading intentionally does not exist' })).toBeVisible({ timeout: 250 })
})

test('Interrupted work never becomes a passing test', { tag: '@ADAPTER-TIMEOUT' }, async ({ page, api }) => {
  console.log('The interruption verification test is running.')
  await new Promise(() => {})
})
