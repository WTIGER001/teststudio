import { createRequire } from 'node:module'
import { writeFileSync } from 'node:fs'
import { createInterface } from 'node:readline'
import { recordingProxy } from './recording-proxy.mjs'

const [cli, output, address] = process.argv.slice(2)
if (!cli || !output || !address) throw new Error('Embedded recording requires a trusted runtime, private output, and start address')
const require = createRequire(cli)
const playwright = require('playwright-core')
const { server } = require('playwright-core/lib/coreBundle')
if (require('playwright-core/package.json').version !== '1.64.1') throw new Error('The embedded recorder requires the pinned Playwright runtime')

const width = 1280, height = 720
const originalDeadline = process.env.TESTSTUDIO_RECORDING_DEADLINE ? Date.parse(process.env.TESTSTUDIO_RECORDING_DEADLINE) : Date.now() + 20 * 60 * 1000
if (!Number.isFinite(originalDeadline) || originalDeadline <= Date.now() || originalDeadline > Date.now() + 20 * 60 * 1000) throw new Error('Recording requires a current bounded deadline')
const proxy = process.env.TESTSTUDIO_RECORDING_GATEWAY_SOCKET ? await recordingProxy(process.env.TESTSTUDIO_RECORDING_GATEWAY_SOCKET) : undefined
const browser = await playwright.chromium.launch({ headless: true, chromiumSandbox: true })
const context = await browser.newContext({ viewport: { width, height }, serviceWorkers: 'block', acceptDownloads: false, ...(proxy ? { proxy: proxy.settings } : {}) })
context.setDefaultTimeout(10000)
context.setDefaultNavigationTimeout(15000)
const page = await context.newPage()
const recordingContext = playwright._connection.toImpl(context)
const extendInjectedScript = recordingContext.extendInjectedScript.bind(recordingContext)
recordingContext.extendInjectedScript = (source, options) => extendInjectedScript(source, options && Object.hasOwn(options, 'recorderMode') ? { ...options, hideToolbar: true } : options)
const controller = playwright._connection.toImpl(playwright).debugController
controller.initialize(server.nullProgress, 'playwright-test', 'javascript')
let code = '', steps = [], sequence = 0, lastInput = '', mode = 'recording', stopping = false, deadline
controller.on('sourceChanged', source => {
  if (Buffer.byteLength(source.text) > 256 * 1024) { void stop(); return }
  code = source.text
  writeFileSync(output, code, { mode: 0o600 })
})
await controller.setRecorderMode(server.nullProgress, { mode: 'recording' })
const [recorder] = await controller._allRecorders()
recorder.on('actionAdded', action => {
  const item = action.action
  const name = item.selector?.match(/\[name=("(?:[^"\\]|\\.)*")/)?.[1]
  let label = ''
  try { label = name ? JSON.parse(name) : '' } catch {}
  const target = label ? `“${label}”` : 'the selected item'
  const description = ({ navigate: 'Open the website', click: `Click ${target}`, fill: `Enter text in ${target}`, press: `Press ${item.key ?? 'a key'}`, check: `Select ${target}`, uncheck: `Clear ${target}`, select: `Choose an option in ${target}`, assertVisible: `Check ${target} is visible`, assertText: `Check the text in ${target}`, assertValue: `Check the value in ${target}` })[item.name]
  if (description) steps.push({ kind: item.name, description })
  if (steps.length > 500) void stop()
})
controller.on('setModeRequested', value => { mode = value.mode })
context.on('page', popup => { if (popup !== page) void popup.close() })
try {
  await page.goto(address, { waitUntil: 'domcontentloaded' })
} catch {
  await stop()
  process.exit(1)
}

async function view() {
  const image = await page.screenshot({ type: 'jpeg', quality: 65, timeout: 10000 })
  return { image: image.toString('base64'), width, height, url: page.url(), mode, sequence, steps }
}

async function stop() {
  if (stopping) return
  stopping = true
  clearTimeout(deadline)
  await context.close().catch(() => {})
  await browser.close().catch(() => {})
  proxy?.close()
}

deadline = setTimeout(() => { void stop().finally(() => process.exit(0)) }, Math.max(1, originalDeadline - Date.now()))
process.on('SIGTERM', () => { void stop().finally(() => process.exit(0)) })
process.on('SIGINT', () => { void stop().finally(() => process.exit(0)) })
const input = createInterface({ input: process.stdin, crlfDelay: Infinity })
let chain = Promise.resolve()
input.on('line', line => {
  chain = chain.then(async () => {
    try {
      const command = JSON.parse(line)
      if (command.action !== 'view') {
        const fingerprint = JSON.stringify(command)
        if (command.sequence === sequence && fingerprint === lastInput) {
          process.stdout.write(JSON.stringify({ view: await view(), code }) + '\n')
          return
        }
        if (command.sequence !== sequence + 1) throw new Error('Refresh the recording browser before continuing')
        sequence = command.sequence
        lastInput = fingerprint
        if (command.action === 'click') {
          if (!Number.isFinite(command.x) || !Number.isFinite(command.y) || command.x < 0 || command.x >= width || command.y < 0 || command.y >= height) throw new Error('Choose an item inside the recording browser')
          await page.mouse.move(command.x, command.y)
          await page.mouse.click(command.x, command.y)
        } else if (command.action === 'text') {
          if (typeof command.text !== 'string' || command.text.length > 2048) throw new Error('Enter up to 2048 characters at a time')
          await page.keyboard.insertText(command.text)
        } else if (command.action === 'key') {
          if (!['Enter', 'Tab', 'Shift+Tab', 'Backspace', 'Delete', 'ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Home', 'End', 'Escape', 'ControlOrMeta+A'].includes(command.key)) throw new Error('Unsupported recording key')
          await page.keyboard.press(command.key)
        } else if (command.action === 'scroll') {
          if (!Number.isFinite(command.y) || Math.abs(command.y) > 1000) throw new Error('Unsupported scroll distance')
          await page.mouse.wheel(0, command.y)
        } else if (command.action === 'assert-visible') {
          await controller.setRecorderMode(server.nullProgress, { mode: 'assertingVisibility' })
        } else if (command.action === 'record') {
          await controller.setRecorderMode(server.nullProgress, { mode: 'recording' })
        } else throw new Error('Unsupported recording action')
      }
      await recorder.flushPendingActions()
      process.stdout.write(JSON.stringify({ view: await view(), code }) + '\n')
    } catch {
      process.stdout.write(JSON.stringify({ error: 'The browser action could not finish. Refresh the browser view and check the recorded steps before continuing.', code }) + '\n')
    }
  })
})
input.on('close', () => { void chain.finally(() => stop()).finally(() => process.exit(0)) })
