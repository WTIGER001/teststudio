import { test, expect, expectHTTPFailure } from './fixtures'

test('Custom policies, inherited assignment, access previews, and guarded editors', { tag: '@STUDIO-POLICY-001' }, async ({ page, api }, testInfo) => {
  const { project, path } = await api.scaffold()
  const rolesPath = `/projects/${project}/roles`, child = 'health-reader', parent = 'composite-reader'
  const prepared = await page.request.post('http://127.0.0.1:8050/access/prepare', { data: { project } })
  expect(prepared.ok()).toBe(true)
  const fixture = await prepared.json()
  await api.call('POST', '/api/v1/groups', { id: 'policy-reviewers', name: 'Policy reviewers', projectId: project })
  await api.call('PUT', `/api/v1/groups/policy-reviewers/members/${fixture.readerId}`, {}, 0)
  const editor = () => page.getByRole('region', { name: /^(Create custom role|Edit role policy)$/ })
  const previewRow = () => page.getByRole('region', { name: 'Access preview results', exact: true }).getByRole('row').filter({ hasText: 'teststudio.catalog.read' })
  const preview = async () => {
    await page.getByLabel('Preview user identifier', { exact: true }).fill(fixture.readerId)
    await page.getByLabel('Preview exact object (optional)', { exact: true }).fill('suite/health')
    await page.getByRole('button', { name: 'Preview access', exact: true }).click()
    await expect(page.getByText('Preview complete. No policy, assignment, or audit change was saved.', { exact: true })).toBeVisible()
    await expect(previewRow().getByRole('cell').nth(0)).toHaveText('Denied')
    await expect(previewRow().getByRole('cell').nth(1)).toHaveText('Allowed')
  }
  const help = async (title: string) => {
    await page.getByRole('button', { name: 'Help', exact: true }).click()
    const drawer = page.getByRole('dialog', { name: `${title} help`, exact: true })
    for (const name of ['Introduction', 'Concepts', 'Tutorials']) await expect(drawer.getByRole('heading', { name, exact: true })).toBeVisible()
    await page.keyboard.press('Escape')
  }
  const rule = async (index: number, effect: 'allow' | 'deny', expiry = '') => {
    await page.getByRole('button', { name: '+ Add permission rule', exact: true }).click()
    const row = page.getByRole('group', { name: `Permission rule ${index}`, exact: true })
    await row.getByLabel('Exact object (optional)', { exact: true }).fill('suite/health')
    await row.getByRole('button', { name: 'Refresh rule permissions', exact: true }).click()
    await row.getByLabel('Permission', { exact: true }).selectOption('teststudio.catalog.read')
    await row.getByLabel('Effect', { exact: true }).selectOption(effect)
    await row.getByLabel('Expiry in UTC (optional)', { exact: true }).fill(expiry)
  }
  const save = async (reason: string) => {
    await page.getByLabel('Policy change reason', { exact: true }).fill(reason)
    await page.getByRole('button', { name: 'Save role policy', exact: true }).click()
    await expect(editor()).toHaveCount(0)
  }

  await test.step('Create an exact policy, preview without publication, and preserve help edits', async () => {
    await page.goto(rolesPath)
    await page.getByRole('button', { name: '+ Create role', exact: true }).click()
    await page.getByLabel('Role name', { exact: true }).fill('Health reader')
    await page.getByLabel('Role identifier', { exact: true }).fill(child)
    await page.getByLabel('Role description (optional)', { exact: true }).fill('Exact health catalog inspection')
    await rule(1, 'allow')
    await preview()
    await page.getByLabel('Show unchanged permissions', { exact: true }).check()
    await expect.poll(() => page.getByRole('region', { name: 'Access preview results', exact: true }).getByRole('row').count()).toBeGreaterThan(10)
    await page.getByLabel('Show unchanged permissions', { exact: true }).uncheck()
    await expect(page.getByRole('region', { name: 'Access preview results', exact: true }).getByRole('row')).toHaveCount(2)
    expect((await page.request.get(`/api/v1/roles/${child}`)).status()).toBe(404)
    await help('Roles and permissions')
    await expect(page.getByLabel('Role name', { exact: true })).toHaveValue('Health reader')
    await page.getByLabel('Role name', { exact: true }).fill('Changed preview input')
    await expect(page.getByText('Preview is out of date. Run it again for the current fields.', { exact: true })).toBeVisible()
    await expect(page.getByRole('region', { name: 'Access preview results', exact: true })).toHaveCount(0)
    await page.getByLabel('Role name', { exact: true }).fill('Health reader')
    await save('Create an exact health reader')
    await page.getByRole('table', { name: 'Managed custom role catalog', exact: true }).getByRole('link', { name: 'Health reader', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Role details', exact: true })).toBeVisible()
    await expect(page.getByRole('region', { name: 'Stored permission rules', exact: true })).toContainText('suite/health')
  })

  await test.step('Retain opened revisions and unsaved values through a concurrent update', async () => {
    await page.getByRole('button', { name: 'Edit role policy', exact: true }).click()
    await page.getByLabel('Role description (optional)', { exact: true }).fill('Keep these unsaved notes')
    await page.getByLabel('Policy change reason', { exact: true }).fill('Keep this conflict reason')
    const stored = await api.call('GET', `/api/v1/roles/${child}`)
    await api.call('PUT', `/api/v1/roles/${child}`, { ...stored.data, name: 'Concurrent reader', reason: 'Concurrent policy editor' }, stored.revision)
    await page.getByRole('button', { name: 'Refresh role', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Concurrent reader', exact: true })).toBeVisible()
    await expect(page.getByLabel('Role name', { exact: true })).toHaveValue('Health reader')
    await expectHTTPFailure(page, 'PUT', `/api/v1/roles/${child}`, 409, () => page.getByRole('button', { name: 'Save role policy', exact: true }).click())
    await expect(page.getByLabel('Role description (optional)', { exact: true })).toHaveValue('Keep these unsaved notes')
    await expect(page.getByLabel('Policy change reason', { exact: true })).toHaveValue('Keep this conflict reason')
    await expect(page.getByRole('button', { name: 'Save role policy', exact: true })).toBeDisabled()
    await page.getByRole('button', { name: 'Reload stored role', exact: true }).click()
    await expect(page.getByLabel('Role name', { exact: true })).toHaveValue('Concurrent reader')
    await expect(page.getByLabel('Role identifier', { exact: true })).toHaveAttribute('readonly', '')
    await page.getByLabel('Role name', { exact: true }).fill('Health reader final')
    await rule(2, 'deny', '2000-01-01T00:00:00.123456Z')
    await preview()
    await help('Role details')
    for (const theme of ['light', 'dark']) {
      await page.getByLabel('Color theme').selectOption(theme)
      await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await testInfo.attach(`policy-editor-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' })
    }
    await page.setViewportSize({ width: 1280, height: 900 })
    await save('Retain the expired deny for inspection')
    await expect(page.getByRole('region', { name: 'Stored permission rules', exact: true })).toContainText('Deny')
    for (const theme of ['light', 'dark']) {
      await page.getByLabel('Color theme').selectOption(theme)
      await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await testInfo.attach(`policy-details-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' })
    }
    await page.setViewportSize({ width: 1280, height: 900 })
  })

  await test.step('Create inheritance, assign through the UI, and withdraw and restore real access', async () => {
    await page.goto(rolesPath)
    await page.getByRole('button', { name: '+ Create role', exact: true }).click()
    await page.getByLabel('Role name', { exact: true }).fill('Composite reader')
    await page.getByLabel('Role identifier', { exact: true }).fill(parent)
    await page.getByRole('button', { name: '+ Inherit role', exact: true }).click()
    await page.getByLabel('Inherited role 1', { exact: true }).selectOption(child)
    await preview()
    await save('Compose the exact reader policy')
    await page.goto(`/projects/${project}/members`)
    await page.getByRole('button', { name: '+ Add role', exact: true }).click()
    await page.getByLabel('Principal type', { exact: true }).selectOption('group')
    await page.getByLabel('Registered principal identifier', { exact: true }).fill('policy-reviewers')
    await page.getByLabel('Role to assign', { exact: true }).selectOption(parent)
    await page.getByLabel('Assignment reason', { exact: true }).fill('Grant inherited group access')
    await page.getByRole('button', { name: 'Assign role', exact: true }).click()
    await expect(page.getByRole('table', { name: 'Role assignments in the selected scope', exact: true }).getByRole('row').filter({ hasText: 'Policy reviewers' })).toContainText('Composite reader')
    const health = `${path}/suites/health/catalog-drafts`, reader = { headers: { Cookie: `teststudio_session=${fixture.readerCookie}` } }
    expect((await page.request.get(health, reader)).status()).toBe(200)
    expect((await page.request.get(`${path}/suites/other/catalog-drafts`, reader)).status()).toBe(403)
    expect((await page.request.get(`${path}/suites/health/tests/missing`, reader)).status()).toBe(403)
    await page.goto(`${rolesPath}/${parent}`)
    await page.getByRole('button', { name: 'Disable role', exact: true }).click()
    await page.getByLabel('Reason', { exact: true }).fill('Withdraw inherited access temporarily')
    await page.getByRole('button', { name: 'Confirm action', exact: true }).click()
    await expect(page.getByRole('region', { name: 'Composite reader', exact: true })).toContainText('Disabled')
    expect((await page.request.get(health, reader)).status()).toBe(403)
    await page.getByRole('button', { name: 'Enable role', exact: true }).click()
    await page.getByLabel('Reason', { exact: true }).fill('Restore the reviewed retained policy')
    await page.getByRole('button', { name: 'Confirm action', exact: true }).click()
    await expect(page.getByRole('region', { name: 'Composite reader', exact: true })).toContainText('Enabled')
    expect((await page.request.get(health, reader)).status()).toBe(200)
  })

  await test.step('Apply real read-only and foreign-scope controls without private editing requests', async () => {
    const original = await page.context().cookies('http://127.0.0.1:8040')
    try {
      await page.context().addCookies([{ name: 'teststudio_session', value: fixture.readerCookie, domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax', secure: false }])
      await page.goto(`${rolesPath}/${parent}`)
      await expect(page.getByRole('heading', { name: 'Composite reader', exact: true })).toBeVisible()
      await expect(page.getByRole('button', { name: 'Edit role policy', exact: true })).toHaveCount(0)
      await expect(page.getByRole('button', { name: 'Disable role', exact: true })).toHaveCount(0)
      await expect(page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'Installation roles', exact: true, includeHidden: true })).toHaveCount(0)
      await page.goto('/settings/roles')
      await expect(page.getByRole('alert')).toContainText('Role read authority is required')
      await expect(page.getByRole('button', { name: '+ Create role', exact: true })).toHaveCount(0)
    } finally { await page.context().addCookies(original); await page.goto(rolesPath) }
  })

  await test.step('Clear an open private editor when preview authority is revoked', async () => {
    await api.call('POST', '/api/v1/roles', { id: 'policy-management', name: 'Policy management', projectId: project, rules: [{ permission: 'teststudio.roles.manage', effect: 'allow' }], inherits: [], reason: 'Delegate the role editor to the scoped manager' })
    await api.call('POST', `${path}/role-assignments`, { principalKind: 'user', principalId: fixture.userId, roleId: 'policy-management', reason: 'Allow actual scoped policy editing' })
    const managerAssignment = `${path}/role-assignments/user/${fixture.userId}/builtin.teststudio.project-admin`
    const assigned = await api.call('GET', managerAssignment)
    const original = await page.context().cookies('http://127.0.0.1:8040'), root = original.find(cookie => cookie.name === 'teststudio_session')!
    const rootCSRF = (await (await page.request.get('/auth/session')).json()).session.csrfToken
    try {
      await page.context().addCookies([{ name: 'teststudio_session', value: fixture.cookie, domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax', secure: false }])
      await page.goto(`${rolesPath}/${child}`)
      await page.getByRole('button', { name: 'Edit role policy', exact: true }).click()
      await page.getByLabel('Role description (optional)', { exact: true }).fill('Private unsaved policy review')
      await page.getByLabel('Preview user identifier', { exact: true }).fill(fixture.readerId)
      await page.getByLabel('Preview exact object (optional)', { exact: true }).fill('suite/health')
      await page.waitForLoadState('networkidle')
      const revoked = await page.request.delete(managerAssignment, { headers: { Cookie: `${root.name}=${root.value}`, Origin: 'http://127.0.0.1:8040', 'X-CSRF-Token': rootCSRF, 'If-Match': `"${assigned.revision}"` }, data: { reason: 'Revoke preview and role read authority during editing' } })
      expect(revoked.ok()).toBe(true)
      await expectHTTPFailure(page, 'POST', '/api/v1/roles/preview', 403, () => page.getByRole('button', { name: 'Preview access', exact: true }).click())
      await expect(page.getByText('Role read authority is required for this scope.', { exact: true })).toBeVisible()
      await expect(editor()).toHaveCount(0)
      await expect(page.getByRole('heading', { name: 'Health reader final', exact: true })).toHaveCount(0)
    } finally { await page.context().addCookies(original); await page.goto(rolesPath) }
  })

  await test.step('Continue bounded catalogs and retain the layout in both themes', async () => {
    const total = (await api.call('GET', `/api/v1/roles/custom?project=${project}&limit=1000`)).items.length
    for (let i = 0; i < 31; i++) await api.call('POST', '/api/v1/roles', { id: `a-page-${String(i).padStart(2, '0')}`, name: `Paged policy ${i}`, projectId: project, rules: [], inherits: [], reason: 'Verify a bounded policy catalog' })
    await page.getByRole('button', { name: 'Refresh roles', exact: true }).click()
    await expect(page.getByRole('table', { name: 'Managed custom role catalog', exact: true }).getByRole('row')).toHaveCount(31)
    await page.getByRole('button', { name: 'More custom roles', exact: true }).click()
    await expect(page.getByRole('table', { name: 'Managed custom role catalog', exact: true }).getByRole('row')).toHaveCount(total + 32)
    for (const theme of ['light', 'dark']) {
      await page.getByLabel('Color theme').selectOption(theme)
      await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await testInfo.attach(`policy-catalog-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' })
    }
    await page.setViewportSize({ width: 1280, height: 900 })
  })

  await test.step('Reject last-administrator policy disabling through the actual confirmation form', async () => {
    const id = 'safety-administrator'
    await api.call('POST', '/api/v1/roles', { id, name: 'Safety administrator', projectId: '', rules: ['system.manage', 'users.manage', 'permissions.grant', 'permissions.read', 'roles.manage', 'roles.read', 'members.manage', 'members.read'].map(action => ({ permission: `teststudio.${action}`, effect: 'allow' })), inherits: [], reason: 'Verify the last custom administrator guard' })
    await api.call('POST', '/api/v1/role-assignments', { principalKind: 'user', principalId: 'local-administrator', roleId: id, reason: 'Retain an explicit custom administrator' })
    const root = '/api/v1/role-assignments/user/local-administrator/builtin.teststudio.installation-admin', current = await api.call('GET', root)
    await api.call('DELETE', root, { reason: 'Verify recovery with the custom role' }, current.revision)
    await page.goto(`/settings/roles/${id}`)
    await page.getByRole('button', { name: 'Disable role', exact: true }).click()
    await page.getByLabel('Reason', { exact: true }).fill('Reject the final usable administrator change')
    await expectHTTPFailure(page, 'PUT', `/api/v1/roles/${id}`, 422, () => page.getByRole('button', { name: 'Confirm action', exact: true }).click())
    await expect(page.getByRole('alert')).toContainText('last usable installation administrator')
    await page.getByRole('button', { name: 'Cancel action', exact: true }).click()
    await expect(page.getByRole('region', { name: 'Safety administrator', exact: true })).toContainText('Enabled')
    await api.call('POST', '/api/v1/role-assignments', { principalKind: 'user', principalId: 'local-administrator', roleId: 'builtin.teststudio.installation-admin', reason: 'Restore the isolated verification administrator' })
  })
})
