import { chromium } from 'playwright'
import fs from 'node:fs/promises'

const directory = process.argv[2]
if (!directory?.startsWith('/tmp/teststudio-diagnostics-') || directory.includes('..')) throw new Error('Use the isolated diagnostic fixture directory')
await fs.mkdir(directory, { recursive: true, mode: 0o700 })
const browser = await chromium.launch({ headless: true })
process.once('SIGTERM', () => { void browser.close() })
try {
  const context = await browser.newContext()
  await context.tracing.start({ screenshots: true, snapshots: true })
  const page = await context.newPage()
  await page.setContent('<main><h1>Diagnostic fixture</h1><p>Confirmation is unavailable</p></main>')
  await page.screenshot({ path: `${directory}/screenshot.png` })
  await context.tracing.stop({ path: `${directory}/trace.zip` })
} finally { await browser.close() }
console.log(JSON.stringify({ closed: true }))
