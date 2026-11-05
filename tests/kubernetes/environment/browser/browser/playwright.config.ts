import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: '.',
  testMatch: '*.spec.ts',
  workers: 1,
  projects: [{ name: 'chromium', use: { browserName: 'chromium', headless: true } }]
})
