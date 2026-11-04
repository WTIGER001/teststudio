import { chromium } from 'playwright'
import fs from 'node:fs/promises'
import { createServer } from 'node:http'

const directory = process.argv[2]
if (!directory?.startsWith('/tmp/teststudio-diagnostics-') || directory.includes('..')) throw new Error('Use the isolated diagnostic fixture directory')
await fs.mkdir(directory, { recursive: true, mode: 0o700 })
const website = process.argv[3] === 'website'
const server = website ? createServer((request, response) => {
  if (request.url === '/style.css') { response.setHeader('Content-Type', 'text/css'); response.end('h1 { color: rgb(25, 125, 75); }'); return }
  if (request.url === '/mark.svg') { response.setHeader('Content-Type', 'image/svg+xml'); response.end('<svg xmlns="http://www.w3.org/2000/svg" width="80" height="40"><rect width="80" height="40" fill="green"/></svg>'); return }
  response.setHeader('Content-Type', 'text/html'); response.end('<!doctype html><link rel="stylesheet" href="/style.css"><main><h1>Diagnostic fixture</h1><p>Confirmation is unavailable</p><img src="/mark.svg" alt="Fixture mark"></main>')
}) : null
if (server) await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
const browser = await chromium.launch({ headless: true })
process.once('SIGTERM', () => { void browser.close() })
try {
  const context = await browser.newContext()
  await context.tracing.start({ screenshots: true, snapshots: true })
  const page = await context.newPage()
  if (server) { await page.goto(`http://127.0.0.1:${server.address().port}/`); await page.locator('img').evaluate(image => image.decode()) }
  else await page.setContent('<main><h1>Diagnostic fixture</h1><p>Confirmation is unavailable</p></main>')
  await page.screenshot({ path: `${directory}/screenshot.png` })
  await context.tracing.stop({ path: `${directory}/trace.zip` })
} finally { await browser.close(); if (server) { server.closeAllConnections(); await new Promise(resolve => server.close(resolve)) } }
console.log(JSON.stringify({ closed: true }))
