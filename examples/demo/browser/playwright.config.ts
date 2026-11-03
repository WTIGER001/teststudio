import { defineConfig } from '@playwright/test'

export default defineConfig({ testDir: './checks', outputDir: '/tmp/teststudio-demo-browser', projects: [{ name: 'chromium', use: { browserName: 'chromium' } }] })
