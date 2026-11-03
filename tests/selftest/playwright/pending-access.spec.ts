import { test, expect, expectHTTPFailure } from './fixtures'

test('Pending project access, verified provider claims, and retained approvals', { tag: '@STUDIO-PENDING-001' }, async ({ page, api }, testInfo) => {
  const { project, path } = await api.scaffold(), other = await api.scaffold(), base = `${path}/pending-access`, route = `/projects/${project}/members`
  for (const [id, name, issuer] of [['pending-one', 'Pending identity one', 'one'], ['pending-two', 'Pending identity two', 'two']]) await api.call('POST', '/api/v1/identity-providers', { id, name, issuer: `http://127.0.0.1:8050/oidc/${issuer}`, clientId: 'teststudio-public', scopes: ['openid', 'email', 'profile'], disabled: false, autoProvision: true, groupMappings: {}, membershipTtlSeconds: 0, reason: 'Verify pending access with independent local issuers' })
  const original = await page.context().cookies('http://127.0.0.1:8040'), root = original.find(cookie => cookie.name === 'teststudio_session')!, csrf = (await (await page.request.get('/auth/session')).json()).session.csrfToken
  const rootCall = async (method: string, resource: string, body?: unknown, revision?: number) => {
    const response = await page.request.fetch(resource, { method, data: body, headers: { Cookie: `${root.name}=${root.value}`, Origin: 'http://127.0.0.1:8040', 'X-CSRF-Token': csrf, ...(revision === undefined ? {} : { 'If-Match': `"${revision}"` }) } })
    expect(response.ok(), `${method} ${resource}: ${await response.text()}`).toBe(true); return response.json()
  }
  const prepared = await page.request.post('http://127.0.0.1:8050/access/prepare', { data: { project } }); expect(prepared.ok()).toBe(true)
  const fixture = await prepared.json()
  const manager = async () => { await page.context().addCookies([{ name: 'teststudio_session', value: fixture.cookie, domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax', secure: false }]); await page.goto(route); await expect(page.getByRole('heading', { name: 'Project access', exact: true })).toBeVisible() }
  const table = () => page.getByRole('table', { name: 'Pending project role approvals', exact: true })
  const row = (email: string, state: string) => table().getByRole('row').filter({ hasText: email }).filter({ has: page.getByText(state, { exact: true }) })
  const editor = () => page.getByRole('region', { name: 'Prepare project access', exact: true })
  const fill = async (email = 'same@example.test') => { await page.getByLabel('Allowed sign-in provider', { exact: true }).selectOption('pending-one'); await page.getByLabel('Required verified email', { exact: true }).fill(email); await page.getByLabel('Pending project role', { exact: true }).selectOption('builtin.teststudio.viewer'); await page.getByLabel('Pending access reason', { exact: true }).fill('Owner approval for a future verified sign-in') }
  const updateProvider = async (name: string) => { const stored = await rootCall('GET', '/api/v1/identity-providers/pending-one'); const { canManage, callbackUrl, ...definition } = stored.data; await rootCall('PUT', '/api/v1/identity-providers/pending-one', { ...definition, name, reason: 'Review a changed provider configuration' }, stored.revision) }
  const signIn = async (name: string, subject: string, verified = true, email = 'same@example.test') => {
    await page.context().clearCookies(); await page.goto('/projects')
    await page.getByRole('link', { name: `Continue with ${name}`, exact: true }).click()
    await page.getByLabel('Fixture subject', { exact: true }).fill(subject)
    await page.getByLabel('Fixture email', { exact: true }).fill(email)
    if (!verified) await page.getByLabel('Fixture verified email', { exact: true }).uncheck()
    await page.getByRole('button', { name: 'Confirm fixture sign-in', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
    return (await (await page.request.get('/auth/session')).json()).user
  }
  let pendingID = '', targetUser = ''

  await test.step('Project owner prepares access without installation directory authority and retains its opened provider revision', async () => {
    await manager()
    await page.getByRole('button', { name: '+ Prepare project access', exact: true }).click(); await fill()
    await expect(page.getByLabel('Pending project role', { exact: true }).getByRole('option', { name: 'Installation administrator', exact: true })).toHaveCount(0)
    await updateProvider('Pending identity one reviewed')
    await page.getByRole('button', { name: 'Refresh pending access', exact: true }).click()
    await expect(editor()).toContainText('Configuration revision 1')
    await expectHTTPFailure(page, 'POST', base, 409, () => page.getByRole('button', { name: 'Prepare access', exact: true }).click())
    await expect(page.getByLabel('Pending access reason', { exact: true })).toHaveValue('Owner approval for a future verified sign-in')
    await expect(page.getByRole('button', { name: 'Prepare access', exact: true })).toBeDisabled()
    await page.getByRole('button', { name: 'Review pending records and reload choices', exact: true }).click()
    await expect(editor()).toContainText('Configuration revision 2')
    await page.getByRole('button', { name: 'Help', exact: true }).click()
    const help = page.getByRole('dialog', { name: 'Project access help', exact: true })
    for (const title of ['Introduction', 'Concepts', 'Tutorials']) await expect(help.getByRole('heading', { name: title, exact: true })).toBeVisible()
    await expect(help).toContainText('Pending project access'); await page.keyboard.press('Escape')
    for (const theme of ['light', 'dark']) {
      await page.getByLabel('Color theme').selectOption(theme)
      await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await testInfo.attach(`pending-access-editor-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' })
    }
    await page.setViewportSize({ width: 1280, height: 900 })
    await page.getByRole('button', { name: 'Prepare access', exact: true }).focus(); await page.keyboard.press('Enter')
    await expect(editor()).toHaveCount(0); await expect(row('same@example.test', 'pending')).toBeVisible()
  })

  await test.step('Preserve stale cancellation and unreadable preparation outcomes for explicit review', async () => {
    const first = (await rootCall('GET', base)).items[0]
    await row('same@example.test', 'pending').getByRole('button', { name: 'Cancel pending access for same@example.test', exact: true }).click()
    await page.getByLabel('Reason', { exact: true }).fill('Keep this cancellation review')
    await rootCall('POST', `${base}/${first.id}/cancel`, { reason: 'Concurrent owner cancellation' }, first.revision)
    await expectHTTPFailure(page, 'POST', `${base}/${first.id}/cancel`, 409, () => page.getByRole('button', { name: 'Confirm action', exact: true }).click())
    await expect(page.getByRole('button', { name: 'Confirm action', exact: true })).toBeDisabled()
    await expect(page.getByLabel('Reason', { exact: true })).toHaveValue('Keep this cancellation review')
    await page.getByRole('button', { name: 'Cancel action', exact: true }).click(); await page.getByRole('button', { name: 'Refresh pending access', exact: true }).click()
    await expect(row('same@example.test', 'cancelled')).toBeVisible()
    await page.getByRole('button', { name: '+ Prepare project access', exact: true }).click(); await fill()
    let submissions = 0
    await page.route(`**${base}`, async intercepted => { if (intercepted.request().method() !== 'POST') return intercepted.continue(); submissions++; const response = await intercepted.fetch(); expect(response.ok()).toBe(true); await intercepted.fulfill({ response, contentType: 'application/json', body: 'unreadable successful approval' }) })
    try {
      await page.getByRole('button', { name: 'Prepare access', exact: true }).click()
      await expect(editor().getByRole('alert')).toContainText('unreadable response')
      await expect(page.getByRole('button', { name: 'Prepare access', exact: true })).toBeDisabled()
      expect(submissions).toBe(1)
    } finally { await page.unroute(`**${base}`) }
    await page.getByRole('button', { name: 'Review pending records and reload choices', exact: true }).click()
    await expect(row('same@example.test', 'pending')).toBeVisible()
    await page.getByRole('button', { name: 'Close pending access editor', exact: true }).click()
    pendingID = (await rootCall('GET', base)).items.find((item: any) => item.data.state === 'pending').id
  })

  await test.step('Require the exact allowed provider and its verified email, then consume access only once', async () => {
    const otherIdentity = await signIn('Pending identity two', 'same-subject')
    await expect(page.getByRole('heading', { name: 'Access not granted', exact: true })).toBeVisible()
    expect((await rootCall('GET', `${base}/${pendingID}`)).data.state).toBe('pending')
    const unverified = await signIn('Pending identity one reviewed', 'same-subject', false)
    await expect(page.getByRole('heading', { name: 'Access not granted', exact: true })).toBeVisible()
    expect(unverified.id).not.toBe(otherIdentity.id)
    expect((await rootCall('GET', `${base}/${pendingID}`)).data.state).toBe('pending')
    const verified = await signIn('Pending identity one reviewed', 'same-subject')
    expect(verified.id).toBe(unverified.id); targetUser = verified.id
    await expect(page.getByRole('link', { name: /Self-test workspace/ })).toHaveCount(1)
    expect((await page.request.get(other.path)).status()).toBe(403)
    const activated = await rootCall('GET', `${base}/${pendingID}`)
    expect(activated.data.state).toBe('activated'); expect(activated.data.activatedUserId).toBe(targetUser)
    expect((await rootCall('GET', `/api/v1/users/${otherIdentity.id}/identities`)).items).toHaveLength(1)
    const assignmentPath = `${path}/role-assignments/user/${targetUser}/builtin.teststudio.viewer`, assignment = await rootCall('GET', assignmentPath)
    await rootCall('DELETE', assignmentPath, { reason: 'Revoke the consumed pending role' }, assignment.revision)
    await signIn('Pending identity one reviewed', 'same-subject')
    await expect(page.getByRole('heading', { name: 'Access not granted', exact: true })).toBeVisible()
    expect((await rootCall('GET', `${base}/${pendingID}`)).data.state).toBe('activated')
  })

  await test.step('Show blocked approvals and clear private edits when owner authority changes', async () => {
    await manager()
    await expect(row('same@example.test', 'activated')).toBeVisible()
    await page.getByRole('button', { name: '+ Prepare project access', exact: true }).click(); await fill('blocked@example.test')
    await page.getByRole('button', { name: 'Prepare access', exact: true }).click(); await expect(editor()).toHaveCount(0)
    await updateProvider('Pending identity one current')
    await signIn('Pending identity one current', 'blocked-subject', true, 'blocked@example.test')
    await expect(page.getByRole('heading', { name: 'Access not granted', exact: true })).toBeVisible()
    await manager()
    await expect(row('blocked@example.test', 'blocked')).toContainText('Provider configuration changed')
    await rootCall('POST', '/api/v1/roles', { id: 'pending-reader', name: 'Pending access reader', projectId: project, description: '', disabled: false, rules: ['projects.read', 'members.read', 'permissions.read', 'roles.read'].map(name => ({ permission: `teststudio.${name}`, effect: 'allow' })), inherits: [], reason: 'Keep read access after removing delegation' })
    await rootCall('POST', `${path}/role-assignments`, { principalKind: 'user', principalId: fixture.userId, roleId: 'pending-reader', reason: 'Retain explicit project access inspection' })
    await page.getByRole('button', { name: '+ Prepare project access', exact: true }).click(); await fill('private-edit@example.test')
    const ownerPath = `${path}/role-assignments/user/${fixture.userId}/builtin.teststudio.project-admin`, owner = await rootCall('GET', ownerPath)
    await rootCall('DELETE', ownerPath, { reason: 'Remove future access delegation' }, owner.revision)
    await expectHTTPFailure(page, 'POST', base, 403, () => page.getByRole('button', { name: 'Prepare access', exact: true }).click())
    await expect(editor()).toHaveCount(0)
    await page.getByRole('button', { name: 'Refresh access', exact: true }).click()
    await expect(table()).toBeVisible()
    await expect(page.getByRole('button', { name: '+ Prepare project access', exact: true })).toHaveCount(0)
    await expect(table().getByRole('button')).toHaveCount(0)
    await expect(page.getByText('private-edit@example.test', { exact: true })).toHaveCount(0)
  })
})
