import { test, expect, expectHTTPFailure, followNavigation } from './fixtures'

test('Effective decisions, inherited source paths, bounded scans, and revoked inspection', { tag: '@STUDIO-PERMISSIONS-001' }, async ({ page, api }, testInfo) => {
  const { project, path } = await api.scaffold(), route = `/projects/${project}/explain-access`
  const prepared = await page.request.post('http://127.0.0.1:8050/access/prepare', { data: { project } }); expect(prepared.ok()).toBe(true)
  const fixture = await prepared.json(), permission = 'teststudio.catalog.read', group = 'explanation-reviewers'
  const rule = (effect = 'allow', expiresAt?: string) => ({ permission, effect, object: 'suite/health', ...(expiresAt ? { expiresAt } : {}) })
  await api.call('POST', '/api/v1/groups', { id: group, name: 'Explanation reviewers', projectId: project })
  await api.call('PUT', `/api/v1/groups/${group}/members/${fixture.readerId}`, {}, 0)
  const policy = async (id: string, rules: any[], inherits: string[] = []) => api.call('POST', '/api/v1/roles', { id, projectId: project, name: id, rules, inherits, reason: 'Verify independent permission source inspection' })
  await policy('health-source', [rule()])
  await policy('composite-source', [], ['health-source'])
  await api.call('POST', `${path}/role-assignments`, { principalKind: 'group', principalId: group, roleId: 'composite-source', reason: 'Verify inherited group paths' })
  const grant = { principalKind: 'user', principalId: fixture.readerId, permission, effect: 'allow', object: 'suite/health', reason: 'Verify a direct source beside inherited access' }
  await api.call('POST', `${path}/permissions`, grant)
  const deny = await api.call('POST', `${path}/permissions`, { ...grant, principalKind: 'group', principalId: group, effect: 'deny' })
  await api.call('POST', `${path}/permissions`, { ...grant, effect: 'deny', expiresAt: '2000-01-01T00:00:00.123456Z' })
  const decision = () => page.getByRole('region', { name: 'Current decision', exact: true }), sources = () => page.getByRole('region', { name: 'Permission sources', exact: true })
  const check = async () => { await page.getByRole('button', { name: 'Check access', exact: true }).click(); await expect(decision()).toBeVisible() }
  const inspect = async () => {
    await page.getByLabel('Inspect user identifier', { exact: true }).fill(fixture.readerId)
    await page.getByLabel('Exact object (optional)', { exact: true }).fill('suite/health')
    await page.getByLabel('Permission to check', { exact: true }).selectOption(permission)
    await check()
  }
  const original = await page.context().cookies('http://127.0.0.1:8040'), root = original.find(cookie => cookie.name === 'teststudio_session')!
  const csrf = (await (await page.request.get('/auth/session')).json()).session.csrfToken
  const rootHeaders = { Cookie: `${root.name}=${root.value}`, Origin: 'http://127.0.0.1:8040', 'X-CSRF-Token': csrf }

  await test.step('Explain direct and inherited sources with deny precedence and expiry', async () => {
    await page.goto(route)
    await inspect()
    await expect(decision()).toContainText('Denied')
    await expect(decision()).toContainText('An active deny overrides')
    await expect(sources().getByRole('article')).toHaveCount(4)
    const inherited = sources().getByRole('article', { name: 'Inherited role grant', exact: true })
    for (const text of ['Explanation reviewers', 'composite-source', 'health-source', project]) await expect(inherited).toContainText(text)
    await expect(sources()).toContainText('Expired · no effect')
    await expect(sources()).toContainText('Source scan complete')
    await page.getByRole('button', { name: 'Help', exact: true }).click()
    const help = page.getByRole('dialog', { name: 'Explain access help', exact: true })
    for (const heading of ['Introduction', 'Concepts', 'Tutorials']) await expect(help.getByRole('heading', { name: heading, exact: true })).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(page.getByRole('button', { name: 'Help', exact: true })).toBeFocused()
    await expect(page.getByLabel('Inspect user identifier', { exact: true })).toHaveValue(fixture.readerId)
    for (const theme of ['light', 'dark']) {
      await page.getByLabel('Color theme').selectOption(theme)
      await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
      const overflow = await page.evaluate(() => Array.from(document.querySelectorAll('main *')).filter(element => element.getBoundingClientRect().right > innerWidth + 1).map(element => `${element.tagName}.${element.className}`))
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), overflow.join(', ')).toBe(true)
      await page.evaluate(() => scrollTo(0, 0))
      await testInfo.attach(`access-explanation-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' })
    }
    await page.setViewportSize({ width: 1280, height: 900 })
    await page.getByLabel('Exact object (optional)', { exact: true }).fill('')
    await expect(decision()).toHaveCount(0); await expect(sources()).toHaveCount(0)
    await page.getByLabel('Exact object (optional)', { exact: true }).fill('suite/health')
    await api.call('PUT', `${path}/permissions/${deny.id}`, { expiresAt: '2000-01-01T00:00:00Z', reason: 'Withdraw the active deny without deleting its history' }, deny.revision)
    await check(); await expect(decision()).toContainText('Allowed')
  })

  await test.step('Ignore a late private response after the inspection context changes', async () => {
    const pattern = `**/api/v1/users/${fixture.readerId}/effective-permissions?*`
    let release!: () => void, ready!: () => void
    const gate = new Promise<void>(resolve => { release = resolve }), captured = new Promise<void>(resolve => { ready = resolve })
    await page.route(pattern, async intercepted => { const response = await intercepted.fetch(); ready(); await gate; await intercepted.fulfill({ response }) })
    try {
      await page.getByRole('button', { name: 'Check access', exact: true }).click()
      await captured
      await page.getByLabel('Exact object (optional)', { exact: true }).fill('')
      await expect(decision()).toHaveCount(0)
      const response = page.waitForResponse(response => response.url().includes(`/users/${fixture.readerId}/effective-permissions?`))
      release(); await (await response).finished()
      await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
      await expect(decision()).toHaveCount(0); await expect(sources()).toHaveCount(0)
    } finally { release(); await page.unroute(pattern) }
  })

  await test.step('Keep self inspection available without revealing private group sources', async () => {
    try {
      await page.context().addCookies([{ name: 'teststudio_session', value: fixture.readerCookie, domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax', secure: false }])
      await page.goto(`${route}?object=suite%2Fhealth&permission=${permission}`)
      await expect(decision()).toContainText('Allowed')
      await expect(page.getByLabel('Inspect user identifier', { exact: true })).toHaveAttribute('readonly', '')
      const restricted = sources().getByRole('article', { name: 'Restricted source', exact: true })
      await expect(restricted).toHaveCount(2)
      for (const card of await restricted.all()) {
        await expect(card).not.toContainText(group)
        await expect(card).not.toContainText('health-source')
        await expect(card).not.toContainText('2000-01-01')
      }
      await followNavigation(page, 'Explain installation access')
      await expect(decision()).toContainText('Denied')
      await expect(decision()).toContainText('No active allow')
      await expect(sources().getByRole('article')).toHaveCount(0)
    } finally { await page.context().addCookies(original); await page.goto(route); await expect(decision()).toContainText('Allowed') }
  })

  await test.step('Continue real bounded scans and discard all earlier sources on a conflict', async () => {
    const children: string[] = []
    for (let n = 0; n < 16; n++) { const id = `scan-leaf-${n}`; children.push(id); await policy(id, [rule(), rule('deny', '2000-01-01T00:00:00Z')]) }
    await policy('scan-composite', [], children)
    await api.call('POST', `${path}/role-assignments`, { principalKind: 'user', principalId: fixture.readerId, roleId: 'scan-composite', reason: 'Verify bounded source traversal' })
    const firstResponse = page.waitForResponse(response => response.url().includes(`/users/${fixture.readerId}/effective-permissions?`) && new URL(response.url()).searchParams.get('permission') === permission && !new URL(response.url()).searchParams.has('cursor'))
    await inspect()
    const firstReport = await (await firstResponse).json()
    await expect(sources()).toContainText('Source scan is incomplete')
    await expect(decision()).toContainText('Allowed')
    const user = await api.call('GET', `/api/v1/users/${fixture.readerId}`)
    await api.call('PUT', `/api/v1/users/${fixture.readerId}`, { id: fixture.readerId, name: 'Updated inspected user', email: user.data.email || '' }, user.revision)
    const query = new URLSearchParams({ project, object: 'suite/health', permission, limit: '30', cursor: firstReport.cursor })
    await expectHTTPFailure(page, 'GET', `/api/v1/users/${fixture.readerId}/effective-permissions?${query}`, 409, () => page.getByRole('button', { name: 'Scan more sources', exact: true }).click())
    await expect(page.getByRole('alert')).toContainText('Access changed during inspection')
    await expect(decision()).toHaveCount(0); await expect(sources()).toHaveCount(0)
    expect(firstReport.partial).toBe(true)
    await check()
    while (await page.getByRole('button', { name: 'Scan more sources', exact: true }).count()) {
      const before = await sources().getByRole('article').count()
      const response = page.waitForResponse(response => response.url().includes(`/users/${fixture.readerId}/effective-permissions?`) && new URL(response.url()).searchParams.has('cursor'))
      await page.getByRole('button', { name: 'Scan more sources', exact: true }).click()
      const next = await (await response).json()
      await expect(sources().getByRole('article')).toHaveCount(before + next.items.length)
      if (!next.partial) await expect(page.getByRole('button', { name: 'Scan more sources', exact: true })).toHaveCount(0)
      else await expect(page.getByRole('button', { name: 'Scan more sources', exact: true })).toBeEnabled()
    }
    await expect(sources()).toContainText('Source scan complete')
    await expect(sources().getByRole('article')).toHaveCount(36)
  })

  await test.step('Remove private reports immediately when administrative inspection is revoked', async () => {
    const assignments = await api.call('GET', `${path}/role-assignments`)
    const assignment = assignments.items.find((item: any) => item.data.principalId === fixture.userId && item.data.roleId === 'builtin.teststudio.project-admin')
    expect(assignment).toBeTruthy()
    try {
      await page.context().addCookies([{ name: 'teststudio_session', value: fixture.cookie, domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax', secure: false }])
      await page.goto(`${route}?user=${fixture.readerId}&object=suite%2Fhealth&permission=${permission}`)
      await expect(decision()).toContainText('Allowed')
      await expect(sources().getByRole('article')).not.toHaveCount(0)
      const deleted = await page.request.delete(`${path}/role-assignments/user/${fixture.userId}/builtin.teststudio.project-admin`, { headers: { ...rootHeaders, 'If-Match': `"${assignment.revision}"` }, data: { reason: 'Withdraw inspection while private sources are visible' } }); expect(deleted.ok()).toBe(true)
      const query = new URLSearchParams({ project, object: 'suite/health', permission, limit: '30' })
      await expectHTTPFailure(page, 'GET', `/api/v1/users/${fixture.readerId}/effective-permissions?${query}`, 403, () => page.getByRole('button', { name: 'Check access', exact: true }).click())
      await expect(decision()).toHaveCount(0); await expect(sources()).toHaveCount(0)
      await expect(page.getByLabel('Inspect user identifier', { exact: true })).toHaveAttribute('readonly', '')
    } finally { await page.context().addCookies(original) }
  })
})
