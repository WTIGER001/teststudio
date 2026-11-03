export class DemoAPIError extends Error { constructor(status, message) { super(message); this.status = status } }

export async function demoAPI(origin) {
  let cookie = '', csrf = ''
  async function call(method, resource, value, revision, key) {
    const binary = Buffer.isBuffer(value)
    const response = await fetch(origin + resource, { method, redirect: 'error', signal: AbortSignal.timeout(20000), headers: {
      Accept: 'application/json', ...(cookie ? { Cookie: cookie } : {}), ...(method === 'GET' ? {} : { Origin: origin, 'X-CSRF-Token': csrf }),
      ...(value === undefined ? {} : { 'Content-Type': binary ? 'application/zip' : 'application/json' }), ...(revision === undefined ? {} : { 'If-Match': `"${revision}"` }), ...(key ? { 'Idempotency-Key': key } : {})
    }, body: value === undefined ? undefined : binary ? value : JSON.stringify(value) })
    const session = response.headers.getSetCookie().find(value => value.startsWith('teststudio_session='))
    if (session) cookie = session.split(';')[0]
    const result = await response.json()
    if (!response.ok) throw new DemoAPIError(response.status, `${method} ${resource}: ${result.message ?? response.status}`)
    return result
  }
  await call('POST', '/auth/local', {})
  csrf = (await call('GET', '/auth/session')).session.csrfToken
  return { call, async logout() { try { await call('POST', '/auth/logout', {}) } finally { cookie = ''; csrf = '' } } }
}

export async function waitForRun(api, base, id, milliseconds = 90000) {
  const deadline = Date.now() + milliseconds
  while (Date.now() < deadline) {
    const run = (await api.call('GET', `${base}/runs/${id}`)).data
    if (['completed', 'cancelled', 'timed-out', 'blocked', 'incomplete'].includes(run.state)) return run
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  await api.call('POST', `${base}/runs/${id}/cancel`, {})
  throw new Error('Demo execution exceeded its setup deadline; cancellation was requested')
}
