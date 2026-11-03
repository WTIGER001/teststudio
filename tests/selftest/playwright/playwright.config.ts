import { defineConfig } from '@playwright/test'

const input = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}')
const capture = input.capture ?? { screenshots: 'failure', trace: 'failure', video: 'off' }
export default defineConfig({
  testDir: '.', testMatch: '*.spec.ts', forbidOnly: true, fullyParallel: false, retries: 0, workers: 1,
  outputDir: '/tmp/teststudio-playwright-results',
  use: {
    baseURL: 'http://127.0.0.1:8040', browserName: 'chromium', headless: true,
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
