import { createHash, generateKeyPairSync, randomBytes, sign } from 'node:crypto'

export function createOIDCFixture(secret) {
  const requests = new Map(), codes = new Map(), logouts = new Map()
  const generations = { one: 1, two: 1 }, failures = { one: false, two: false }
  const keys = Object.fromEntries(['one', 'two'].map(id => [id, generateKeyPairSync('rsa', { modulusLength: 2048 })]))
  const json = (response, status, value) => { response.writeHead(status, { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' }); response.end(JSON.stringify(value)) }
  return async (request, response) => {
    const url = new URL(request.url, 'http://127.0.0.1:8050')
    if (request.method==='POST' && url.pathname==='/oidc/control') {
      let raw='';for await(const chunk of request){raw+=chunk.toString();if(raw.length>4096){json(response,413,{});return true}}
      const input=JSON.parse(raw), id=input.provider
      if(!['one','two'].includes(id)){json(response,400,{});return true}
      if(input.rotate){keys[id]=generateKeyPairSync('rsa',{modulusLength:2048});generations[id]++}
      if(typeof input.discoveryFailure==='boolean')failures[id]=input.discoveryFailure
      json(response,200,{generation:generations[id]});return true
    }
    if (url.pathname === '/favicon.ico') { response.writeHead(204).end(); return true }
    const match = /^\/oidc\/(one|two)\/(.+)$/.exec(url.pathname)
    if (!match) return false
    const [, id, operation] = match, issuer = `http://127.0.0.1:8050/oidc/${id}`
    if (request.method === 'GET' && operation === '.well-known/openid-configuration') {
      if(failures[id]){json(response,503,{error:'fixture_discovery_unavailable'});return true}
      json(response, 200, { issuer, end_session_endpoint:`${issuer}/end-session`, authorization_endpoint: `${issuer}/authorize`, token_endpoint: `${issuer}/token`, jwks_uri: `${issuer}/keys`, id_token_signing_alg_values_supported: ['RS256'] }); return true
    }
    if (request.method === 'GET' && operation === 'keys') { json(response, 200, { keys: [{ ...keys[id].publicKey.export({ format: 'jwk' }), kid: `fixture-${id}-${generations[id]}`, alg: 'RS256', use: 'sig' }] }); return true }
    if (request.method === 'GET' && operation === 'authorize') {
      const redirect = new URL(url.searchParams.get('redirect_uri'))
      if (redirect.origin !== 'http://127.0.0.1:8040' || !/^\/auth\/callback\/[A-Za-z0-9._-]+$/.test(redirect.pathname) || !['teststudio-public', 'teststudio-private'].includes(url.searchParams.get('client_id')) || url.searchParams.get('response_type') !== 'code' || url.searchParams.get('code_challenge_method') !== 'S256' || !url.searchParams.get('nonce') || !url.searchParams.get('state')) { json(response, 400, { error: 'invalid_fixture_request' }); return true }
      const key = randomBytes(24).toString('hex')
      requests.set(key, { id, redirect: redirect.href, state: url.searchParams.get('state'), nonce: url.searchParams.get('nonce'), challenge: url.searchParams.get('code_challenge'), client: url.searchParams.get('client_id') })
      response.writeHead(200, { 'Content-Type': 'text/html', 'Cache-Control': 'no-store' })
      response.end(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Local provider ${id}</title><body><main><h1>Local provider ${id}</h1><form method="post" action="${issuer}/approve"><input type="hidden" name="transaction" value="${key}"><label>Fixture subject<input name="subject" required value="shared-subject"></label><label>Fixture groups<input name="groups" value="reviewers"></label><label>Fixture email<input name="email" type="email" required value="same@example.test"></label><label><input name="verified" type="checkbox" value="yes" checked>Fixture verified email</label><label for="fixture-group-mode">Fixture group response</label><select id="fixture-group-mode" name="groupMode"><option value="complete">Complete</option><option value="missing">Missing</option><option value="overflow">Overflow</option><option value="distributed">Distributed</option></select><button type="submit">Confirm fixture sign-in</button><button type="submit" name="cancel" value="yes">Cancel fixture sign-in</button></form></main></body></html>`)
      return true
    }
    if(request.method==='GET' && operation==='end-session') {
      const redirect=new URL(url.searchParams.get('post_logout_redirect_uri'))
      if(redirect.origin!=='http://127.0.0.1:8040' || !/^\/auth\/logout\/callback\/[A-Za-z0-9._-]+$/.test(redirect.pathname) || !['teststudio-public','teststudio-private'].includes(url.searchParams.get('client_id')) || !url.searchParams.get('state') || ['id_token_hint','access_token','refresh_token','client_secret'].some(key=>url.searchParams.has(key))){json(response,400,{error:'invalid_fixture_logout'});return true}
      const key=randomBytes(24).toString('hex');logouts.set(key,{id,redirect:redirect.href,state:url.searchParams.get('state')})
      response.writeHead(200,{'Content-Type':'text/html','Cache-Control':'no-store'})
      response.end(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Provider sign-out</title><body><main><h1>Local provider ${id} sign-out</h1><form method="post" action="${issuer}/confirm-logout"><input type="hidden" name="transaction" value="${key}"><button type="submit">Confirm provider sign-out</button><button type="submit" name="cancel" value="yes">Cancel provider sign-out</button></form></main></body></html>`);return true
    }
    let raw = ''
    for await (const chunk of request) { raw += chunk.toString(); if (raw.length > 16384) { json(response, 413, { error: 'too_large' }); return true } }
    const body = new URLSearchParams(raw)
    if (request.method === 'POST' && operation === 'approve') {
      const key = body.get('transaction'), transaction = requests.get(key)
      if (!transaction || transaction.id !== id || !body.get('subject')) { json(response, 400, { error: 'invalid_fixture_approval' }); return true }
      requests.delete(key)
      if(body.get('cancel')==='yes'){const redirect=new URL(transaction.redirect);redirect.searchParams.set('state',transaction.state);redirect.searchParams.set('error','access_denied');redirect.searchParams.set('error_description','private fixture diagnostic');response.writeHead(302,{Location:redirect.href,'Cache-Control':'no-store'}).end();return true}
      const code = randomBytes(24).toString('hex')
      codes.set(code, { ...transaction, subject: body.get('subject'), email: body.get('email'), emailVerified: body.get('verified') === 'yes', groupMode:body.get('groupMode'), groups: body.get('groups').split(',').map(value => value.trim()).filter(Boolean) })
      const redirect = new URL(transaction.redirect); redirect.searchParams.set('code', code); redirect.searchParams.set('state', transaction.state)
      response.setHeader('Set-Cookie',`sso_fixture_${id}=${randomBytes(24).toString('hex')}; Path=/oidc/${id}; HttpOnly; SameSite=Lax`)
      response.writeHead(302, { Location: redirect.href, 'Cache-Control': 'no-store' }).end(); return true
    }
    if (request.method === 'POST' && operation === 'token') {
      const code = body.get('code'), transaction = codes.get(code)
      const [headerClient, headerSecret] = String(request.headers.authorization ?? '').startsWith('Basic ') ? Buffer.from(request.headers.authorization.slice(6), 'base64').toString().split(':') : []
      const client = headerClient || body.get('client_id'), credential = headerSecret || body.get('client_secret') || ''
      if (!transaction || transaction.id !== id || client !== transaction.client || (client === 'teststudio-private' && credential !== secret) || body.get('redirect_uri') !== transaction.redirect || createHash('sha256').update(body.get('code_verifier') || '').digest('base64url') !== transaction.challenge) { json(response, 400, { error: 'invalid_grant' }); return true }
      codes.delete(code)
      const now = Math.floor(Date.now() / 1000)
      const claims = { iss: issuer, sub: transaction.subject, aud: client, iat: now, auth_time: now, exp: now + 3600, nonce: transaction.nonce, name: 'Local provider user', email: transaction.email, email_verified: transaction.emailVerified, groups: transaction.groups }
      if(transaction.groupMode==='missing')delete claims.groups
      if(transaction.groupMode==='overflow')claims.hasgroups=true
      if(transaction.groupMode==='distributed'){claims._claim_names={groups:'unavailable'};delete claims.groups}
      const unsigned = `${Buffer.from(JSON.stringify({ alg: 'RS256', kid: `fixture-${id}-${generations[id]}` })).toString('base64url')}.${Buffer.from(JSON.stringify(claims)).toString('base64url')}`
      json(response, 200, { access_token: randomBytes(24).toString('hex'), token_type: 'Bearer', id_token: `${unsigned}.${sign('RSA-SHA256', Buffer.from(unsigned), keys[id].privateKey).toString('base64url')}` }); return true
    }
    if(request.method==='POST' && operation==='confirm-logout') {
      const key=body.get('transaction'),transaction=logouts.get(key)
      if(!transaction || transaction.id!==id){json(response,400,{error:'invalid_logout_confirmation'});return true}
      logouts.delete(key)
      const redirect=new URL(transaction.redirect);redirect.searchParams.set('state',transaction.state)
      if(body.get('cancel')==='yes')redirect.searchParams.set('error','access_denied')
      else response.setHeader('Set-Cookie',`sso_fixture_${id}=; Path=/oidc/${id}; HttpOnly; SameSite=Lax; Max-Age=0`)
      response.writeHead(302,{Location:redirect.href,'Cache-Control':'no-store'}).end();return true
    }
    json(response, 404, { error: 'fixture_route_not_found' }); return true
  }
}
