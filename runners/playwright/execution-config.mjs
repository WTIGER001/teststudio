import fs from 'node:fs'
import path from 'node:path'

export function executionConfig(context, capture, overrides = {}, options = {}) {
  const filename = options.filename ?? '/tmp/teststudio-execution.config.ts'
  const restricted = options.restricted ?? false
  const use = {
    screenshot: restricted || capture.screenshots === 'off' ? 'off' : 'on',
    trace: restricted || capture.trace === 'off' ? 'off' : 'on',
    video: restricted || capture.video === 'off' ? 'off' : 'on',
    ...overrides
  }
  const directory = path.dirname(context.config)
  fs.writeFileSync(filename, `import suite from ${JSON.stringify(context.config)}
import path from 'node:path'
import { createRequire } from 'node:module'
const use = ${JSON.stringify(use)}
const restricted = ${restricted}
const selectedUse = options => { const value = { ...options, ...use }; if (restricted) delete value.proxy; return value }
const directory = ${JSON.stringify(directory)}
const sourceConfig = ${JSON.stringify(context.config)}
const resolveHook = value => Array.isArray(value) ? value.map(resolveHook) : typeof value === 'string' ? value.startsWith('.') || path.isAbsolute(value) ? path.resolve(directory, value) : createRequire(sourceConfig).resolve(value) : value
const rooted = value => typeof value === 'string' ? path.resolve(directory, value) : value
const server = value => ({ ...value, cwd: rooted(value.cwd ?? '.') })
if (${Boolean(options.environment)} && suite.webServer) throw new Error('The environment browser profile uses a registered target and cannot start a suite web server')
export default { ...suite, globalSetup: resolveHook(suite.globalSetup), globalTeardown: resolveHook(suite.globalTeardown), tsconfig: rooted(suite.tsconfig), snapshotDir: rooted(suite.snapshotDir), webServer: Array.isArray(suite.webServer) ? suite.webServer.map(server) : suite.webServer ? server(suite.webServer) : undefined, testDir: path.resolve(directory, suite.testDir || '.'), outputDir: '/output/playwright-results', use: selectedUse(suite.use), projects: (suite.projects || [{}]).map(project => ({ ...project, snapshotDir: rooted(project.snapshotDir ?? suite.snapshotDir), testDir: path.resolve(directory, project.testDir || suite.testDir || '.'), outputDir: '/output/playwright-results', use: selectedUse({ ...suite.use, ...project.use }) })) }
`)
  return filename
}
