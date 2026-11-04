import { test, expect } from '@playwright/test'

test.use({ screenshot: 'off', trace: 'off', video: 'off' })

test('A passing check produces no browser media', { tag: '@ADAPTER-NOMEDIA' }, async () => {
  expect(2 + 2).toBe(4)
})
