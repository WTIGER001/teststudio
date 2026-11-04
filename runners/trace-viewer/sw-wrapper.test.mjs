import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import { test } from 'node:test'

test('cached trace responses recheck grants and cannot fetch external resources', async () => {
  let authorize = 204, listener
  const network = [], navigations = []
  const self = {
    location: new URL('https://artifacts.test/trace/grant/sw.bundle.js'),
    registration: { scope: 'https://artifacts.test/trace/grant/', unregister: async () => true },
    clients: { matchAll: async () => [{ url: 'https://artifacts.test/trace/grant/index.html', navigate: async url => navigations.push(url) }] },
    fetch: async input => { network.push(String(input)); return new Response(null, { status: authorize }) },
    addEventListener: (type, callback) => { if (type === 'fetch') listener = callback }
  }
  const context = vm.createContext({ self, URL, Headers, Response, Set, Promise, importScripts: () => self.addEventListener('fetch', event => event.respondWith(Promise.resolve(new Response('PRIVATE_CAPTURED', { headers: { 'Content-Type': 'image/svg+xml' } })))) })
  vm.runInContext(fs.readFileSync(new URL('./sw-wrapper.js', import.meta.url), 'utf8').replace('__TESTSTUDIO_ASSETS__', '[]'), context)
  async function request() { let response; listener({ request: { url: 'https://artifacts.test/trace/grant/contexts' }, respondWith: value => { response = value } }); return await response }
  let response = await request()
  assert.equal(await response.text(), 'PRIVATE_CAPTURED')
  assert.match(response.headers.get('Content-Security-Policy'), /script-src 'none'/)
  authorize = 401
  response = await request()
  assert.equal(response.status, 401)
  assert.notEqual(await response.text(), 'PRIVATE_CAPTURED')
  assert.deepEqual(navigations, ['https://artifacts.test/trace-closed'])
  assert.equal((await self.fetch('https://external.test/trace.zip')).status, 403)
  assert.ok(network.every(url => url.startsWith('https://artifacts.test/trace/grant/authorization')))
})
