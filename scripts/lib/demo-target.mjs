import { createServer } from 'node:http'

const page = ready => `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>TestStudio demo target</title><style>body{font:17px system-ui;background:#f2f4f7;color:#20293c;padding:36px;margin:0}main{max-width:680px;margin:auto;background:#fff;padding:32px;border-radius:16px}h1{font-size:32px}.status{padding:14px;background:${ready ? '#dcf7eb' : '#fff1d8'};border-radius:8px}button{padding:12px 18px;border:0;border-radius:8px;background:#6554d9;color:white;font:inherit;cursor:pointer}code{font-size:14px}li{margin:16px 0}</style></head><body><main><small>LOCAL DEMONSTRATION FIXTURE</small><h1>TestStudio demo target</h1><p>This target provides controlled results for your demonstration.</p><p class="status">The example service is ${ready ? 'recovered and healthy' : 'deliberately unavailable'}.</p><ul><li><code>/health</code> always returns a healthy response.</li><li><code>/availability</code> returns ${ready ? '200' : '503'}.</li><li><code>/slow</code> deliberately exceeds the demo test timeout.</li></ul><form method="post" action="/${ready ? 'break' : 'recover'}"><button type="submit">${ready ? 'Make the example unavailable' : 'Recover the example service'}</button></form><p>After recovery, use Rerun failures in TestStudio. The availability check passes. The deliberate timeout remains visible.</p></main></body></html>`

export async function startDemoTarget(port) {
  let recovered = false
  const timers = new Set()
  const server = createServer((request, response) => {
    response.setHeader('Cache-Control', 'no-store')
    response.setHeader('X-Content-Type-Options', 'nosniff')
    if (request.method === 'GET' && request.url === '/') { response.setHeader('Content-Type', 'text/html; charset=utf-8'); response.end(page(recovered)); return }
    if (request.method === 'POST' && ['/recover', '/break'].includes(request.url)) {
      if (request.headers.origin !== `http://127.0.0.1:${server.address().port}`) { response.writeHead(403).end(); return }
      recovered = request.url === '/recover'; response.writeHead(303, { Location: '/' }).end(); return
    }
    if (request.method !== 'GET') { response.writeHead(405).end(); return }
    const status = request.url === '/availability' && !recovered ? 503 : ['/health', '/availability', '/slow'].includes(request.url) ? 200 : 404
    const send = () => { if (response.destroyed) return; response.writeHead(status, { 'Content-Type': 'application/json' }); response.end(JSON.stringify({ status: status === 200 ? 'ready' : 'unavailable', service: 'local-demo' })) }
    if (request.url === '/slow') { const timer = setTimeout(() => { timers.delete(timer); send() }, 2500); timers.add(timer); response.once('close', () => { clearTimeout(timer); timers.delete(timer) }); return }
    send()
  })
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(port, '127.0.0.1', resolve) })
  return { origin: `http://127.0.0.1:${server.address().port}`, async close() { for (const timer of timers) clearTimeout(timer); server.closeAllConnections(); await new Promise(resolve => server.close(resolve)) } }
}
