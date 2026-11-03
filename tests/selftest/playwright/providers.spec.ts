import { test, expect, expectHTTPFailure } from './fixtures'

test('Provider configuration, mapped groups, guarded editors, and two isolated OIDC issuers', { tag: '@STUDIO-PROVIDERS-001' }, async ({ page, api }, testInfo) => {
  const { project, path } = await api.scaffold(), base = '/api/v1/identity-providers'
  const prepared = await page.request.post('http://127.0.0.1:8050/access/prepare', { data: { project } }); expect(prepared.ok()).toBe(true)
  const fixture = await prepared.json(), group = 'provider-reviewers', route = '/settings/providers/local-one'
  let firstUser = '', firstCookie = ''
  const original = await page.context().cookies('http://127.0.0.1:8040'), root = original.find(cookie => cookie.name === 'teststudio_session')!
  const csrf = (await (await page.request.get('/auth/session')).json()).session.csrfToken
  const rootCall = async (method: string, resource: string, body?: unknown, revision?: number) => {
    const response = await page.request.fetch(resource, { method, data: body, headers: { Cookie: `${root.name}=${root.value}`, Origin: 'http://127.0.0.1:8040', 'X-CSRF-Token': csrf, ...(revision === undefined ? {} : { 'If-Match': `"${revision}"` }) } })
    expect(response.ok(), `${method} ${resource}: ${await response.text()}`).toBe(true); return response.json()
  }
  const mutation = (record: any, reason: string) => { const { callbackUrl, logoutCallbackUrl, canManage, ...definition } = record.data; return { ...definition, reason } }
  const editor = () => page.getByRole('region', { name: /^(Register identity provider|Edit identity provider)$/ })
  const stored = () => page.getByRole('region', { name: 'Stored provider', exact: true })
  const save = async (reason: string, button = 'Save provider') => { await page.getByLabel('Provider change reason', { exact: true }).fill(reason); await page.getByRole('button', { name: button, exact: true }).click(); await expect(editor()).toHaveCount(0) }
  const register = async (id: string, name: string, issuer: string, privateClient: boolean) => {
    await page.goto('/settings/providers')
    await page.getByRole('button', { name: '+ Register provider', exact: true }).click()
    await page.getByLabel('Provider identifier', { exact: true }).fill(id)
    await page.getByLabel('Provider name', { exact: true }).fill(name)
    await page.getByLabel('Exact issuer URL', { exact: true }).fill(`http://127.0.0.1:8050/oidc/${issuer}`)
    await page.getByLabel('Client identifier', { exact: true }).fill(privateClient ? 'teststudio-private' : 'teststudio-public')
    if (privateClient) await page.getByLabel('Credential reference', { exact: true }).selectOption('env:TESTSTUDIO_SELFTEST_OIDC_SECRET')
    await page.getByLabel('Create pending user profiles at first sign-in', { exact: true }).check()
    await expect(page.getByLabel('Provider disabled', { exact: true })).toBeChecked()
    await save('Register a new isolated provider', 'Register provider')
    await page.getByRole('table', { name: 'Identity provider configuration', exact: true }).getByRole('link', { name, exact: true }).click()
    await expect(stored()).toContainText(`/auth/callback/${id}`)
  }

  await test.step('Register a private client and create a group with permanent provider ownership', async () => {
    await register('local-one', 'Local identity one', 'one', true)
    await page.getByLabel('New group identifier', { exact: true }).fill(group)
    await page.getByLabel('New group name', { exact: true }).fill('Provider reviewers')
    await page.getByLabel('Group project identifier (optional)', { exact: true }).fill(project)
    await page.getByRole('button', { name: 'Create provider group', exact: true }).click()
    await expect(page.getByRole('status')).toContainText(`Provider group ${group} created`)
    await page.getByRole('button', { name: 'Edit provider', exact: true }).click()
    await page.getByLabel('Group claim name (optional)', { exact: true }).fill('groups')
    await page.getByRole('button', { name: '+ Add group mapping', exact: true }).click()
    await page.getByLabel('External group value 1', { exact: true }).fill('reviewers')
    await page.getByLabel('Destination group identifier 1', { exact: true }).fill(group)
    await page.getByLabel('Provider disabled', { exact: true }).uncheck()
    await page.getByLabel('Provider change reason', { exact: true }).fill('Keep this provider review')
    await page.getByRole('button', { name: 'Help', exact: true }).click()
    const help = page.getByRole('dialog', { name: 'Identity provider help', exact: true })
    for (const name of ['Introduction', 'Concepts', 'Tutorials']) await expect(help.getByRole('heading', { name, exact: true })).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(page.getByLabel('Provider change reason', { exact: true })).toHaveValue('Keep this provider review')
    for (const theme of ['light', 'dark']) {
      await page.getByLabel('Color theme').selectOption(theme)
      await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await page.evaluate(() => scrollTo(0, 0))
      await testInfo.attach(`provider-editor-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' })
    }
    await page.setViewportSize({ width: 1280, height: 900 })
    await save('Enable exact provider group mapping')
    await expect(stored()).toContainText('Enabled')
    await expect(page.getByRole('region', { name: 'Provider group mappings', exact: true })).toContainText(group)
    await rootCall('POST', `${path}/role-assignments`, { principalKind: 'group', principalId: group, roleId: 'builtin.teststudio.viewer', reason: 'Verify provider group access through real sign-in' })
    const g = await rootCall('GET', `/api/v1/groups/${group}`); expect(g.data.managedBy).toBe('oidc:local-one')
  })

  await test.step('Keep opened revisions and block repeat submission after an unconfirmed response', async () => {
    await page.getByRole('button', { name: 'Edit provider', exact: true }).click()
    await page.getByLabel('Provider name', { exact: true }).fill('Private unsaved provider name')
    await page.getByLabel('Provider change reason', { exact: true }).fill('Keep this conflict reason')
    const before = await rootCall('GET', base+'/local-one')
    await rootCall('PUT', base+'/local-one', { ...mutation(before, 'Concurrent provider update'), name: 'Concurrent identity one' }, before.revision)
    await page.getByRole('button', { name: 'Refresh provider', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Concurrent identity one', exact: true })).toBeVisible()
    await expect(page.getByLabel('Provider name', { exact: true })).toHaveValue('Private unsaved provider name')
    await expectHTTPFailure(page, 'PUT', base+'/local-one', 409, () => page.getByRole('button', { name: 'Save provider', exact: true }).click())
    await expect(page.getByRole('button', { name: 'Save provider', exact: true })).toBeDisabled()
    await expect(page.getByLabel('Provider change reason', { exact: true })).toHaveValue('Keep this conflict reason')
    await page.getByRole('button', { name: 'Reload stored provider', exact: true }).click()
    await expect(page.getByLabel('Provider name', { exact: true })).toHaveValue('Concurrent identity one')
    await page.getByLabel('Provider name', { exact: true }).fill('Local identity one')
    await page.getByLabel('Membership lifetime in seconds', { exact: true }).fill('1700')
    await page.getByLabel('Provider change reason', { exact: true }).fill('Review an unconfirmed provider save')
    let submissions = 0
    await page.route(`**${base}/local-one`, async intercepted => { if (intercepted.request().method() !== 'PUT') return intercepted.continue(); submissions++; const response = await intercepted.fetch(); expect(response.ok()).toBe(true); await intercepted.fulfill({ status: 200, contentType: 'application/json', body: 'unreadable provider response' }) })
    try {
      await page.getByRole('button', { name: 'Save provider', exact: true }).click()
      await expect(page.getByRole('alert')).toContainText('unreadable response')
      await expect(page.getByRole('button', { name: 'Save provider', exact: true })).toBeDisabled()
      expect((await rootCall('GET', base+'/local-one')).data.membershipTtlSeconds).toBe(1700)
      expect(submissions).toBe(1)
    } finally { await page.unroute(`**${base}/local-one`) }
    await page.getByRole('button', { name: 'Reload stored provider', exact: true }).click()
    await expect(page.getByLabel('Provider name', { exact: true })).toHaveValue('Local identity one')
    await page.getByRole('button', { name: 'Close provider editor', exact: true }).click()
    await register('local-two', 'Local identity two', 'two', false)
    await page.getByRole('button', { name: 'Edit provider', exact: true }).click()
    await page.getByLabel('Provider disabled', { exact: true }).uncheck()
    await save('Enable an independent public client')
  })

  await test.step('Keep equal emails at different issuers separate and require fresh claims after edits', async () => {
    const signIn = async (name: string) => {
      await page.context().clearCookies(); await page.goto('/projects')
      await page.getByRole('link', { name: `Continue with ${name}`, exact: true }).click()
      await expect(page.getByRole('heading', { name: /^Local provider (one|two)$/ })).toBeVisible()
      await page.getByRole('button', { name: 'Confirm fixture sign-in', exact: true }).click()
      await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
      return (await (await page.request.get('/auth/session')).json()).user
    }
    const first = await signIn('Local identity one')
    await expect(page.getByRole('link', { name: /Self-test workspace/ })).toBeVisible()
    const before = await rootCall('GET', base+'/local-one')
    await rootCall('PUT', base+'/local-one', { ...mutation(before, 'Withdraw old provider authentication'), name: 'Local identity one refreshed' }, before.revision)
    const membership = await rootCall('GET', `/api/v1/groups/${group}/members/${first.id}?source=oidc:local-one`); expect(membership.data.active).toBe(false)
    await page.goto('/projects')
    await expect(page.getByRole('link', { name: 'Continue with Local identity one refreshed', exact: true })).toBeVisible()
    const refreshed = await signIn('Local identity one refreshed'); expect(refreshed.id).toBe(first.id)
    firstUser = first.id; firstCookie = (await page.context().cookies('http://127.0.0.1:8040')).find(cookie => cookie.name === 'teststudio_session')!.value
    await expect(page.getByRole('link', { name: /Self-test workspace/ })).toBeVisible()
    expect((await rootCall('GET', `/api/v1/groups/${group}/members/${first.id}?source=oidc:local-one`)).data.active).toBe(true)
    const second = await signIn('Local identity two')
    expect(second.email).toBe(first.email); expect(second.id).not.toBe(first.id)
    await expect(page.getByRole('heading', { name: 'Access not granted', exact: true })).toBeVisible()
    await expect(page.getByRole('link', { name: /Self-test workspace/ })).toHaveCount(0)
    const links = await rootCall('GET', `/api/v1/users/${second.id}/identities`)
    expect(links.items).toHaveLength(1); expect(links.items[0].data.issuer).toBe('http://127.0.0.1:8050/oidc/two')
    await page.context().addCookies(original)
  })

  await test.step('Refresh public choices after saving the current sign-in provider', async () => {
    const assignment = await rootCall('POST', '/api/v1/role-assignments', { principalKind: 'user', principalId: firstUser, roleId: 'builtin.teststudio.installation-admin', reason: 'Verify explicit administrator self-provider editing' })
    try {
      await page.context().addCookies([{ name: 'teststudio_session', value: firstCookie, domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax', secure: false }])
      await page.goto(route)
      await page.getByRole('button', { name: 'Edit provider', exact: true }).click()
      await page.getByLabel('Provider name', { exact: true }).fill('Local identity one self edited')
      await page.getByLabel('Provider change reason', { exact: true }).fill('Save the provider used by this administrator session')
      await expectHTTPFailure(page, 'GET', '/api/v1/me/capabilities?project=', 401, () => page.getByRole('button', { name: 'Save provider', exact: true }).click())
      await expect(page.getByRole('link', { name: 'Continue with Local identity one self edited', exact: true })).toBeVisible()
      await expect(editor()).toHaveCount(0); await expect(stored()).toHaveCount(0)
    } finally {
      await page.context().addCookies(original)
      await rootCall('DELETE', `/api/v1/role-assignments/user/${firstUser}/builtin.teststudio.installation-admin`, { reason: 'Remove temporary self-edit administration' }, assignment.revision)
    }
  })

  await test.step('Clear private provider edits on revoked authority and preserve authorized read-only inspection', async () => {
    const policy = { id: 'provider-review-administrator', name: 'Provider review administrator', projectId: '', disabled: false, rules: ['providers.read', 'providers.manage', 'groups.read', 'groups.manage', 'members.manage'].map(permission => ({ permission: `teststudio.${permission}`, effect: 'allow' })), inherits: [], reason: 'Verify actual provider administration revocation' }
    await rootCall('POST', '/api/v1/roles', policy)
    await rootCall('POST', '/api/v1/permissions/grants', { principalKind: 'user', principalId: fixture.readerId, permission: 'teststudio.providers.read', effect: 'allow', reason: 'Retain provider inspection after management withdrawal' })
    const assignment = await rootCall('POST', '/api/v1/role-assignments', { principalKind: 'user', principalId: fixture.readerId, roleId: policy.id, reason: 'Temporarily permit provider administration' })
    try {
      await page.context().addCookies([{ name: 'teststudio_session', value: fixture.readerCookie, domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax', secure: false }])
      await page.goto(route)
      await page.getByRole('button', { name: 'Edit provider', exact: true }).click()
      await page.getByLabel('Provider name', { exact: true }).fill('Private provider change after revocation')
      await page.getByLabel('Provider change reason', { exact: true }).fill('Review management withdrawal')
      await rootCall('DELETE', `/api/v1/role-assignments/user/${fixture.readerId}/${policy.id}`, { reason: 'Withdraw provider management during editing' }, assignment.revision)
      await expectHTTPFailure(page, 'PUT', base+'/local-one', 403, () => page.getByRole('button', { name: 'Save provider', exact: true }).click())
      await expect(editor()).toHaveCount(0); await expect(stored()).toHaveCount(0)
      await page.getByRole('button', { name: 'Refresh provider', exact: true }).click()
      await expect(stored()).toContainText('Read only')
      await expect(page.getByRole('button', { name: 'Edit provider', exact: true })).toHaveCount(0)
      await expect(page.getByRole('region', { name: 'Create a provider group', exact: true })).toHaveCount(0)
    } finally { await page.context().addCookies(original) }
  })
})
