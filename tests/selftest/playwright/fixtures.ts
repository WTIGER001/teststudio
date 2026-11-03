import { test as base, expect, type APIRequestContext } from '@playwright/test'
import crypto from 'node:crypto'

export class API {
  constructor(private readonly request: APIRequestContext, private readonly csrf: string) {}
  async call(method: string, path: string, body?: unknown, revision?: number) {
    const response = await this.request.fetch(path, { method, data: body, headers: {
      Origin: 'http://127.0.0.1:8040', 'X-CSRF-Token': this.csrf, 'Idempotency-Key': crypto.randomUUID(),
      ...(revision == null ? {} : { 'If-Match': `"${revision}"` })
    } })
    expect(response.ok(), `${method} ${path}: ${await response.text()}`).toBeTruthy()
    return response.json()
  }
  async scaffold(project = `pw-${crypto.randomUUID().slice(0, 8)}`) {
    const path = `/api/v1/projects/${project}`
    await this.call('POST', '/api/v1/projects', { id: project, name: 'Self-test workspace', description: 'Disposable browser test data' })
    await this.call('PUT', `${path}/applications/app`, { id: 'app', name: 'TestStudio' }, 0)
    await this.call('PUT', `${path}/environments/local`, { id: 'local', name: 'Self-test instance', applicationId: 'app', stage: 'local', endpoints: { api: 'http://127.0.0.1:8040' } }, 0)
    await this.call('PUT', `${path}/suites/health`, { id: 'health', name: 'Health checks', applications: ['app'], requiresTarget: true }, 0)
    return { project, path }
  }
  async createCheck(path: string) {
    await this.call('POST', `${path}/suites/health/tests`, {
      id: 'SELF-CHECK', title: 'TestStudio health', category: 'Availability', applicationId: 'app', type: 'https', runner: 'http', state: 'enabled', safety: 'safe', rationale: 'Reads this isolated TestStudio instance',
      description: '', tags: [], dependencies: [], requiredBindings: [], timeoutSeconds: 10,
      check: { endpoint: 'api', path: '/healthz', method: 'GET', statusCodes: [200] }
    })
    const draft = await this.call('POST', `${path}/suites/health/catalog-drafts`, {})
    await this.call('POST', `${path}/suites/health/catalog-drafts/${draft.id}/publish`, { activate: true }, draft.revision)
  }
  async run(path: string) {
    const created = await this.call('POST', `${path}/runs`, { suiteId: 'health', environmentId: 'local', timeoutSeconds: 60 })
    let run: any
    await expect.poll(async () => { run = (await this.call('GET', `${path}/runs/${created.id}`)).data; return run.state }, { timeout: 20000 }).toBe('completed')
    expect(run.outcome).toBe('passed')
    return run
  }
}

export const test = base.extend<{ api: API }>({
  page: async ({ page }, use, testInfo) => {
    const diagnostics: { kind: string; message: string }[] = []
    const add = (kind: string, message: string) => { if (diagnostics.length < 200) diagnostics.push({ kind, message: message.slice(0, 2048) }) }
    page.on('pageerror', error => add('pageerror', error.message))
    page.on('console', message => { if (message.type() === 'error' || message.type() === 'warning') add(message.type(), message.text()) })
    page.on('requestfailed', request => add('requestfailed', `${request.method()} ${request.url()} ${request.failure()?.errorText}`))
    await use(page)
    await testInfo.attach('browser-diagnostics', { body: Buffer.from(JSON.stringify(diagnostics)), contentType: 'application/json' })
    expect(diagnostics.filter(item => item.kind === 'pageerror' || item.kind === 'error'), 'Browser runtime errors').toEqual([])
  },
  api: async ({ page }, use) => {
    await page.goto('/projects')
    await page.getByRole('button', { name: 'Open local workspace' }).click()
    await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
    const session = await (await page.request.get('/auth/session')).json()
    await use(new API(page.request, session.session.csrfToken))
  }
})
export { expect }
