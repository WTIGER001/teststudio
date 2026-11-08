import net from 'node:net'

export async function recordingProxy(socketPath) {
  const connections = new Set()
  const server = net.createServer(client => {
    const upstream = net.connect(socketPath)
    connections.add(client); connections.add(upstream)
    for (const socket of [client, upstream]) {
      socket.once('close', () => connections.delete(socket))
      socket.once('error', () => { client.destroy(); upstream.destroy() })
    }
    client.pipe(upstream).pipe(client)
    client.once('close', () => upstream.destroy())
    upstream.once('close', () => client.destroy())
  })
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve) })
  return {
    settings: { server: `http://127.0.0.1:${server.address().port}`, bypass: '<-loopback>' },
    close: () => { for (const socket of connections) socket.destroy(); server.close() }
  }
}
