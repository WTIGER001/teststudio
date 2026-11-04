const expires = Date.parse(new URL(location.href).searchParams.get('expires') ?? '')
let closed = false
async function close() {
  if (closed) return
  closed = true
  try { await (await navigator.serviceWorker?.getRegistration())?.unregister() } catch {}
  location.replace('/trace-closed')
}
if (!Number.isFinite(expires) || expires <= Date.now()) void close()
else setTimeout(close, Math.min(expires - Date.now(), 2147483647))
async function check() {
  if (closed) return
  try {
    const response = await fetch('./authorization', { credentials: 'omit', cache: 'no-store', headers: { 'x-pw-serviceworker': 'skip' } })
    if ([401, 403, 404, 410].includes(response.status)) await close()
  } catch {}
}
void check()
setInterval(check, 5000)
