import net from 'node:net'
import { executionConfig } from './execution-config.mjs'

export async function browserEgress(input, context) {
  const socketPath = '/output/.browser-egress.sock'
  const deadline = Date.now() + 10000
  for (;;) {
    try {
      await new Promise((resolve, reject) => {
        const socket = net.connect(socketPath)
        socket.once('error', reject)
        socket.once('connect', () => { socket.end(); resolve() })
      })
      break
    } catch (error) {
      if (Date.now() >= deadline) throw new Error('The selected browser gateway is unavailable; inspect the execution worker configuration')
      await new Promise(resolve => setTimeout(resolve, 100))
    }
  }
  const sockets = new Set()
  const server = net.createServer(client => {
    const upstream = net.connect(socketPath)
    sockets.add(client); sockets.add(upstream)
    for (const socket of [client, upstream]) {
      socket.once('close', () => sockets.delete(socket))
      socket.once('error', () => { client.destroy(); upstream.destroy() })
    }
    client.pipe(upstream).pipe(client)
    client.once('close', () => upstream.destroy())
    upstream.once('close', () => client.destroy())
  })
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve) })
  const proxy = `http://127.0.0.1:${server.address().port}`
  const restricted = Boolean(input.browser.requests)
  const baseURL = restricted ? proxy : input.browser.origin
  const filename = executionConfig(context, input.capture, { baseURL, ...(restricted ? {} : { proxy: { server: proxy } }), serviceWorkers: 'block' }, { restricted, environment: true })
  return { config: filename, env: { TESTSTUDIO_BASE_URL: baseURL, ...(restricted ? { TESTSTUDIO_API_BASE_URL: baseURL, ALL_PROXY: '', all_proxy: '' } : {}), HTTP_PROXY: restricted ? '' : proxy, HTTPS_PROXY: restricted ? '' : proxy, http_proxy: restricted ? '' : proxy, https_proxy: restricted ? '' : proxy, NO_PROXY: '', no_proxy: '' }, close: () => { for (const socket of sockets) socket.destroy(); server.close() } }
}
