import net from 'node:net'
import fs from 'node:fs'
import path from 'node:path'

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
  const filename = '/tmp/teststudio-environment.config.ts'
  const directory = path.dirname(context.config)
  const capture = input.capture
  const use = { baseURL: input.browser.origin, proxy: { server: proxy }, serviceWorkers: 'block', screenshot: capture.screenshots === 'off' ? 'off' : 'on', trace: capture.trace === 'off' ? 'off' : 'on', video: capture.video === 'off' ? 'off' : 'on' }
  fs.writeFileSync(filename, `import suite from ${JSON.stringify(context.config)}\nimport path from 'node:path'\nconst use = ${JSON.stringify(use)}\nconst directory = ${JSON.stringify(directory)}\nif (suite.webServer) throw new Error('The environment browser profile uses a registered target and cannot start a suite web server')\nexport default { ...suite, testDir: path.resolve(directory, suite.testDir || '.'), outputDir: '/tmp/teststudio-browser-results', use: { ...suite.use, ...use }, projects: (suite.projects || [{}]).map(project => ({ ...project, testDir: path.resolve(directory, project.testDir || suite.testDir || '.'), outputDir: '/tmp/teststudio-browser-results', use: { ...suite.use, ...project.use, ...use } })) }\n`)
  return { config: filename, env: { TESTSTUDIO_BASE_URL: input.browser.origin, HTTP_PROXY: proxy, HTTPS_PROXY: proxy, http_proxy: proxy, https_proxy: proxy, NO_PROXY: '', no_proxy: '' }, close: () => { for (const socket of sockets) socket.destroy(); server.close() } }
}
