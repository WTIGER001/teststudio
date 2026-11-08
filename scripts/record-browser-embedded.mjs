import { randomUUID } from 'node:crypto'
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

let width = 1280, height = 720
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
let target = null, picking = Promise.resolve(), check = null
const extensions = new Map()
controller.on('sourceChanged', source => {
  const actions = source.actions.map(action => {
    for (const [marker, extension] of extensions) if (action.includes(marker)) return `  ${extension.statement}`
    return action
  })
  const generated = [source.header, ...actions, source.footer].join('\n')
  if (Buffer.byteLength(generated) > 256 * 1024) { void stop(); return }
  code = generated
  writeFileSync(output, code, { mode: 0o600 })
})
await controller.setRecorderMode(server.nullProgress, { mode: 'recording' })
const [recorder] = await controller._allRecorders()
recorder.on('actionAdded', action => {
  const item = action.action
  const extension = extensions.get(item.ariaSnapshot)
  if (extension) { steps.push({ kind: extension.kind, description: extension.description }); return }
  const name = item.selector?.match(/\[name=("(?:[^"\\]|\\.)*")/)?.[1]
  let label = ''
  try { label = name ? JSON.parse(name) : '' } catch {}
  const target = label ? `“${label}”` : 'the selected item'
  const description = ({ navigate: 'Open the website', click: `Click ${target}`, fill: `Enter text in ${target}`, press: `Press ${item.key ?? 'a key'}`, check: `Select ${target}`, uncheck: `Clear ${target}`, select: `Choose an option in ${target}`, assertVisible: `Check ${target} is visible`, assertText: `Check the text in ${target}`, assertValue: `Check the value in ${target}` })[item.name]
  if (description) steps.push({ kind: item.name, description })
  if (steps.length > 500) void stop()
})
controller.on('inspectRequested', info => { picking = pick(info.selector) })
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
  return { image: image.toString('base64'), width, height, url: page.url(), mode, sequence, steps, ...(target ? { target: { label: target.label, matches: target.matches, text: target.text } } : {}), ...(check ? { message: check } : {}) }
}


async function addExtension(kind, description, statement) {
  if (steps.length >= 500) throw new Error('Recording step limit reached')
  const marker = `teststudio_${randomUUID().replaceAll('-', '')}`
  extensions.set(marker, { kind, description, statement })
  await controller.setRecorderMode(server.nullProgress, { mode: 'recording' })
  await recorder._recordAction(server.nullProgress, playwright._connection.toImpl(page).mainFrame(), { name: 'assertSnapshot', selector: 'body', ariaSnapshot: marker })
}

async function pick(selector) {
  try {
    const picked = page.locator(selector)
    const details = await picked.evaluate(element => {
      const item = element.closest('article,li,tr,[role="listitem"],[role="row"]') ?? element
      const segments = []
      let parent = item.parentElement
      while (parent && parent.tagName !== 'HTML') {
        if (parent.id) { segments.unshift('#' + CSS.escape(parent.id)); break }
        const tag = parent.tagName.toLowerCase()
        const siblings = parent.parentElement ? [...parent.parentElement.children].filter(child => child.tagName === parent.tagName) : []
        segments.unshift(tag + (siblings.length > 1 ? ':nth-of-type(' + (siblings.indexOf(parent) + 1) + ')' : ''))
        parent = parent.parentElement
      }
      const role = item.getAttribute('role')
      const tag = item.tagName.toLowerCase()
      segments.push(tag + (role ? '[role=' + JSON.stringify(role) + ']' : ''))
      return { selector: segments.join(' > '), label: ({ article: 'News articles', li: 'List items', tr: 'Table rows' })[tag] ?? 'Matching ' + tag + ' items', text: (element.textContent ?? '').trim().slice(0, 200) }
    }, { timeout: 5000 })
    const parts = selector.split(' >> internal:control=enter-frame >> ')
    const grouped = [...parts.slice(0, -1), details.selector].join(' >> internal:control=enter-frame >> ')
    target = { ...details, selector, grouped, matches: await page.locator(grouped).count() }
    check = null
  } catch { target = null; check = 'That item is no longer available. Choose it again.' }
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
        } else if (command.action === 'assert-pick') {
          target = null; check = null
          await controller.setRecorderMode(server.nullProgress, { mode: 'inspecting' })
        } else if (command.action === 'assert-add') {
          check = null
          if (!target || !['at-least', 'exactly', 'text'].includes(command.key)) throw new Error('Choose the items to check first')
          const locator = `page.locator(${JSON.stringify(command.key === 'text' ? target.selector : target.grouped)})`
          if (command.key === 'text') {
            if (typeof command.text !== 'string' || !command.text.length || command.text.length > 2048) throw new Error('Enter the expected text')
            await addExtension('assertText', `Check the selected item contains “${command.text}”`, `await expect(${locator}).toContainText(${JSON.stringify(command.text)});`)
          } else {
            if (!Number.isInteger(command.count) || command.count < 0 || command.count > 10000 || command.key === 'at-least' && command.count < 1) throw new Error('Enter a valid item count')
            const matches = await page.locator(target.grouped).count()
            if (command.key === 'at-least' ? matches < command.count : matches !== command.count) {
              check = `There are ${matches} matching items now. Change the expected count or choose different items before adding this check.`
            } else {
              await addExtension('assertCount', `Check ${target.label.toLowerCase()}: ${command.key === 'at-least' ? 'at least' : 'exactly'} ${command.count}`, command.key === 'at-least' ? `await expect.poll(() => ${locator}.count()).toBeGreaterThanOrEqual(${command.count});` : `await expect(${locator}).toHaveCount(${command.count});`)
            }
          }
          if (!check) { target = null; await controller.setRecorderMode(server.nullProgress, { mode: 'recording' }) }
        } else if (command.action === 'assert-url') {
          await addExtension('assertURL', 'Check the current page address', `await expect(page).toHaveURL(${JSON.stringify(page.url())});`)
        } else if (command.action === 'resize') {
          if (!Number.isInteger(command.width) || command.width < 320 || command.width > 1920 || !Number.isInteger(command.height) || command.height < 240 || command.height > 1200) throw new Error('Choose a supported browser size')
          await page.setViewportSize({ width: command.width, height: command.height })
          width = command.width; height = command.height
          target = null; check = null
          await controller.setRecorderMode(server.nullProgress, { mode: 'recording' })
          await addExtension('resize', `Set browser size to ${width} × ${height}`, `await page.setViewportSize({ width: ${width}, height: ${height} });`)
        } else if (command.action === 'record') {
          target = null; check = null
          await controller.setRecorderMode(server.nullProgress, { mode: 'recording' })
        } else throw new Error('Unsupported recording action')
      }
      await picking
      await recorder.flushPendingActions()
      process.stdout.write(JSON.stringify({ view: await view(), code }) + '\n')
    } catch {
      process.stdout.write(JSON.stringify({ error: 'The browser action could not finish. Refresh the browser view and check the recorded steps before continuing.', code }) + '\n')
    }
  })
})
input.on('close', () => { void chain.finally(() => stop()).finally(() => process.exit(0)) })
