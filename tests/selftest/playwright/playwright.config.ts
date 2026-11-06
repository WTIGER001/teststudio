import { defineConfig } from '@playwright/test'

const input = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}')
const capture = input.capture ?? { screenshots: 'failure', trace: 'failure', video: 'off' }
export default defineConfig({
  testDir: '.', testMatch: '*.spec.ts', forbidOnly: true, fullyParallel: false, retries: 0, workers: 1,
  outputDir: '/tmp/teststudio-playwright-results',
  use: {
    launchOptions: input.test?.selector === '@STUDIO-PROJECTS-001' ? { env: { ...process.env, DISPLAY: ':97' }, args: ['--auto-select-tab-capture-source-by-title=TestStudio cover capture target', '--enable-usermedia-screen-capturing', '--allow-http-screen-capture'] } : {},
    baseURL: 'http://127.0.0.1:8040', browserName: 'chromium', headless: input.test?.selector !== '@STUDIO-PROJECTS-001',
    screenshot: capture.screenshots === 'off' ? 'off' : 'on',
    trace: capture.trace === 'off' ? 'off' : 'on', video: capture.video === 'off' ? 'off' : 'on',
    serviceWorkers: 'block'
  },
  webServer: {
    command: 'node /suite/start-server.mjs',
    url: 'http://127.0.0.1:8040/healthz', reuseExistingServer: false, timeout: 30000,
    gracefulShutdown: { signal: 'SIGTERM', timeout: 5000 }, stdout: 'pipe', stderr: 'pipe'
  }
})
