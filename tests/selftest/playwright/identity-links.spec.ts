import { test, expect, expectHTTPFailure } from './fixtures'

const temporary = 'calm lantern birch winter 38', password = 'quiet valley cedar summer 52'

test('Fresh account proof, explicit identity linking, and session rotation', { tag: '@STUDIO-LINKS-001' }, async ({ page, api }, testInfo) => {
  const { path } = await api.scaffold(), user = 'link-owner', base = '/api/v1/me/identity-links'
  for (const [id, name, issuer] of [['link-one', 'Link identity one', 'one'], ['link-two', 'Link identity two', 'two']]) {
    await api.call('POST', '/api/v1/identity-providers', { id, name, issuer: `http://127.0.0.1:8050/oidc/${issuer}`, clientId: 'teststudio-public', scopes: ['openid', 'profile', 'email'], disabled: false, autoProvision: true, groupMappings: {}, membershipTtlSeconds: 0, reason: 'Verify explicit account linking locally' })
  }
  await api.call('POST', '/api/v1/users', { id: user, name: 'Link owner' })
  await api.call('POST', `${path}/role-assignments`, { principalKind: 'user', principalId: user, roleId: 'builtin.teststudio.viewer', reason: 'Keep project access independent of sign-in method' })
  await api.call('PUT', `/api/v1/users/${user}/local-credential`, { password: temporary, reason: 'Provision a new local self-test account' }, 0)
  const region = () => page.getByRole('region', { name: 'Link another sign-in', exact: true })
  const identities = () => page.getByRole('table', { name: 'Your external identities', exact: true })
  const begin = async (provider: string, source: 'password' | 'oidc' = 'password') => {
    await page.getByLabel('New identity provider', { exact: true }).selectOption(provider)
    await page.getByLabel('Identity link reason', { exact: true }).fill('Add a verified independent sign-in')
    if (source === 'password') await page.getByLabel('Current account password', { exact: true }).fill(password)
    await page.getByRole('button', { name: 'Start identity link', exact: true }).click()
    await expect(page.getByRole('link', { name: source === 'password' ? 'Verify new identity' : 'Verify current account', exact: true })).toBeVisible()
  }
  const prove = async (source: boolean, subject: string) => {
    await page.getByRole('link', { name: source ? 'Verify current account' : 'Verify new identity', exact: true }).click()
    expect(new URL(page.url()).searchParams.get('max_age')).toBe('0')
    expect(new URL(page.url()).searchParams.get('prompt')).toBe('login')
    await page.getByLabel('Fixture subject', { exact: true }).fill(subject)
    await page.getByRole('button', { name: 'Confirm fixture sign-in', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Your account', exact: true })).toBeVisible()
  }
  const session = async () => (await (await page.request.get('/auth/session')).json()).session
  const operation = () => new URL(page.url()).searchParams.get('identityLink')!

  await test.step('Require a fresh password and keep a proved identity unlinked until confirmation', async () => {
    await page.context().clearCookies(); await page.goto('/projects')
    await page.getByLabel('User identifier', { exact: true }).fill(user)
    await page.getByLabel('Password', { exact: true }).fill(temporary)
    await page.getByRole('button', { name: 'Sign in with password', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Change your password', exact: true })).toBeVisible()
    await page.getByLabel('Current password', { exact: true }).fill(temporary)
    await page.getByLabel('New password', { exact: true }).fill(password)
    await page.getByLabel('Confirm new password', { exact: true }).fill(password)
    await page.getByRole('button', { name: 'Change password', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
    await page.goto('/account')
    await page.getByLabel('New identity provider', { exact: true }).selectOption('link-one')
    await page.getByLabel('Identity link reason', { exact: true }).fill('Reject an incorrect password')
    await page.getByLabel('Current account password', { exact: true }).fill('wrong')
    await expectHTTPFailure(page, 'POST', base, 422, () => page.getByRole('button', { name: 'Start identity link', exact: true }).click())
    await expect(region().getByRole('alert')).toContainText('Current password is incorrect')
    await expect(page.getByLabel('Current account password', { exact: true })).toHaveValue('')
    await begin('link-one'); await prove(false, 'owner-one')
    await expect(page.getByRole('heading', { name: 'Review the identity to link', exact: true })).toBeVisible()
    await expect(region()).toContainText('owner-one')
    await expect(identities()).not.toContainText('owner-one')
    await page.getByRole('button', { name: 'Cancel identity link', exact: true }).click()
    await expect(region()).toContainText('No identity was added')
    await page.getByRole('button', { name: 'Start a new identity link', exact: true }).click()
    await begin('link-one'); await prove(false, 'owner-one')
  })

  await test.step('Review exact proof in help and both themes, then recover an unconfirmed successful mutation', async () => {
    await page.getByRole('button', { name: 'Help', exact: true }).click()
    const help = page.getByRole('dialog', { name: 'Your account help', exact: true })
    for (const name of ['Introduction', 'Concepts', 'Tutorials']) await expect(help.getByRole('heading', { name, exact: true })).toBeVisible()
    await expect(help).toContainText('Identity linking')
    await page.keyboard.press('Escape')
    for (const theme of ['light', 'dark']) {
      await page.getByLabel('Color theme').selectOption(theme)
      await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await testInfo.attach(`identity-link-review-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' })
    }
    await page.setViewportSize({ width: 1280, height: 900 })
    const prior = await session(), oldCookie = (await page.context().cookies('http://127.0.0.1:8040')).find(cookie => cookie.name === 'teststudio_session')!
    let confirmations = 0
    const confirmPath = `${base}/${operation()}/confirm`
    await page.route(`**${confirmPath}`, async route => { confirmations++; const response = await route.fetch(); expect(response.ok()).toBe(true); await route.fulfill({ response, contentType: 'application/json', body: 'unreadable successful confirmation' }) })
    try {
      await page.getByRole('button', { name: 'Confirm identity link', exact: true }).click()
      await expect(region().getByRole('alert')).toContainText('unreadable response')
      await expect(page.getByRole('button', { name: 'Confirm identity link', exact: true })).toBeDisabled()
      expect(confirmations).toBe(1)
    } finally { await page.unroute(`**${confirmPath}`) }
    await page.getByRole('button', { name: 'Review current link state', exact: true }).click()
    await expect(region().getByRole('status').filter({ hasText: 'Identity linked.' })).toBeVisible()
    await expect(region().getByText('Checking identity link…', { exact: true })).toHaveCount(0)
    await expect(identities()).toContainText('owner-one')
    const next = await session()
    expect(next.csrfToken).not.toBe(prior.csrfToken)
    expect(next.expiresAt).toBe(prior.expiresAt)
    expect(next.authenticationUntil).toBe(prior.authenticationUntil)
    const old = await page.request.get('/auth/session', { headers: { Cookie: `${oldCookie.name}=${oldCookie.value}` } })
    expect((await old.json()).user).toBeNull()
    expect((await (await page.request.get('/api/v1/me')).json()).user.id).toBe(user)
  })

  await test.step('Use the linked provider and require proof of the same current OIDC account', async () => {
    await page.context().clearCookies(); await page.goto('/projects')
    await page.getByRole('link', { name: 'Continue with Link identity one', exact: true }).click()
    await page.getByLabel('Fixture subject', { exact: true }).fill('owner-one')
    await page.getByRole('button', { name: 'Confirm fixture sign-in', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
    expect((await (await page.request.get('/auth/session')).json()).user.id).toBe(user)
    await page.goto('/account')
    await begin('link-two', 'oidc'); await prove(true, 'wrong-source-account')
    await expect(region()).toContainText('Identity proof was rejected')
    await page.getByRole('button', { name: 'Start a new identity link', exact: true }).click()
    await begin('link-two', 'oidc'); await prove(true, 'owner-one')
    await expect(page.getByRole('link', { name: 'Verify new identity', exact: true })).toBeVisible()
    await prove(false, 'owner-two')
    await expect(region()).toContainText('owner-two')
    await expect(identities()).not.toContainText('owner-two')
    await page.getByRole('button', { name: 'Confirm identity link', exact: true }).focus(); await page.keyboard.press('Enter')
    await expect(region().getByRole('status').filter({ hasText: 'Identity linked.' })).toBeVisible()
    await expect(region().getByText('Checking identity link…', { exact: true })).toHaveCount(0)
    await expect(identities()).toContainText('owner-one')
    await expect(identities()).toContainText('owner-two')
    expect((await (await page.request.get('/api/v1/me')).json()).user.id).toBe(user)
    expect((await page.request.get(path)).ok()).toBe(true)
    await expectHTTPFailure(page, 'GET', base+'/unavailable', 404, () => page.goto('/account?identityLink=unavailable'))
    await expect(region().getByRole('alert')).toContainText('record not found')
    await page.getByRole('button', { name: 'Clear unavailable link view', exact: true }).click()
    await expect(page.getByLabel('New identity provider', { exact: true })).toBeVisible()
    expect(new URL(page.url()).searchParams.has('identityLink')).toBe(false)
  })
})
