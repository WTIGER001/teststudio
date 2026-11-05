import { test, expect } from '@playwright/test'
import fs from 'node:fs'

const css = fs.readFileSync(new URL('../theme.css', import.meta.url), 'utf8')
function contrast(first: string, second: string): number {
  const luminance = (value: string) => {
    const channels = value.match(/[\d.]+/g)!.slice(0, 3).map(channel => Number(channel) / 255).map(channel => channel <= .04045 ? channel / 12.92 : ((channel + .055) / 1.055) ** 2.4)
    return channels[0] * .2126 + channels[1] * .7152 + channels[2] * .0722
  }
  const a = luminance(first), b = luminance(second)
  return (Math.max(a, b) + .05) / (Math.min(a, b) + .05)
}

test('Keep button text readable on the first frame of a theme change', { tag: '@STUDIO-COLOR-001' }, async ({ page }, testInfo) => {
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await page.setContent('<!doctype html><html data-theme="light"><head><title>TestStudio button theme verification</title><style>' + css + '</style></head><body><main style="padding:32px"><h1>TestStudio theme controls</h1><button class="button secondary">Secondary action</button><button class="button primary">Primary action</button><button class="button danger">Destructive action</button></main></body></html>')
  await testInfo.attach('buttons-light', { body: await page.screenshot(), contentType: 'image/png' })
  const observations: unknown[] = []
  for (const theme of ['dark', 'light']) {
    const colors = await page.evaluate(async theme => {
      document.documentElement.dataset.theme = theme
      await new Promise(requestAnimationFrame)
      return [...document.querySelectorAll<HTMLButtonElement>('.button')].map(button => {
        const style = getComputedStyle(button)
        return { text: button.textContent, foreground: style.color, background: style.backgroundColor, opacity: style.opacity }
      })
    }, theme)
    for (const color of colors) {
      expect(Number(color.opacity)).toBe(1)
      expect(contrast(color.foreground, color.background), `${theme} ${color.text} first-frame contrast`).toBeGreaterThanOrEqual(4.5)
    }
    observations.push({ theme, colors })
    await testInfo.attach('buttons-' + theme + '-first-frame', { body: await page.screenshot(), contentType: 'image/png' })
  }
  await page.keyboard.press('Tab')
  await expect(page.getByRole('button', { name: 'Secondary action', exact: true })).toBeFocused()
  await testInfo.attach('button-theme-observations', { body: Buffer.from(JSON.stringify(observations)), contentType: 'application/json' })
  expect(errors).toEqual([])
})
