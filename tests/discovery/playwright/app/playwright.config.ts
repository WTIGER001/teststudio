import { defineConfig } from '@playwright/test'
const item = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}')
export default defineConfig({
 testDir: '.', testMatch: 'fresh.spec.ts', outputDir: '/tmp/teststudio-playwright-results',
 projects: [{ name: 'chromium', use: { browserName: 'chromium' } }, { name: 'firefox', use: { browserName: 'firefox' } }],
 globalSetup: './setup.ts',
 webServer: item.test?.id ? undefined : { command: 'node -e "require(\'node:fs\').writeFileSync(\'/output/server-started\',\'started\')"', url: 'http://127.0.0.1:19090', timeout: 1000 }
})
