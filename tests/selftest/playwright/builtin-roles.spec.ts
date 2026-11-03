import { test, expect, expectHTTPFailure } from './fixtures'

test('Predefined role state, retained assignments, and installation administration guards', { tag: '@STUDIO-BUILTIN-001' }, async ({ page, api }, testInfo) => {
  const first = await api.scaffold(), second = await api.scaffold(`builtin-other-${crypto.randomUUID().slice(0, 8)}`)
  const response = await page.request.post('http://127.0.0.1:8050/access/prepare', { data: { project: first.project } }); expect(response.ok()).toBe(true)
  const fixture = await response.json(), id = 'builtin.teststudio.editor', endpoint = `/api/v1/predefined-roles/${id}`
  for (const path of [first.path, second.path]) await api.call('POST', `${path}/role-assignments`, { principalKind: 'user', principalId: fixture.readerId, roleId: id, reason: 'Verify retained role assignments in independent projects' })
  const reader = { headers: { Cookie: `teststudio_session=${fixture.readerCookie}` } }
  const card = (name = 'Catalog editor') => page.getByRole('region', { name, exact: true })
  const action = () => page.getByRole('region', { name: /^(Disable predefined role|Enable predefined role)$/ })
  const check = async (status: number) => { for (const path of [first.path, second.path]) expect((await page.request.get(`${path}/suites/health/catalog-drafts`, reader)).status()).toBe(status) }
  const confirm = async (reason: string) => { await page.getByLabel('Reason', { exact: true }).fill(reason); await page.getByRole('button', { name: 'Confirm action', exact: true }).click(); await expect(action()).toHaveCount(0) }
  const state = async (disabled: boolean) => { await card().getByRole('button', { name: disabled ? 'Disable predefined role' : 'Enable predefined role', exact: true }).click(); await confirm(disabled ? 'Withdraw this role across the installation' : 'Restore reviewed retained access'); await expect(card()).toContainText(disabled ? 'Disabled' : 'Enabled') }

  await test.step('Withdraw access from both projects while retaining assignments and definitions', async () => {
    await page.goto('/settings/roles')
    await check(200)
    const before = await api.call('GET', endpoint)
    await card().getByRole('button', { name: 'Disable predefined role', exact: true }).click()
    await expect(action()).toContainText('Entire installation')
    await page.getByLabel('Reason', { exact: true }).fill('Keep this predefined role review')
    await page.getByRole('button', { name: 'Help', exact: true }).click()
    const help = page.getByRole('dialog', { name: 'Roles and permissions help', exact: true })
    for (const name of ['Introduction', 'Concepts', 'Tutorials']) await expect(help.getByRole('heading', { name, exact: true })).toBeVisible()
    await expect(help).toContainText('installation-wide disabled-state controls')
    await expect(help).toContainText('Change a predefined role state')
    await page.keyboard.press('Escape')
    await expect(page.getByLabel('Reason', { exact: true })).toHaveValue('Keep this predefined role review')
    await confirm('Disable a retained shared role')
    await expect(card()).toContainText('Disabled')
    await check(403)
    const after = await api.call('GET', endpoint)
    expect(after.data.permissions).toEqual(before.data.permissions)
    expect(after.revision).toBeGreaterThan(before.revision)
    for (const path of [first.path, second.path]) {
      const assignments = await api.call('GET', `${path}/role-assignments`)
      expect(assignments.items.some((row: any) => row.data.principalId === fixture.readerId && row.data.roleId === id && row.data.roleDisabled)).toBe(true)
    }
    await page.goto(`/projects/${first.project}/members`)
    await page.getByRole('button', { name: '+ Add role', exact: true }).click()
    await expect(page.getByLabel('Role to assign', { exact: true }).locator(`option[value="${id}"]`)).toBeDisabled()
    await page.goto(`/projects/${first.project}/roles`)
    await expect(page.getByRole('heading', { name: 'Catalog editor', exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Enable predefined role', exact: true })).toHaveCount(0)
    await page.goto('/settings/roles')
    await state(false)
    await check(200)
  })

  await test.step('Retain the opened revision and reason through a concurrent state review', async () => {
    const before = await api.call('GET', endpoint)
    await card().getByRole('button', { name: 'Disable predefined role', exact: true }).click()
    await page.getByLabel('Reason', { exact: true }).fill('Keep this conflict reason')
    await api.call('PUT', endpoint, { disabled: false, reason: 'Concurrent explicit enabled-state review' }, before.revision)
    await page.getByRole('button', { name: 'Refresh predefined roles', exact: true }).click()
    await expectHTTPFailure(page, 'PUT', endpoint, 409, () => page.getByRole('button', { name: 'Confirm action', exact: true }).click())
    await expect(page.getByLabel('Reason', { exact: true })).toHaveValue('Keep this conflict reason')
    await expect(page.getByRole('button', { name: 'Confirm action', exact: true })).toBeDisabled()
    await page.getByRole('button', { name: 'Cancel action', exact: true }).click()
    await page.getByRole('button', { name: 'Refresh predefined roles', exact: true }).click()
    await state(true); await check(403); await state(false); await check(200)
  })

  await test.step('Require explicit review after an unreadable successful state change', async () => {
    let submissions = 0
    await card().getByRole('button', { name: 'Disable predefined role', exact: true }).click()
    await page.getByLabel('Reason', { exact: true }).fill('Keep this unconfirmed state reason')
    await page.route(`**${endpoint}`, async intercepted => { if (intercepted.request().method() !== 'PUT') return intercepted.continue(); submissions++; const response = await intercepted.fetch(); expect(response.status()).toBe(200); await intercepted.fulfill({ status: 200, contentType: 'application/json', body: 'unreadable state response' }) })
    try {
      await page.getByRole('button', { name: 'Confirm action', exact: true }).click()
      await expect(page.getByRole('alert')).toContainText('The server returned an unreadable response')
      await expect(page.getByRole('button', { name: 'Confirm action', exact: true })).toBeDisabled()
      await expect(page.getByLabel('Reason', { exact: true })).toHaveValue('Keep this unconfirmed state reason')
      expect((await api.call('GET', endpoint)).data.disabled).toBe(true)
      expect(submissions).toBe(1)
    } finally { await page.unroute(`**${endpoint}`) }
    await page.getByRole('button', { name: 'Cancel action', exact: true }).click()
    await page.getByRole('button', { name: 'Refresh predefined roles', exact: true }).click()
    await state(false)
  })

  await test.step('Separate installation inspection from state-management authority', async () => {
    await api.call('POST', '/api/v1/permissions/grants', { principalKind: 'user', principalId: fixture.readerId, permission: 'teststudio.roles.read', effect: 'allow', reason: 'Permit code-definition inspection without global mutation' })
    const original = await page.context().cookies('http://127.0.0.1:8040')
    try {
      await page.context().addCookies([{ name: 'teststudio_session', value: fixture.readerCookie, domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax', secure: false }])
      await page.goto('/settings/roles')
      await expect(card()).toContainText('Read only')
      await expect(page.getByRole('button', { name: 'Disable predefined role', exact: true })).toHaveCount(0)
      await expect(page.getByRole('button', { name: '+ Create role', exact: true })).toHaveCount(0)
      await page.context().addCookies([{ name: 'teststudio_session', value: fixture.cookie, domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax', secure: false }])
      await page.goto('/settings/roles')
      await expect(page.getByRole('alert')).toContainText('Role read authority is required')
      expect((await page.request.get('/api/v1/predefined-roles')).status()).toBe(403)
    } finally { await page.context().addCookies(original); await page.goto('/settings/roles') }
  })

  await test.step('Clear private controls when global administration is revoked during review', async () => {
    const permissions = (await api.call('GET', '/api/v1/permissions')).items.map((item: any) => ({ permission: item.id, effect: 'allow' }))
    await api.call('POST', '/api/v1/roles', { id: 'temporary-global-administration', name: 'Temporary global administration', projectId: '', rules: permissions, reason: 'Verify revoked state management with a real session' })
    const original = await page.context().cookies('http://127.0.0.1:8040'), root = original.find(cookie => cookie.name === 'teststudio_session')!
    const csrf = (await (await page.request.get('/auth/session')).json()).session.csrfToken
    for (const actor of [{ id: fixture.userId, cookie: fixture.cookie, readRetained: false }, { id: fixture.readerId, cookie: fixture.readerCookie, readRetained: true }]) {
      const assignment = await api.call('POST', '/api/v1/role-assignments', { principalKind: 'user', principalId: actor.id, roleId: 'temporary-global-administration', reason: 'Temporarily permit actual installation administration' })
      try {
        await page.context().addCookies([{ name: 'teststudio_session', value: actor.cookie, domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax', secure: false }])
        await page.goto('/settings/roles')
        await card().getByRole('button', { name: 'Disable predefined role', exact: true }).click()
        await page.getByLabel('Reason', { exact: true }).fill('Private unsaved global role review')
        const revoked = await page.request.delete(`/api/v1/role-assignments/user/${actor.id}/temporary-global-administration`, { headers: { Cookie: `${root.name}=${root.value}`, Origin: 'http://127.0.0.1:8040', 'X-CSRF-Token': csrf, 'If-Match': `"${assignment.revision}"` }, data: { reason: 'Revoke global administration during state review' } }); expect(revoked.ok()).toBe(true)
        await expectHTTPFailure(page, 'PUT', endpoint, 403, () => page.getByRole('button', { name: 'Confirm action', exact: true }).click())
        await expect(action()).toHaveCount(0)
        await expect(card()).toHaveCount(0)
        if (actor.readRetained) {
          await expect(page.getByRole('alert')).toContainText('Access changed. Refresh predefined roles')
          await page.getByRole('button', { name: 'Refresh predefined roles', exact: true }).click()
          await expect(card()).toContainText('Read only')
          await expect(page.getByRole('button', { name: 'Disable predefined role', exact: true })).toHaveCount(0)
        } else await expect(page.getByRole('alert')).toContainText('Role read authority is required')
      } finally { await page.context().addCookies(original); await page.goto('/settings/roles') }
    }
  })

  await test.step('Protect the final administrator and preserve theme and mobile layouts', async () => {
    await expect(card()).toBeVisible()
    await expect(card('Installation administrator')).toBeVisible()
    await expect(page.getByText('Loading predefined roles…', { exact: true })).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Refresh roles', exact: true })).toBeEnabled()
    for (const theme of ['light', 'dark']) {
      await page.getByLabel('Color theme').selectOption(theme)
      await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
      await card().getByRole('button', { name: 'Disable predefined role', exact: true }).click()
      await page.getByLabel('Reason', { exact: true }).fill('Preserve this themed global state review')
      await expect(action()).toContainText('Entire installation')
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await testInfo.attach(`predefined-roles-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' })
      await page.getByRole('button', { name: 'Cancel action', exact: true }).click()
    }
    await page.setViewportSize({ width: 1280, height: 900 })
    await card('Installation administrator').getByRole('button', { name: 'Disable predefined role', exact: true }).click()
    await page.getByLabel('Reason', { exact: true }).fill('Reject the final usable administrator change')
    await expectHTTPFailure(page, 'PUT', '/api/v1/predefined-roles/builtin.teststudio.installation-admin', 422, () => page.getByRole('button', { name: 'Confirm action', exact: true }).click())
    await expect(page.getByRole('alert')).toContainText('last usable installation administrator')
    await page.getByRole('button', { name: 'Cancel action', exact: true }).click()
    await expect(card('Installation administrator')).toContainText('Enabled')
  })
})
