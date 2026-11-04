const fs = require('node:fs')
const path = require('node:path')
const { stripVTControlCharacters } = require('node:util')
const clean = (message, limit = 16384) => stripVTControlCharacters(message || '').slice(0, limit)

class TestStudioReporter {
  constructor() {
    this.input = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM)
    this.results = []
    this.steps = []
    this.problems = []
    this.attachmentCount = 0
  }
  onBegin(config, suite) {
    const tests = suite.allTests()
    if (tests.length !== 1 || !tests[0].tags.includes(this.input.test.selector)) throw new Error('The selector must resolve to exactly one tagged test')
    if (config.projects.some(project => project.retries !== 0)) throw new Error('TestStudio owns retries')
  }
  onTestBegin(test, result) { console.log("TESTSTUDIO_EVENT " + JSON.stringify({ event: "test.started", testId: this.input.test.id, selector: this.input.test.selector, at: result.startTime.toISOString() })); console.log(`Starting ${this.input.test.id}`) }
  onStepEnd(test, result, step) {
    if (this.steps.length < 500) this.steps.push({ title: step.title.slice(0, 2048), category: step.category, durationMs: Math.max(0, step.duration), error: step.error ? clean(step.error.message) : undefined })
  }
  onStdOut(chunk) { process.stdout.write(chunk) }
  onStdErr(chunk) { process.stderr.write(chunk) }
  onError(error) { this.problems.push(clean(error.message)) }
  onTestEnd(test, result) {
    const state = result.status === 'timedOut' ? 'timed-out' : result.status === 'skipped' ? 'skipped' : result.status === test.expectedStatus ? 'passed' : 'failed'
    const failureKind = state === 'timed-out' ? 'timeout' : state === 'failed' ? result.status === 'interrupted' ? 'cancelled' : 'assertion' : undefined
    this.results.push({
      result: { testId: this.input.test.id, state, failureKind, error: clean(result.errors.map(error => error.message || error.value || '').join('\n')), startedAt: result.startTime.toISOString(), finishedAt: new Date(result.startTime.getTime() + result.duration).toISOString(), durationMs: Math.max(0, result.duration), runnerStatus: result.status, expectedStatus: test.expectedStatus, steps: this.steps },
      attachments: result.attachments
    })
    console.log(`${this.input.test.id}: ${state} (${result.duration} ms)`)
  }
  async onEnd(summary) {
    if (this.results.length !== 1 || this.problems.length) { console.error(this.problems.join('\n') || 'The selected test produced no unique result'); return }
    const { result, attachments } = this.results[0]
    if (summary.status !== 'passed' && result.state !== 'failed' && result.state !== 'timed-out') { console.error('The worker failed outside the selected test'); return }
    const failed = result.state === 'failed' || result.state === 'timed-out' || this.input.failedBefore || (this.input.timingMode === 'fail' && this.input.slowThresholdMs != null && result.durationMs >= this.input.slowThresholdMs)
    const files = [], evidenceErrors = []
    for (const attachment of attachments) {
      if (files.length >= 30) { evidenceErrors.push('Attachment count exceeds 30'); break }
      const kind = attachment.name === 'trace' ? 'trace' : attachment.contentType?.startsWith('image/') ? 'screenshot' : attachment.contentType?.startsWith('video/') ? 'video' : 'attachment'
      if (this.input.test.type === 'api' && (kind !== 'attachment' || !/^(?:text\/plain|application\/(?:json|[A-Za-z0-9.+-]+\+json))(?:;|$)/i.test(attachment.contentType || ''))) { evidenceErrors.push('API attachment omitted: only bounded UTF-8 text and JSON support diagnostic redaction.'); continue }
      const mode = kind === 'attachment' ? 'always' : this.input.capture[kind === 'screenshot' ? 'screenshots' : kind]
      if (mode === 'off' || (mode === 'failure' && !failed)) continue
      try {
        const extension = kind === 'trace' ? '.zip' : kind === 'screenshot' ? '.png' : kind === 'video' ? '.webm' : '.json'
        const filename = `attachment-${++this.attachmentCount}${extension}`
        if (attachment.path) {
          const info = fs.lstatSync(attachment.path)
          if (!info.isFile() || info.size > this.input.capture.maxFileBytes) throw new Error('Attachment is not a permitted regular file')
          fs.copyFileSync(attachment.path, path.join('/output', filename))
        } else if (attachment.body && attachment.body.length <= this.input.capture.maxFileBytes) fs.writeFileSync(path.join('/output', filename), attachment.body)
        else throw new Error('Attachment data is missing or too large')
        files.push({ path: filename, kind, filename, contentType: attachment.contentType || 'application/octet-stream' })
      } catch (error) { evidenceErrors.push(`${attachment.name}: ${error.message}`) }
    }
    for (const [setting, kind] of [['screenshots', 'screenshot'], ['trace', 'trace'], ['video', 'video']]) {
      const mode = this.input.capture[setting]
      if ((mode === 'always' || mode === 'failure' && failed) && !files.some(file => file.kind === kind)) evidenceErrors.push(`Requested ${kind} was not produced. Check the Playwright capture configuration or report attachments.`)
    }
    this.writeReport({ schemaVersion: 'teststudio.playwright/v1', testId: this.input.test.id, selector: this.input.test.selector, result, attachments: files, evidenceErrors: evidenceErrors.slice(0, 20) })
  }
  writeReport(report) {
    fs.writeFileSync('/output/report.tmp', JSON.stringify(report))
    fs.renameSync('/output/report.tmp', '/output/report.json')
  }
}
module.exports = TestStudioReporter
