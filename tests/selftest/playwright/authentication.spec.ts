import { test, expect } from './fixtures'

test('OIDC key rotation, group refresh, provider logout, and callback feedback', { tag: '@STUDIO-AUTH-001' }, async ({ page, api }, testInfo) => {
  const { project, path } = await api.scaffold(), route = `/projects/${project}/catalog?focus=preserved`, providerPath = '/api/v1/identity-providers/auth-one'
  const original = await page.context().cookies('http://127.0.0.1:8040'), root = original.find(cookie => cookie.name === 'teststudio_session')!, rootCSRF = (await (await page.request.get('/auth/session')).json()).session.csrfToken
  const rootCall = async (method: string, resource: string, body?: unknown, revision?: number) => { const response = await page.request.fetch(resource, { method, data: body, headers: { Cookie: `${root.name}=${root.value}`, Origin: 'http://127.0.0.1:8040', 'X-CSRF-Token': rootCSRF, ...(revision === undefined ? {} : { 'If-Match': `"${revision}"` }) } }); expect(response.ok(), `${resource}: ${await response.text()}`).toBe(true); return response.json() }
  await rootCall('POST', '/api/v1/identity-providers', { id: 'auth-one', name: 'Authentication one', issuer: 'http://127.0.0.1:8050/oidc/one', clientId: 'teststudio-public', disabled: false, autoProvision: true, scopes: ['openid', 'profile', 'email'], groupMappings: {}, membershipTtlSeconds: 1800, reason: 'Verify independent browser authentication' })
  await rootCall('POST', '/api/v1/groups', { id: 'auth-viewers', name: 'Authentication viewers', projectId: project, managedBy: 'oidc:auth-one' })
  await rootCall('POST', `${path}/role-assignments`, { principalKind: 'group', principalId: 'auth-viewers', roleId: 'builtin.teststudio.viewer', reason: 'Bind verified group membership to one project' })
  const control = async (body: object) => { expect((await page.request.post('http://127.0.0.1:8050/oidc/control', { data: { provider: 'one', ...body } })).ok()).toBe(true) }
  const approve = async (mode = 'complete') => { await page.getByLabel('Fixture group response', { exact: true }).selectOption(mode); await page.getByRole('button', { name: 'Confirm fixture sign-in', exact: true }).click() }
  const begin = async (destination = '/projects') => { await page.goto(destination); await page.getByRole('link', { name: 'Continue with Authentication one', exact: true }).click(); await expect(page.getByRole('heading', { name: 'Local provider one', exact: true })).toBeVisible() }
  const signedOut = async (oldCookie: string) => { const response = await page.request.get('/auth/session', { headers: { Cookie: `teststudio_session=${oldCookie}` } }); expect((await response.json()).user).toBeNull() }
  const cookie = async () => (await page.context().cookies('http://127.0.0.1:8040')).find(item => item.name === 'teststudio_session')!.value

  await test.step('Configure exact provider logout and group mappings through the settings editor', async () => {
    await page.goto('/settings/providers/auth-one'); await page.getByRole('button', { name: 'Edit provider', exact: true }).click()
    await page.getByLabel('Provider logout endpoint (optional)', { exact: true }).fill('http://127.0.0.1:8050/oidc/one/end-session')
    await page.getByLabel('Group claim name (optional)', { exact: true }).fill('groups')
    await page.getByRole('button', { name: '+ Add group mapping', exact: true }).click()
    await page.getByLabel('External group value 1', { exact: true }).fill('reviewers'); await page.getByLabel('Destination group identifier 1', { exact: true }).fill('auth-viewers')
    await page.getByLabel('Provider change reason', { exact: true }).fill('Enable explicit provider sign-out and project group access')
    await page.getByRole('button', { name: 'Save provider', exact: true }).click()
    await expect(page.getByRole('region', { name: 'Stored provider', exact: true })).toContainText('/auth/logout/callback/auth-one')
    expect((await rootCall('GET', providerPath)).data.logoutEndpoint).toBe('http://127.0.0.1:8050/oidc/one/end-session')
  })

  await test.step('Preserve a deep link, rotate signing keys, and replace the earlier browser session', async () => {
    await page.waitForLoadState('networkidle'); await page.context().clearCookies(); await begin(route); await approve()
    await expect(page.getByRole('heading', { name: 'Test catalog', exact: true })).toBeVisible()
    expect(new URL(page.url()).pathname + new URL(page.url()).search).toBe(route)
    const previous = await cookie()
    await control({ rotate: true })
    await page.goto('/auth/login/auth-one?returnTo=' + encodeURIComponent(route)); await approve()
    await expect(page.getByRole('heading', { name: 'Test catalog', exact: true })).toBeVisible()
    expect(await cookie()).not.toBe(previous); await signedOut(previous)
    const storage = await page.evaluate(() => Object.entries(localStorage))
    expect(storage.some(([key, value]) => /id_token|access_token|refresh_token/i.test(key) || /^eyJ/.test(value))).toBe(false)
  })

  await test.step('Close TestStudio first, then explicitly confirm provider logout in both themes', async () => {
    const previous = await cookie()
    await page.getByRole('button', { name: 'Sign out', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Signed out of TestStudio', exact: true })).toBeVisible(); await signedOut(previous)
    await page.getByRole('button', { name: 'Help', exact: true }).click(); const help = page.getByRole('dialog', { name: 'Sign-in help', exact: true }); await expect(help).toContainText('Provider sign-out'); await page.keyboard.press('Escape')
    for (const theme of ['light', 'dark']) { await page.getByLabel('Color theme').selectOption(theme); await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 }); expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true); await testInfo.attach(`provider-logout-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' }) }
    await page.setViewportSize({ width: 1280, height: 900 })
    await page.getByRole('link', { name: 'Sign out of Authentication one', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Local provider one sign-out', exact: true })).toBeVisible()
    expect(new URL(page.url()).searchParams.has('id_token_hint')).toBe(false)
    await page.getByRole('button', { name: 'Confirm provider sign-out', exact: true }).focus(); await page.keyboard.press('Enter')
    await expect(page.getByRole('status').filter({ hasText: 'Provider sign-out completed.' })).toBeVisible()
    expect((await page.context().cookies('http://127.0.0.1:8050/oidc/one')).some(item => item.name === 'sso_fixture_one')).toBe(false)
    await signedOut(previous)
  })

  await test.step('Keep the local session closed when the user cancels provider logout', async () => {
    await begin(); await approve(); await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
    const previous = await cookie(); await page.getByRole('button', { name: 'Sign out', exact: true }).click()
    await page.getByRole('link', { name: 'Sign out of Authentication one', exact: true }).click()
    await page.getByRole('button', { name: 'Cancel provider sign-out', exact: true }).click()
    await expect(page.getByRole('status').filter({ hasText: 'Provider sign-out could not be confirmed.' })).toBeVisible(); await signedOut(previous)
    expect((await page.context().cookies('http://127.0.0.1:8050/oidc/one')).some(item => item.name === 'sso_fixture_one')).toBe(true)
  })

  await test.step('Keep the local session closed when the optional provider becomes unavailable', async () => {
    await begin(); await approve(); await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
    const previous = await cookie(); await page.getByRole('button', { name: 'Sign out', exact: true }).click()
    await expect(page.getByRole('link', { name: 'Sign out of Authentication one', exact: true })).toBeVisible()
    await control({ discoveryFailure: true })
    await page.getByRole('link', { name: 'Sign out of Authentication one', exact: true }).click()
    await expect(page.getByRole('status').filter({ hasText: 'Provider sign-out could not be confirmed.' })).toBeVisible(); await signedOut(previous)
    await control({ discoveryFailure: false })
  })

  await test.step('Show cancellation and expiry, then withdraw stale group access after an incomplete verified claim', async () => {
    await begin(); await page.getByRole('button', { name: 'Cancel fixture sign-in', exact: true }).click()
    await expect(page.getByRole('status').filter({ hasText: 'Sign-in was cancelled or denied.' })).toBeVisible(); await expect(page.locator('body')).not.toContainText('private fixture diagnostic')
    await page.goto('/auth/callback/auth-one?code=old&state=old'); await expect(page.getByRole('alert')).toContainText('sign-in request expired')
    await begin(); await approve(); await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
    const previous = await cookie()
    await page.waitForLoadState('networkidle'); await page.context().clearCookies(); await begin(); await approve('distributed')
    await expect(page.getByRole('alert')).toContainText('Sign-in could not be verified')
    const oldSession = await page.request.get('/auth/session', { headers: { Cookie: `teststudio_session=${previous}` } }); expect((await oldSession.json()).user).not.toBeNull()
    expect((await page.request.get(path, { headers: { Cookie: `teststudio_session=${previous}` } })).status()).toBe(403)
  })
})
