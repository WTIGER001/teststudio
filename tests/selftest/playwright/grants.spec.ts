import { test, expect, expectHTTPFailure } from './fixtures'

test('Direct grants, exact expiry, deny precedence, and guarded editors', { tag: '@STUDIO-GRANTS-001' }, async ({ page, api }, testInfo) => {
  const { project, path } = await api.scaffold(), base = `${path}/permissions`, route = `/projects/${project}/permissions`
  const prepared = await page.request.post('http://127.0.0.1:8050/access/prepare', { data: { project } }); expect(prepared.ok()).toBe(true)
  const fixture = await prepared.json()
  const definition = { principalKind: 'user', principalId: fixture.readerId, permission: 'teststudio.catalog.read', effect: 'allow', object: 'suite/health', reason: 'Review exact catalog access' }
  const table = () => page.getByRole('table', { name: 'Managed direct grants', exact: true })
  const row = (kind: string, principal: string, effect: string) => table().getByRole('row').filter({ hasText: `${kind}/${principal}` }).filter({ has: page.getByText(effect, { exact: true }) })
  const editor = () => page.getByRole('region', { name: /^(Create direct grant|Edit grant expiry)$/ })
  const allowRow = () => row('user', fixture.readerId, 'Allow'), denyRow = () => row('group', 'direct-reviewers', 'Deny')
  const health = `${path}/suites/health/catalog-drafts`, reader = { headers: { Cookie: `teststudio_session=${fixture.readerCookie}` } }
  const current = async (effect = 'allow') => (await api.call('GET', `${base}?limit=1000`)).items.find((item: any) => item.data.principalId === (effect === 'allow' ? fixture.readerId : 'direct-reviewers') && item.data.effect === effect)
  const create = async (kind: string, principal: string, effect: string) => {
    await page.getByRole('button', { name: '+ Add direct grant', exact: true }).click()
    await page.getByLabel('Principal type', { exact: true }).selectOption(kind)
    await page.getByLabel('Registered principal identifier', { exact: true }).fill(principal)
    await page.getByLabel('Exact object (optional)', { exact: true }).fill('suite/health')
    await page.getByRole('button', { name: 'Refresh grant permissions', exact: true }).click()
    await page.getByLabel('Permission', { exact: true }).selectOption('teststudio.catalog.read')
    await page.getByLabel('Effect', { exact: true }).selectOption(effect)
  }
  const save = async (reason: string, button = 'Create grant') => {
    await page.getByLabel('Grant change reason', { exact: true }).fill(reason)
    await page.getByRole('button', { name: button, exact: true }).click()
    await expect(editor()).toHaveCount(0)
  }

  await test.step('Create exact user access and preserve unsaved values through help', async () => {
    await page.goto(route)
    await create('user', fixture.readerId, 'allow')
    await page.getByLabel('Grant change reason', { exact: true }).fill('Keep this grant reason')
    await page.getByRole('button', { name: 'Help', exact: true }).click()
    const help = page.getByRole('dialog', { name: 'Direct permissions help', exact: true })
    for (const name of ['Introduction', 'Concepts', 'Tutorials']) await expect(help.getByRole('heading', { name, exact: true })).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(page.getByLabel('Grant change reason', { exact: true })).toHaveValue('Keep this grant reason')
    for (const theme of ['light', 'dark']) {
      await page.getByLabel('Color theme').selectOption(theme)
      await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await testInfo.attach(`direct-grant-editor-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' })
    }
    await page.setViewportSize({ width: 1280, height: 900 })
    await save('Create exact user inspection')
    await expect(allowRow()).toContainText('suite/health')
    expect((await page.request.get(health, reader)).status()).toBe(200)
    expect((await page.request.get(`${path}/suites/other/catalog-drafts`, reader)).status()).toBe(403)
  })

  await test.step('Retain the opened revision through refresh and conflict', async () => {
    await allowRow().getByRole('button', { name: 'Edit expiry', exact: true }).click()
    await page.getByLabel('Expiry in UTC (optional)', { exact: true }).fill('2030-01-01T00:00:00.123456Z')
    await page.getByLabel('Grant change reason', { exact: true }).fill('Keep this expiry review')
    const before = await current()
    await api.call('PUT', `${base}/${before.id}`, { expiresAt: '2000-01-01T00:00:00.123456Z', reason: 'Concurrent expiry change' }, before.revision)
    await page.getByRole('button', { name: 'Refresh permissions', exact: true }).click()
    await expect(allowRow()).toContainText('Expired · no access')
    await expect(page.getByLabel('Expiry in UTC (optional)', { exact: true })).toHaveValue('2030-01-01T00:00:00.123456Z')
    await expectHTTPFailure(page, 'PUT', `${base}/${before.id}`, 409, () => page.getByRole('button', { name: 'Save grant expiry', exact: true }).click())
    await expect(page.getByLabel('Grant change reason', { exact: true })).toHaveValue('Keep this expiry review')
    await expect(page.getByRole('button', { name: 'Save grant expiry', exact: true })).toBeDisabled()
    await page.getByRole('button', { name: 'Reload stored grant', exact: true }).click()
    await expect(page.getByLabel('Expiry in UTC (optional)', { exact: true })).toHaveValue('2000-01-01T00:00:00.123456Z')
    await page.getByLabel('Expiry in UTC (optional)', { exact: true }).fill('')
    await save('Restore non-expiring exact access', 'Save grant expiry')
    expect((await page.request.get(health, reader)).status()).toBe(200)
  })

  await test.step('Keep group deny precedence and expired rules visible', async () => {
    await api.call('POST', '/api/v1/groups', { id: 'direct-reviewers', name: 'Direct reviewers', projectId: project })
    await api.call('PUT', '/api/v1/groups/direct-reviewers/members/' + fixture.readerId, {}, 0)
    await api.call('POST', `${path}/role-assignments`, { principalKind: 'group', principalId: 'direct-reviewers', roleId: 'builtin.teststudio.viewer', reason: 'Verify role and direct grant interaction' })
    await create('group', 'direct-reviewers', 'deny')
    await save('Deny exact inherited inspection')
    expect((await page.request.get(health, reader)).status()).toBe(403)
    await denyRow().getByRole('button', { name: 'Edit expiry', exact: true }).click()
    await page.getByLabel('Expiry in UTC (optional)', { exact: true }).fill('2000-01-01T00:00:00.123456Z')
    await save('Retain an expired deny for review', 'Save grant expiry')
    await expect(denyRow()).toContainText('Expired · no access')
    expect((await page.request.get(health, reader)).status()).toBe(200)
    await denyRow().getByRole('button', { name: 'Edit expiry', exact: true }).click()
    await page.getByLabel('Expiry in UTC (optional)', { exact: true }).fill('')
    await save('Restore the exact deny', 'Save grant expiry')
    expect((await page.request.get(health, reader)).status()).toBe(403)
    await denyRow().getByRole('button', { name: 'Remove grant', exact: true }).click()
    await page.getByLabel('Reason', { exact: true }).fill('Restore access by removing this deny')
    await page.getByRole('button', { name: 'Confirm action', exact: true }).click()
    await expect(denyRow()).toHaveCount(0)
    expect((await page.request.get(health, reader)).status()).toBe(200)
    const retained = await api.call('POST', base, { ...definition, principalKind: 'group', principalId: 'direct-reviewers', effect: 'deny', expiresAt: '2000-01-01T00:00:00.123456Z' })
    await page.getByRole('button', { name: 'Refresh permissions', exact: true }).click()
    await expect(denyRow()).toContainText('Expired · no access')
    await denyRow().getByRole('button', { name: 'Remove grant', exact: true }).click()
    await page.getByLabel('Reason', { exact: true }).fill('Keep this unconfirmed removal reason')
    const endpoint = `${base}/${retained.id}`; let submissions = 0
    await page.route(`**${endpoint}`, async intercepted => { if (intercepted.request().method() !== 'DELETE') return intercepted.continue(); submissions++; const response = await intercepted.fetch(); expect(response.status()).toBe(200); await intercepted.fulfill({ status: 200, contentType: 'application/json', body: 'unreadable removal response' }) })
    try {
      await page.getByRole('button', { name: 'Confirm action', exact: true }).click()
      await expect(page.getByRole('alert')).toContainText('The server returned an unreadable response')
      await expect(page.getByRole('button', { name: 'Confirm action', exact: true })).toBeDisabled()
      await expect(page.getByLabel('Reason', { exact: true })).toHaveValue('Keep this unconfirmed removal reason')
      expect((await page.request.get(endpoint)).status()).toBe(404)
      expect(submissions).toBe(1)
    } finally { await page.unroute(`**${endpoint}`) }
    await page.getByRole('button', { name: 'Cancel action', exact: true }).click()
    await api.call('POST', base, { ...definition, principalKind: 'group', principalId: 'direct-reviewers', effect: 'deny', expiresAt: '2000-01-01T00:00:00.123456Z' })
    await page.getByRole('button', { name: 'Refresh permissions', exact: true }).click()
  })

  await test.step('Reject stale removal after recreation and retain its reason', async () => {
    const before = await current()
    await allowRow().getByRole('button', { name: 'Remove grant', exact: true }).click()
    await page.getByLabel('Reason', { exact: true }).fill('Keep this removal reason')
    await api.call('DELETE', `${base}/${before.id}`, { reason: 'Concurrent removal' }, before.revision)
    const after = await api.call('POST', base, definition)
    expect(after.id).toBe(before.id); expect(after.revision).toBeGreaterThan(before.revision)
    await expectHTTPFailure(page, 'DELETE', `${base}/${before.id}`, 409, () => page.getByRole('button', { name: 'Confirm action', exact: true }).click())
    await expect(page.getByLabel('Reason', { exact: true })).toHaveValue('Keep this removal reason')
    await expect(page.getByRole('button', { name: 'Confirm action', exact: true })).toBeDisabled()
    await page.getByRole('button', { name: 'Cancel action', exact: true }).click()
    await page.getByRole('button', { name: 'Refresh permissions', exact: true }).click()
    await expect(allowRow().getByRole('button', { name: 'Remove grant', exact: true })).toBeVisible()
  })

  await test.step('Require explicit state review after an unreadable successful save', async () => {
    const before = await current(), endpoint = `${base}/${before.id}`
    let submissions = 0
    await allowRow().getByRole('button', { name: 'Edit expiry', exact: true }).click()
    await page.getByLabel('Expiry in UTC (optional)', { exact: true }).fill('2031-01-01T00:00:00Z')
    await page.getByLabel('Grant change reason', { exact: true }).fill('Preserve this unconfirmed reason')
    await page.route(`**${endpoint}`, async intercepted => { if (intercepted.request().method() !== 'PUT') return intercepted.continue(); submissions++; const response = await intercepted.fetch(); expect(response.status()).toBe(200); await intercepted.fulfill({ status: 200, contentType: 'application/json', body: 'unreadable response' }) })
    try {
      await page.getByRole('button', { name: 'Save grant expiry', exact: true }).click()
      await expect(page.getByRole('alert')).toContainText('The server returned an unreadable response')
      await expect(page.getByRole('button', { name: 'Save grant expiry', exact: true })).toBeDisabled()
      await expect(page.getByLabel('Grant change reason', { exact: true })).toHaveValue('Preserve this unconfirmed reason')
      expect(submissions).toBe(1)
      expect((await current()).revision).toBe(before.revision + 1)
    } finally { await page.unroute(`**${endpoint}`) }
    await page.getByRole('button', { name: 'Reload stored grant', exact: true }).click()
    await expect(page.getByLabel('Expiry in UTC (optional)', { exact: true })).toHaveValue('2031-01-01T00:00:00Z')
    await page.getByRole('button', { name: 'Close grant editor', exact: true }).click()
  })

  await test.step('Use actual read-only permissions and clear revoked private edits', async () => {
    const original = await page.context().cookies('http://127.0.0.1:8040'), root = original.find(cookie => cookie.name === 'teststudio_session')!
    const rootCSRF = (await (await page.request.get('/auth/session')).json()).session.csrfToken
    try {
      await page.context().addCookies([{ name: 'teststudio_session', value: fixture.readerCookie, domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax', secure: false }])
      await page.goto(route)
      await expect(allowRow()).toContainText('Read only')
      await expect(page.getByRole('button', { name: '+ Add direct grant', exact: true })).toHaveCount(0)
      await expect(table().getByRole('button', { name: 'Edit expiry', exact: true })).toHaveCount(0)
      await page.goto('/settings/permissions')
      await expect(page.getByRole('alert')).toContainText('Membership and permission read authority is required')
      await page.context().addCookies([{ name: 'teststudio_session', value: fixture.cookie, domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax', secure: false }])
      await page.goto(route)
      await allowRow().getByRole('button', { name: 'Edit expiry', exact: true }).click()
      await page.getByLabel('Grant change reason', { exact: true }).fill('Private unsaved expiry review')
      const response = await page.request.get(`${path}/role-assignments/user/${fixture.userId}/builtin.teststudio.project-admin`, { headers: { Cookie: `${root.name}=${root.value}` } }); expect(response.ok()).toBe(true)
      const assignment = await response.json()
      const revoked = await page.request.delete(`${path}/role-assignments/user/${fixture.userId}/builtin.teststudio.project-admin`, { headers: { Cookie: `${root.name}=${root.value}`, Origin: 'http://127.0.0.1:8040', 'X-CSRF-Token': rootCSRF, 'If-Match': `"${assignment.revision}"` }, data: { reason: 'Revoke permission management during editing' } }); expect(revoked.ok()).toBe(true)
      const before = await page.request.get(base, { headers: { Cookie: `${root.name}=${root.value}` } }), stored = (await before.json()).items.find((item: any) => item.data.effect === 'allow')
      const endpoint = `${base}/${stored.id}`
      await page.route(`**${endpoint}`, async intercepted => { if (intercepted.request().method() !== 'PUT') return intercepted.continue(); const response = await intercepted.fetch(); expect(response.status()).toBe(403); await intercepted.fulfill({ status: 403, contentType: 'application/json', body: 'unreadable rejection' }) })
      try { await expectHTTPFailure(page, 'PUT', endpoint, 403, () => page.getByRole('button', { name: 'Save grant expiry', exact: true }).click()) }
      finally { await page.unroute(`**${endpoint}`) }
      await expect(editor()).toHaveCount(0)
      await expect(table().getByRole('row')).toHaveCount(1)
      await expect(page.getByRole('alert')).toContainText('Access changed')
    } finally { await page.context().addCookies(original); await page.goto(route) }
  })

  await test.step('Continue bounded pages and keep both themes readable on mobile', async () => {
    for (let index = 0; index < 31; index++) {
      const id = `grant-page-${index}`
      await api.call('POST', '/api/v1/users', { id, name: `Grant reviewer ${index}` })
      await api.call('POST', base, { ...definition, principalId: id })
    }
    await page.getByRole('button', { name: 'Refresh permissions', exact: true }).click()
    await expect(table().getByRole('row')).toHaveCount(31)
    await page.getByRole('button', { name: 'More grants', exact: true }).click()
    await expect(table().getByRole('row')).toHaveCount(34)
    for (const theme of ['light', 'dark']) {
      await page.getByLabel('Color theme').selectOption(theme)
      await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await testInfo.attach(`direct-grants-${theme}`, { body: await page.screenshot({ fullPage: theme === 'light', animations: 'disabled' }), contentType: 'image/png' })
    }
    await page.setViewportSize({ width: 1280, height: 900 })
  })

  await test.step('Protect the last administrator from a direct inspection deny', async () => {
    await page.goto('/settings/permissions')
    await page.getByRole('button', { name: '+ Add direct grant', exact: true }).click()
    await page.getByLabel('Registered principal identifier', { exact: true }).fill('local-administrator')
    await page.getByLabel('Permission', { exact: true }).selectOption('teststudio.permissions.read')
    await page.getByLabel('Effect', { exact: true }).selectOption('deny')
    await page.getByLabel('Grant change reason', { exact: true }).fill('Reject the final usable administrator change')
    await expectHTTPFailure(page, 'POST', '/api/v1/permissions/grants', 422, () => page.getByRole('button', { name: 'Create grant', exact: true }).click())
    await expect(page.getByRole('alert')).toContainText('last usable installation administrator')
    expect((await api.call('GET', '/api/v1/permissions/grants')).items).toHaveLength(0)
  })
})
