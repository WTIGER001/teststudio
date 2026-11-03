const fs = require('node:fs')
const path = require('node:path')

module.exports = class DiscoveryReporter {
  onBegin(_config, suite) { this.tests = suite.allTests() }
  onError() { this.failed = true }
  async onEnd(result) {
    if (this.failed || result.status !== 'passed') return
    const { suiteContext } = await import('./suite-context.mjs')
    const context = suiteContext('/suite', true)
    const tests = this.tests.map(test => {
      const file = path.relative(context.root, fs.realpathSync(test.location.file)).split(path.sep).join('/')
      if (file === '..' || file.startsWith('../') || path.isAbsolute(file)) throw new Error('Discovered test source escapes the suite context')
      let parent = test.parent
      while (parent && parent.type !== 'project') parent = parent.parent
      return { file, line: test.location.line, tags: test.tags, project: parent?.project()?.name ?? '' }
    })
    if (tests.length > 10000) throw new Error('Discovery exceeds 10000 framework test instances')
    const report = JSON.stringify({ schemaVersion: 'teststudio.discovery/v1', manifest: context.manifest, origin: context.origin, catalogText: context.catalogText, tests })
    if (Buffer.byteLength(report) > 8 << 20) throw new Error('Discovery report exceeds 8 MiB')
    fs.writeFileSync('/output/discovery.tmp', report, { mode: 0o600 })
    fs.renameSync('/output/discovery.tmp', '/output/discovery.json')
  }
}
