import fs from 'node:fs'

const [, output, address] = process.argv.slice(2)
fs.writeFileSync(output, `import { test, expect } from '@playwright/test';\n\ntest('test', async ({ page }) => {\n  await page.goto(${JSON.stringify(address)});\n  await page.getByRole('button', { name: 'Continue' }).click();\n});\n`)
process.stdin.resume()
process.stdin.on('end', () => process.exit(0))
process.on('SIGTERM', () => process.exit(0))
process.on('SIGINT', () => process.exit(0))
