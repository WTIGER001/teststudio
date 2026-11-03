import { request } from '@playwright/test'
import { API, test, expect, expectHTTPFailure, followNavigation } from './fixtures'

const rootTemporary = 'calm river granite sunrise 46'
const rootPassword = 'bright canyon cedar evening 72'
const temporary = 'silver meadow paper lantern 29'
const personal = 'violet ocean winter lantern 53'
const personalNext = 'copper forest summer lantern 84'
const personalRecovered = 'gentle glacier maple lantern 91'
const reset = 'golden mountain cloud lantern 68'
const finalPassword = 'quiet valley glass lantern 17'

test('Local accounts, required password changes, and credential administration', { tag: '@STUDIO-LOCAL-001' }, async ({ page, api }, testInfo) => {
  const { project, path } = await api.scaffold()
  const user = 'local-reviewer', credentialPath = `/api/v1/users/${user}/local-credential`
  const region = () => page.getByRole('region', { name: 'Local password credential', exact: true })
  const signIn = async (id: string, password: string) => {
    await page.getByLabel('User identifier', { exact: true }).fill(id)
    await page.getByLabel('Password', { exact: true }).fill(password)
    await page.getByRole('button', { name: 'Sign in with password', exact: true }).click()
  }
  const change = async (old: string, next: string) => {
    await page.getByLabel('Current password', { exact: true }).fill(old)
    await page.getByLabel('New password', { exact: true }).fill(next)
    await page.getByLabel('Confirm new password', { exact: true }).fill(next)
    await page.getByRole('button', { name: 'Change password', exact: true }).click()
  }
  const provision = async (password: string, reason: string) => {
    await page.getByLabel('Temporary local password', { exact: true }).fill(password)
    await page.getByLabel('Confirm temporary password', { exact: true }).fill(password)
    await page.getByLabel('Credential change reason', { exact: true }).fill(reason)
  }
  const mode = async (enabled: boolean) => {
    await page.waitForLoadState('networkidle')
    expect((await page.request.post('http://127.0.0.1:8050/local/mode', { data: { development: false, enabled } })).ok()).toBe(true)
    await page.reload()
  }
  await api.call('POST', '/api/v1/users', { id: user, name: 'Local reviewer' })
  await api.call('POST', `${path}/role-assignments`, { principalKind: 'user', principalId: user, roleId: 'builtin.teststudio.viewer', reason: 'Allow only project inspection' })

  await test.step('Provision managed root access, force a real password change, and remove the development sign-in', async () => {
    await page.goto('/settings/users/local-administrator')
    await region().getByRole('button', { name: 'Set local password', exact: true }).click()
    await provision(rootTemporary, 'Prepare password-only administration')
    await page.getByRole('button', { name: 'Save local password', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Sign in with password', exact: true })).toBeVisible()
    await expect(page.getByRole('link', { name: 'Continue with Self-test identity', exact: true })).toBeVisible()
    await signIn('local-administrator', rootTemporary)
    await expect(page.getByRole('heading', { name: 'Change your password', exact: true })).toBeVisible()
    await change(rootTemporary, rootPassword)
    await expect(page.getByRole('heading', { name: 'User details', exact: true })).toBeVisible()
    await mode(true)
    await expect(page.getByRole('heading', { name: 'User details', exact: true })).toBeVisible()
    await region().getByRole('button', { name: 'Disable local sign-in', exact: true }).click()
    await page.getByLabel('Reason', { exact: true }).fill('Protect the final password administrator')
    await expectHTTPFailure(page, 'POST', '/api/v1/users/local-administrator/local-credential/disable', 422, () => page.getByRole('button', { name: 'Confirm action', exact: true }).click())
    await expect(page.getByRole('alert')).toContainText('last usable installation administrator')
    await page.getByRole('button', { name: 'Cancel action', exact: true }).click()
  })

  const admin = await request.newContext({ baseURL: 'http://127.0.0.1:8040' })
  let adminAPI: API
  try {
    expect((await admin.post('/auth/password', { data: { userId: 'local-administrator', password: rootPassword }, headers: { Origin: 'http://127.0.0.1:8040' } })).ok()).toBe(true)
    adminAPI = new API(admin, (await (await admin.get('/auth/session')).json()).session.csrfToken)
    await test.step('Preserve a reset snapshot through concurrent changes while clearing submitted secrets', async () => {
      await page.goto(`/settings/users/${user}`)
      await region().getByRole('button', { name: 'Set local password', exact: true }).click()
      await provision(temporary, 'Provision a local project reviewer')
      await page.getByRole('button', { name: 'Save local password', exact: true }).click()
      await expect(region().getByRole('status')).toContainText('Local password saved')
      await region().getByRole('button', { name: 'Reset local password', exact: true }).click()
      await provision(temporary, 'Keep this credential review reason')
      const before = await adminAPI.call('GET', credentialPath)
      await adminAPI.call('PUT', credentialPath, { password: reset, reason: 'Concurrent administrator reset' }, before.revision)
      await region().getByRole('button', { name: 'Refresh local credential', exact: true }).click()
      await expect(region().getByText('Loading local credential…', { exact: true })).toHaveCount(0)
      await expectHTTPFailure(page, 'PUT', credentialPath, 409, () => page.getByRole('button', { name: 'Save local password', exact: true }).click())
      await expect(page.getByLabel('Temporary local password', { exact: true })).toHaveValue('')
      await expect(page.getByLabel('Confirm temporary password', { exact: true })).toHaveValue('')
      await expect(page.getByLabel('Credential change reason', { exact: true })).toHaveValue('Keep this credential review reason')
      await expect(page.getByRole('button', { name: 'Save local password', exact: true })).toBeDisabled()
      await page.getByRole('button', { name: 'Close password editor', exact: true }).click()
      await region().getByRole('button', { name: 'Refresh local credential', exact: true }).click()
      await region().getByRole('button', { name: 'Reset local password', exact: true }).click()
      await provision(temporary, 'Reviewed current credential state')
      await page.getByRole('button', { name: 'Help', exact: true }).click()
      await expect(page.getByRole('dialog', { name: 'User details help', exact: true })).toContainText('Local credentials')
      await page.keyboard.press('Escape')
      await expect(page.getByLabel('Credential change reason', { exact: true })).toHaveValue('Reviewed current credential state')
      await page.getByRole('button', { name: 'Save local password', exact: true }).click()
      await expect(page.getByRole('heading', { name: 'Provision or reset local password', exact: true })).toHaveCount(0)
      await page.getByRole('button', { name: 'Sign out', exact: true }).click()
      await expect(page.getByRole('button', { name: 'Open local workspace', exact: true })).toHaveCount(0)
    })

    await test.step('Use managed sign-in and keep required-change sessions away from private page requests', async () => {
      await page.getByLabel('User identifier', { exact: true }).fill(user)
      await page.getByLabel('Password', { exact: true }).fill('incorrect credential')
      await expectHTTPFailure(page, 'POST', '/auth/password', 401, () => page.getByRole('button', { name: 'Sign in with password', exact: true }).click())
      await expect(page.getByLabel('User identifier', { exact: true })).toHaveValue(user)
      await expect(page.getByLabel('Password', { exact: true })).toHaveValue('')
      const privateRequests: string[] = []
      const observe = (req: any) => { if (new URL(req.url()).pathname.startsWith('/api/v1/')) privateRequests.push(req.url()) }
      page.on('request', observe)
      try {
        await signIn(user, temporary)
        await expect(page.getByRole('heading', { name: 'Change your password', exact: true })).toBeVisible()
        await expect(page.getByRole('navigation', { name: 'Main navigation' })).toHaveCount(0)
        expect(privateRequests).toEqual([])
        expect((await page.request.get('/api/v1/projects')).status()).toBe(403)
        await page.getByLabel('Current password', { exact: true }).fill(temporary)
        await page.getByRole('button', { name: 'Help', exact: true }).click()
        await expect(page.getByRole('dialog', { name: 'Required password change help', exact: true })).toContainText('Restricted session')
        await page.keyboard.press('Escape')
        await expect(page.getByLabel('Current password', { exact: true })).toHaveValue(temporary)
        for (const theme of ['light', 'dark']) {
          await page.getByLabel('Color theme').selectOption(theme)
          await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
          expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
          await testInfo.attach(`required-password-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' })
        }
      } finally { page.off('request', observe) }
      await change(temporary, personal)
      await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
      await followNavigation(page, 'Your account')
      await expect(page.getByRole('heading', { name: 'Your account', exact: true })).toBeVisible()
      await expect(page.getByText('Local password sign-in is enabled.', { exact: true })).toBeVisible()
      await expect(page.getByRole('navigation').getByRole('link', { name: 'Users', exact: true, includeHidden: true })).toHaveCount(0)
    })

    await test.step('Change an ordinary password without installation authority and rotate its session', async () => {
      const before = (await page.context().cookies('http://127.0.0.1:8040')).find(cookie => cookie.name === 'teststudio_session')!
      await page.getByLabel('Current password', { exact: true }).fill('incorrect current password')
      await page.getByLabel('New password', { exact: true }).fill(personalNext)
      await page.getByLabel('Confirm new password', { exact: true }).fill(personalNext)
      await expectHTTPFailure(page, 'POST', '/auth/password/change', 422, () => page.getByRole('button', { name: 'Change password', exact: true }).click())
      await expect(page.getByLabel('Current password', { exact: true })).toHaveValue('')
      await expect(page.getByLabel('New password', { exact: true })).toHaveValue('')
      await change(personal, personalNext)
      await expect(page.getByRole('region', { name: 'Change password', exact: true }).getByRole('status')).toContainText('Password changed')
      const after = (await page.context().cookies('http://127.0.0.1:8040')).find(cookie => cookie.name === 'teststudio_session')!
      expect(after.value !== before.value).toBe(true)
      for (const theme of ['light', 'dark']) {
        await page.getByLabel('Color theme').selectOption(theme)
        await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
        await testInfo.attach(`personal-password-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' })
      }
      let passwordWrites = 0
      const unreadableChange = async (route: import('@playwright/test').Route) => {
        passwordWrites++
        const response = await route.fetch()
        expect(response.ok()).toBe(true)
        await route.fulfill({ response, body: '{', contentType: 'application/json' })
      }
      await page.route('**/auth/password/change', unreadableChange)
      try {
        await change(personalNext, personalRecovered)
        await expect(page.getByRole('alert')).toContainText('unreadable response')
        await expect(page.getByLabel('Current password', { exact: true })).toHaveValue('')
        await expect(page.getByLabel('New password', { exact: true })).toHaveValue('')
        await expect(page.getByRole('button', { name: 'Change password', exact: true })).toBeDisabled()
        await page.getByRole('button', { name: 'Refresh session', exact: true }).click()
        await expect(page.getByRole('button', { name: 'Change password', exact: true })).toBeEnabled()
        expect(passwordWrites).toBe(1)
      } finally { await page.unroute('**/auth/password/change', unreadableChange) }
      const values = await page.evaluate(() => JSON.stringify({ local: { ...localStorage }, session: { ...sessionStorage } }))
      expect([temporary, personal, personalNext, personalRecovered].some(secret => values.includes(secret))).toBe(false)
      await page.setViewportSize({ width: 1280, height: 900 })
      await page.getByRole('button', { name: 'Sign out', exact: true }).click()
      await signIn('local-administrator', rootPassword)
      await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
    })

    await test.step('Disable and reset a credential while retaining disabled state and invalidating other sessions', async () => {
      const secondary = await request.newContext({ baseURL: 'http://127.0.0.1:8040' })
      try {
        expect((await secondary.post('/auth/password', { data: { userId: user, password: personalRecovered }, headers: { Origin: 'http://127.0.0.1:8040' } })).ok()).toBe(true)
        await page.goto(`/settings/users/${user}`)
        await region().getByRole('button', { name: 'Disable local sign-in', exact: true }).click()
        await page.getByLabel('Reason', { exact: true }).fill('Suspend local sign-in for review')
        await page.getByRole('button', { name: 'Confirm action', exact: true }).click()
        await expect(region().getByText('Disabled', { exact: true })).toBeVisible()
        expect((await secondary.get('/api/v1/me')).status()).toBe(401)
        await region().getByRole('button', { name: 'Reset local password', exact: true }).click()
        await provision(reset, 'Replace forgotten credentials while disabled')
        await page.getByRole('button', { name: 'Save local password', exact: true }).click()
        await expect(region().getByRole('status')).toContainText('Local password saved')
        await expect(region().getByText('Disabled', { exact: true })).toBeVisible()
        await region().getByRole('button', { name: 'Enable local sign-in', exact: true }).click()
        await page.getByLabel('Reason', { exact: true }).fill('Enable the reviewed account')
        await page.getByRole('button', { name: 'Confirm action', exact: true }).click()
        await expect(region().getByText('Enabled', { exact: true })).toBeVisible()
        expect((await secondary.get('/api/v1/me')).status()).toBe(401)
      } finally { await secondary.dispose() }
      await page.getByRole('button', { name: 'Sign out', exact: true }).click()
      await signIn(user, reset)
      await expect(page.getByRole('heading', { name: 'Change your password', exact: true })).toBeVisible()
      await change(reset, finalPassword)
      await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
      await page.getByRole('button', { name: 'Sign out', exact: true }).click()
      await signIn('local-administrator', rootPassword)
      await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
    })
  } finally { await admin.dispose() }

  await test.step('Switch authentication modes without reviving old cookies and retain OIDC choices', async () => {
    await mode(false)
    await expect(page.getByRole('button', { name: 'Sign in with password', exact: true })).toHaveCount(0)
    await expect(page.getByRole('link', { name: 'Continue with Self-test identity', exact: true })).toBeVisible()
    await mode(true)
    await expect(page.getByRole('button', { name: 'Sign in with password', exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Sign out', exact: true })).toHaveCount(0)
    await signIn('local-administrator', rootPassword)
    await expect(page.getByRole('heading', { name: 'Projects', exact: true })).toBeVisible()
    const original = await page.context().cookies('http://127.0.0.1:8040')
    const prepared = await page.request.post('http://127.0.0.1:8050/access/prepare', { data: { project } })
    expect(prepared.ok()).toBe(true)
    const fixture = await prepared.json()
    try {
      await page.context().addCookies([{ name: 'teststudio_session', value: fixture.cookie, domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax', secure: false }])
      await page.goto('/account')
      await expect(page.getByRole('heading', { name: 'Project manager', exact: true })).toBeVisible()
      await expect(page.getByText('No local password is configured.', { exact: false })).toBeVisible()
      await expect(page.getByRole('table', { name: 'Your external identities', exact: true })).toContainText('https://identity.selftest.invalid')
      await expect(page.getByRole('button', { name: 'Change password', exact: true })).toHaveCount(0)
      await page.goto('/settings/users/local-administrator')
      await expect(page.getByRole('alert')).toContainText('Installation user read permission is required')
      await expect(page.getByRole('heading', { name: 'Local password credential', exact: true })).toHaveCount(0)
      await page.goto('/account')
      await expect(page.getByRole('heading', { name: 'Project manager', exact: true })).toBeVisible()
      const unreadableProfile = async (route: import('@playwright/test').Route) => { await route.fulfill({ status: 401, body: '{', contentType: 'application/json' }) }
      await page.route('**/api/v1/me?limit=30', unreadableProfile)
      try {
        await expectHTTPFailure(page, 'GET', '/api/v1/me?limit=30', 401, () => page.getByRole('button', { name: 'Refresh account', exact: true }).click())
        await expect(page.getByRole('button', { name: 'Sign in with password', exact: true })).toBeVisible()
        await expect(page.getByRole('navigation', { name: 'Main navigation' })).toHaveCount(0)
      } finally { await page.unroute('**/api/v1/me?limit=30', unreadableProfile) }
    } finally { await page.context().addCookies(original); await page.goto('/account') }
  })
})
