const register = self.addEventListener.bind(self)
const network = self.fetch.bind(self)
const viewerAssets = new Set(__TESTSTUDIO_ASSETS__)
const scope = new URL(self.registration.scope).pathname
const contentPolicy = "default-src 'none'; script-src 'self' 'unsafe-inline'; style-src http: https: 'unsafe-inline'; img-src http: https: data: blob:; media-src http: https: blob:; font-src http: https: data:; connect-src 'self' blob:; frame-src 'self' blob: data:; worker-src 'self' blob:; object-src 'none'; base-uri http: https: about:; form-action 'none'"
self.fetch = (input, options) => {
  const url = new URL(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url, self.location.href)
  if (url.origin !== self.location.origin || !['http:', 'https:', 'blob:'].includes(url.protocol)) return Promise.resolve(new Response(null, { status: 403 }))
  return network(input, options)
}
self.addEventListener = (type, listener, options) => {
  if (type !== 'fetch') return register(type, listener, options)
  register(type, event => {
    const respond = event.respondWith.bind(event)
    event.respondWith = response => {
      const pending = Promise.resolve(response)
      pending.catch(() => {})
      respond(network(new URL('authorization', self.registration.scope), { credentials: 'omit', cache: 'no-store', headers: { 'x-pw-serviceworker': 'skip' } }).then(async access => {
        if (!access.ok) {
          await self.registration.unregister()
          const clients = await self.clients.matchAll({ type: 'window' })
          await Promise.allSettled(clients.filter(client => new URL(client.url).pathname === scope + 'index.html').map(client => client.navigate(new URL('/trace-closed', self.location.origin).href)))
          return new Response('Trace access ended', { status: access.status, headers: { 'Cache-Control': 'no-store', 'Content-Type': 'text/plain' } })
        }
        return pending
      }).then(value => {
      if (!value || ['opaque', 'opaqueredirect', 'error'].includes(value.type)) return value
      const headers = new Headers(value.headers)
      headers.set('Cache-Control', 'no-store')
      headers.set('Referrer-Policy', 'no-referrer')
      headers.set('X-Content-Type-Options', 'nosniff')
      const url = new URL(event.request.url)
      const relative = url.origin === self.location.origin && url.pathname.startsWith(scope) ? url.pathname.slice(scope.length) : ''
      const existing = headers.get('Content-Security-Policy')
      const trusted = viewerAssets.has(relative) || relative.startsWith('snapshot/') && existing?.includes("script-src 'nonce-")
      headers.set('Content-Security-Policy', trusted && existing ? `${existing}, ${contentPolicy}` : "script-src 'none'; " + contentPolicy.replace("script-src 'self' 'unsafe-inline'; ", ''))
      return new Response(value.body, { status: value.status, statusText: value.statusText, headers })
      }))
    }
    listener.call(self, event)
  }, options)
}
importScripts('./sw.vendor.js')
