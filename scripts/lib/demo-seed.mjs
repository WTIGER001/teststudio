import path from 'node:path'
import { sourceZIP } from './source-zip.mjs'
import { demoAPI, DemoAPIError, waitForRun } from './demo-api.mjs'

export const demoProject = 'teststudio-demo'
export const demoNativeIDs = ['DEMO-HEALTH-001', 'DEMO-AVAILABILITY-001', 'DEMO-TIMEOUT-001']
const description = 'A local demonstration with passing, failing, and timed-out checks.'

export async function seedDemo({ root, origin, target, owner, save, progress }) {
  const base = `/api/v1/projects/${demoProject}`, api = await demoAPI(origin)
  try {
    if (owner.seeded) { await api.call('GET', base); return owner }
    progress('Creating the explicit local demo project and catalog.')
    let project
    try { project = await api.call('GET', base) } catch (error) { if (!(error instanceof DemoAPIError && error.status === 404)) throw error }
    if (project && project.data.description !== description) throw new Error('The existing demo project is not owned by this setup; use another demo data directory')
    if (!project) await api.call('POST', '/api/v1/projects', { id: demoProject, name: 'TestStudio local demo', description })
    async function ensure(kind, id, value) {
      const existing = (await api.call('GET', `${base}/${kind}`)).items.find(item => item.id === id)
      if (existing) return existing
      return api.call('PUT', `${base}/${kind}/${id}`, { id, ...value }, 0)
    }
    await ensure('applications', 'parcel-desk', { name: 'Parcel desk', description: 'New generic local demonstration fixture' })
    await ensure('environments', 'local', { name: 'Local demo target', applicationId: 'parcel-desk', stage: 'local', protected: false, disabled: false, endpoints: { api: target } })
    const timing = { mode: 'warn', slowdownFactor: 2, minimumIncreaseMs: 100, minimumSamples: 3, window: 20, maxAgeDays: 30, timeoutMode: 'adaptive', timeoutFactor: 3, timeoutMarginMs: 250, minimumTimeoutMs: 1000 }
    const native = await ensure('suites', 'availability', { name: 'Availability examples', description: 'Three checks show a healthy response, an unavailable service, and a timeout.', applications: ['parcel-desk'], requiresTarget: true, archived: false, timing })
    await ensure('suites', 'browser', { name: 'Browser screenshot example', description: 'A new Playwright check captures the parcel summary as screenshot evidence.', applications: ['parcel-desk'], requiresTarget: false, archived: false })
    if (!native.data.activeReleaseId) {
      const tests = [
        { id: demoNativeIDs[0], title: 'Healthy service responds', path: '/health', description: 'A stable passing check for timing and history.', timeout: 3 },
        { id: demoNativeIDs[1], title: 'Unavailable service is detected', path: '/availability', description: 'Deliberately fails until the local demo target is recovered.', timeout: 3 },
        { id: demoNativeIDs[2], title: 'Slow service times out', path: '/slow', description: 'Deliberately exceeds its one-second timeout and retains a distinct timeout status.', timeout: 1 }
      ].map(test => ({ id: test.id, title: test.title, description: test.description, applicationId: 'parcel-desk', category: 'Local availability', type: 'https', runner: 'http', state: 'enabled', safety: 'safe', rationale: 'Reads only the newly created local demo fixture.', timeoutSeconds: test.timeout, tags: ['demo'], dependencies: [], requiredBindings: [], check: { endpoint: 'api', path: test.path, method: 'GET', statusCodes: [200] } }))
      const preview = await api.call('POST', `${base}/suites/availability/source-uploads`, { name: 'demo-http.json', format: 'json', content: JSON.stringify({ schemaVersion: 'teststudio.catalog/v1', tests }) })
      await api.call('POST', `${base}/suites/availability/source-uploads/${preview.upload.id}/publish`, { activate: true }, preview.upload.revision)
    }
    const browserSuite = (await api.call('GET', `${base}/suites`)).items.find(item => item.id === 'browser')
    if (!browserSuite.data.activeReleaseId) {
      progress('Preparing the browser example with the cached trusted runtime.')
      const bundle = sourceZIP(path.join(root, 'examples/demo'))
      const job = await api.call('POST', `${base}/suites/browser/source-bundles?name=demo-browser.zip&manifest=teststudio-suite.json`, bundle, undefined, `demo-browser-${owner.id}`)
      owner.browserJob = job.id; save(owner)
      let current
      const deadline = Date.now() + 150000
      while (Date.now() < deadline) {
        current = await api.call('GET', `${base}/suites/browser/source-preparations/${job.id}`)
        if (['ready', 'published', 'failed', 'cancelled'].includes(current.data.state)) break
        await new Promise(resolve => setTimeout(resolve, 500))
      }
      if (!current || !['ready', 'published'].includes(current.data.state)) throw new Error(`Browser example preparation did not complete: ${current?.data.error || current?.data.state || 'unavailable'}`)
      if (current.data.state === 'ready') {
        const preview = await api.call('GET', `${base}/suites/browser/source-uploads/${current.data.uploadId}`)
        await api.call('POST', `${base}/suites/browser/source-uploads/${current.data.uploadId}/publish`, { activate: true }, preview.upload.revision)
      }
      owner.browserReference = current.data.reference; owner.browserImage = current.data.image; save(owner)
    }
    progress('Recording a short demonstration history through normal test runs.')
    owner.runs ??= []
    const selections = [[demoNativeIDs[0]], demoNativeIDs.slice(0, 2), demoNativeIDs]
    for (let index = 0; index < selections.length; index++) {
      const run = await api.call('POST', `${base}/runs`, { suiteId: 'availability', environmentId: 'local', testIds: selections[index], executionLocation: 'local', timeoutSeconds: 30 }, undefined, `demo-native-${owner.id}-${index}`)
      const result = await waitForRun(api, base, run.id)
      if (result.state !== 'completed' || result.counts.total !== selections[index].length || result.counts.passed !== 1 || result.counts.failed !== (index > 0 ? 1 : 0) || result.counts.timedOut !== (index === 2 ? 1 : 0)) throw new Error('The demo native run did not produce its expected pass, failure, and timeout counts')
      owner.runs[index] = { id: run.id, outcome: result.outcome, counts: result.counts }; save(owner)
    }
    const browser = await api.call('POST', `${base}/runs`, { suiteId: 'browser', testIds: ['DEMO-BROWSER-001'], executionLocation: 'local', timeoutSeconds: 60 }, undefined, `demo-browser-run-${owner.id}`)
    const result = await waitForRun(api, base, browser.id)
    if (result.state !== 'completed' || result.outcome !== 'passed' || result.evidenceState !== 'complete') throw new Error('The browser demonstration did not finish with complete passing evidence')
    const artifacts = await api.call('GET', `${base}/runs/${browser.id}/artifacts`)
    if (!artifacts.items.some(item => item.data.kind === 'screenshot')) throw new Error('The browser demonstration has no screenshot evidence')
    owner.browserRun = browser.id; owner.seeded = true; save(owner)
    return owner
  } finally { await api.logout().catch(() => {}) }
}
