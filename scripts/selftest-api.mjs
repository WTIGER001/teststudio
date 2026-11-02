export class LocalTestClient {
  constructor(origin) {
    this.origin = new URL(origin).origin
    if (!['127.0.0.1', '[::1]', 'localhost'].includes(new URL(this.origin).hostname)) throw new Error('Local verification requires a loopback origin')
    this.cookie = ''; this.csrf = ''
  }
  async call(method, path, body, revision) {
    const response = await fetch(this.origin + path, { method, signal: AbortSignal.timeout(30000), headers: {
      Origin: this.origin, Accept: 'application/json', 'Content-Type': 'application/json', Cookie: this.cookie,
      'X-CSRF-Token': this.csrf, 'Idempotency-Key': crypto.randomUUID(), ...(revision == null ? {} : { 'If-Match': `"${revision}"` })
    }, body: body === undefined ? undefined : JSON.stringify(body) })
    const cookie = response.headers.getSetCookie().find(value => value.startsWith('teststudio_session='))
    if (cookie) this.cookie = cookie.split(';')[0]
    const value = await response.json()
    if (!response.ok) throw new Error(`${method} ${path}: ${response.status} ${value.message}`)
    return value
  }
  async signIn() { await this.call('POST', '/auth/local', {}); this.csrf = (await this.call('GET', '/auth/session')).session.csrfToken }
}
