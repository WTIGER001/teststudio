import { test, expect } from './fixtures'

test('Show configured local and SSO choices and bypass only for SSO-only installations', { tag: '@STUDIO-SIGNIN-001' }, async ({ page, api }, testInfo) => {
  const returnTo = '/account?from=signin-check#profile'
  const definition = (id: string, name: string, hidden = false) => ({ id, name, issuer: 'http://127.0.0.1:8050/oidc/one', clientId: 'teststudio-public', disabled: false, autoProvision: true, scopes: ['openid', 'profile', 'email'], groupMappings: {}, membershipTtlSeconds: 1800, hiddenFromSignIn: hidden, loginBypassEnabled: false, reason: 'Verify generic sign-in choices and optional SSO bypass' })
  await api.call('POST', '/api/v1/identity-providers', definition('organization', 'Organization sign-in'))
  await api.call('POST', '/api/v1/identity-providers', definition('bootstrap', 'TestStudio sign-in', true))
  const mode = async (development: boolean, enabled: boolean) => {
    await page.waitForLoadState('networkidle')
    expect((await page.request.post('http://127.0.0.1:8050/local/mode', { data: { development, enabled } })).ok()).toBe(true)
    await page.context().clearCookies()
    await page.goto('about:blank')
    await page.goto(returnTo)
  }
  const choices = async (password: boolean, sso: boolean) => {
    await expect(page.getByLabel('User identifier', { exact: true })).toHaveCount(password ? 1 : 0)
    await expect(page.getByLabel('Password', { exact: true })).toHaveCount(password ? 1 : 0)
    await expect(page.getByRole('link', { name: 'Continue with Organization sign-in', exact: true })).toHaveCount(sso ? 1 : 0)
    await expect(page.getByRole('link', { name: 'Continue with TestStudio sign-in', exact: true })).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Open local workspace', exact: true })).toHaveCount(0)
  }
  const approve = async () => { await page.getByLabel('Fixture group response', { exact: true }).selectOption('complete'); await page.getByRole('button', { name: 'Confirm fixture sign-in', exact: true }).click() }
  try {
    await test.step('Keep one SSO choice with bypass off and local login disabled', async () => {
      await mode(false, false)
      await expect(page.getByRole('heading', { name: 'A clear view of what works.', exact: true })).toBeVisible()
      await choices(false, true)
    })
    await test.step('Enable bypass through the normal provider editor', async () => {
      await mode(true, false)
      expect((await page.request.post('/auth/local', { data: {}, headers: { Origin: 'http://127.0.0.1:8040' } })).ok()).toBe(true)
      await page.goto('/settings/providers/organization')
      await page.getByRole('button', { name: 'Edit provider', exact: true }).click()
      await page.getByLabel('Automatically open this SSO provider', { exact: true }).check()
      await page.getByLabel('Provider change reason', { exact: true }).fill('Enable reviewed organization SSO bypass')
      await page.getByRole('button', { name: 'Save provider', exact: true }).click()
      await expect(page.getByRole('region', { name: 'Stored provider', exact: true })).toContainText('Enabled when local sign-in is off')
    })
    await test.step('Keep password and SSO choices when password login is enabled', async () => {
      await mode(false, true)
      await choices(true, true)
      for (const [theme, width] of [['light', 1280], ['dark', 390]] as const) {
        await page.getByLabel('Color theme', { exact: true }).selectOption(theme)
        await page.setViewportSize({ width, height: 900 })
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
        await page.getByLabel('User identifier', { exact: true }).fill('unsaved-local-name')
        await page.getByRole('button', { name: 'Help', exact: true }).click()
        await expect(page.getByRole('dialog', { name: 'Sign-in help', exact: true })).toContainText('SSO')
        await page.keyboard.press('Escape')
        await expect(page.getByLabel('User identifier', { exact: true })).toHaveValue('unsaved-local-name')
        await testInfo.attach(`signin-${theme}-${width}`, { body: await page.screenshot({ fullPage: true }), contentType: 'image/png' })
      }
      await page.setViewportSize({ width: 1280, height: 900 })
    })
    await test.step('Automatically open the configured SSO provider without rendering a login form', async () => {
      await mode(false, false)
      await expect(page.getByRole('heading', { name: 'Local provider one', exact: true })).toBeVisible()
      await expect(page.getByRole('heading', { name: 'A clear view of what works.', exact: true })).toHaveCount(0)
      await page.getByRole('button', { name: 'Cancel fixture sign-in', exact: true }).click()
      await expect(page.getByRole('status').filter({ hasText: 'Sign-in was cancelled or denied.' })).toBeVisible()
      await choices(false, true)
      await page.getByRole('button', { name: 'Dismiss sign-in message', exact: true }).click()
      await choices(false, true)
      await page.getByRole('link', { name: 'Continue with Organization sign-in', exact: true }).click()
      await approve()
      await expect(page.getByRole('link', { name: 'Your account', exact: true })).toBeVisible()
      expect(new URL(page.url()).pathname + new URL(page.url()).search + new URL(page.url()).hash).toBe(returnTo)
      const signedIn = await (await page.request.get('/auth/session')).json()
      expect(signedIn.user.id).not.toBe('local-administrator')
      expect((await (await page.request.get('/api/v1/projects')).json()).items).toEqual([])
      await page.getByRole('button', { name: 'Sign out', exact: true }).click()
      await expect(page.getByRole('heading', { name: 'Signed out of TestStudio', exact: true })).toBeVisible()
      let delayedChunks = 0
      await page.route('**/chunk-*.js', async route => { delayedChunks++; const response = await route.fetch(); await new Promise(resolve => setTimeout(resolve, 750)); await route.fulfill({ response }) })
      await page.reload()
      await page.waitForLoadState('networkidle')
      await choices(false, true)
      expect(delayedChunks).toBeGreaterThan(0)
      await page.unroute('**/chunk-*.js')
    })
    await test.step('Keep expired sign-in recovery visible instead of redirecting repeatedly', async () => {
      await page.goto('/auth/callback/organization?code=expired&state=expired')
      await expect(page.getByRole('alert')).toContainText('sign-in request expired')
      await choices(false, true)
      await page.getByRole('button', { name: 'Dismiss sign-in message', exact: true }).click()
      await choices(false, true)
    })
    await test.step('Hide SSO when disabled and retain only the configured password form', async () => {
      await mode(true, true)
      expect((await page.request.post('/auth/local', { data: {}, headers: { Origin: 'http://127.0.0.1:8040' } })).ok()).toBe(true)
      const session = await (await page.request.get('/auth/session')).json()
      const row = await (await page.request.get('/api/v1/identity-providers/organization')).json()
      const response = await page.request.put('/api/v1/identity-providers/organization', { data: { ...definition('organization', 'Organization sign-in'), disabled: true }, headers: { Origin: 'http://127.0.0.1:8040', 'X-CSRF-Token': session.session.csrfToken, 'If-Match': `"${row.revision}"` } })
      expect(response.ok()).toBe(true)
      await mode(false, true)
      await choices(true, false)
      await mode(false, false)
      await expect(page.getByText('No sign-in providers are configured for this installation.', { exact: true })).toBeVisible()
      await choices(false, false)
    })
  } finally { await mode(true, false) }
})
