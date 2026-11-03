import { test, expect, followNavigation } from './fixtures'

test('Durable schedules, editable configuration, and retained firing history', { tag: '@STUDIO-SCHEDULE-001' }, async ({ page, api }, testInfo) => {
  const { project, path } = await api.scaffold()
  await api.createCheck(path)
  const other = await api.scaffold()
  const resource = `${path}/schedules/availability-probe`, url = `/projects/${project}/schedules`
  const row = () => page.getByRole('row').filter({ hasText: 'availability-probe' })
  let firstRun = ''

  await test.step('Preview a selection without creating a run, then enable it', async () => {
    await page.goto(url)
    await page.getByRole('button', { name: '+ New schedule' }).click()
    await page.getByLabel('Schedule name', { exact: true }).fill('Availability probe')
    await page.getByLabel('Schedule identifier', { exact: true }).fill('availability-probe')
    await page.getByLabel('Suite', { exact: true }).selectOption('health')
    await page.getByLabel('Target environment', { exact: true }).selectOption('local')
    await page.getByLabel('Interval (seconds)', { exact: true }).fill('5')
    await page.getByLabel('Run time limit (seconds)', { exact: true }).fill('30')
    await page.getByLabel('Enable this schedule', { exact: true }).check()
    await page.getByRole('button', { name: 'Preview schedule', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Schedule preview', exact: true })).toBeVisible()
    await expect(page.getByText(/1 selected test/)).toBeVisible()
    expect((await api.call('GET', `${path}/runs`)).items).toHaveLength(0)
    await page.getByRole('button', { name: 'Save schedule', exact: true }).click()
    await expect(page.getByRole('status').filter({ hasText: /^Schedule configuration saved/ })).toBeVisible()
    await expect(row()).toContainText('Enabled')
    await expect(row()).toContainText('Every 5 seconds')
  })

  await test.step('Keep an unsaved edit across a firing and save with its original configuration revision', async () => {
    const before = await api.call('GET', resource)
    await row().getByRole('button', { name: 'Edit', exact: true }).click()
    await page.getByLabel('Schedule name', { exact: true }).fill('Renamed availability probe')
    await expect.poll(async () => { firstRun = (await api.call('GET', resource)).data.lastRunId; return !!firstRun }, { timeout: 20000 }).toBe(true)
    expect((await api.call('GET', resource)).revision).toBe(before.revision)
    await expect(page.getByLabel('Schedule name', { exact: true })).toHaveValue('Renamed availability probe')
    await page.getByRole('button', { name: 'Save schedule', exact: true }).click()
    await expect(row()).toContainText('Renamed availability probe')
    await row().getByRole('button', { name: 'Pause', exact: true }).click()
    await expect(row()).toContainText('Paused')
    const paused = (await api.call('GET', resource)).data
    expect(paused.nextAt).toBeUndefined()
    const until = Date.now() + 6000
    while (Date.now() < until) {
      expect((await api.call('GET', resource)).data.lastRunId).toBe(paused.lastRunId)
      await page.waitForTimeout(250)
    }
  })

  await test.step('Follow scheduled runs through normal timing, result, and evidence views', async () => {
    await row().getByRole('button', { name: 'History', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Firings: Renamed availability probe' })).toBeVisible()
    const firings = (await api.call('GET', `${resource}/firings`)).items
    expect(new Set(firings.map((item: any) => item.data.dueAt)).size).toBe(firings.length)
    await page.getByRole('link', { name: firstRun, exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Run details', exact: true })).toBeVisible()
    await expect(page.getByRole('region', { name: 'Execution snapshot', exact: true })).toContainText('availability-probe')
    await expect.poll(async () => (await api.call('GET', `${path}/runs/${firstRun}`)).data.state).toBe('completed')
    const run = (await api.call('GET', `${path}/runs/${firstRun}`)).data
    expect(run.trigger).toBe('schedule'); expect(run.counts.passed).toBe(1); expect(run.evidenceState).toBe('complete')
    expect(run.results[0].timing).toBeTruthy()
    expect((await api.call('GET', `${path}/runs/${firstRun}/artifacts`)).items.length).toBeGreaterThan(0)
    await page.goto(url)
  })

  await test.step('Expose blocked selections and retain their history through archive and restore', async () => {
    await row().getByRole('button', { name: 'Enable', exact: true }).click()
    await expect(row()).toContainText('Enabled')
    await api.call('POST', `${path}/suites/health/tests/SELF-CHECK/actions/disable`, {}, 1)
    await expect.poll(async () => (await api.call('GET', resource)).data.lastState, { timeout: 15000 }).toBe('blocked')
    await page.getByRole('button', { name: 'Refresh schedules', exact: true }).click()
    await expect(row()).toContainText('currently disabled')
    const before = (await api.call('GET', `${resource}/firings`)).items.length
    await row().getByRole('button', { name: 'Archive', exact: true }).click()
    await expect(row()).toHaveCount(0)
    await page.getByLabel('Include archived schedules', { exact: true }).check()
    await expect(row()).toContainText('Archived')
    await row().getByRole('button', { name: 'Restore', exact: true }).click()
    await expect(row()).toContainText('Paused')
    expect((await api.call('GET', resource)).data.enabled).toBe(false)
    expect((await api.call('GET', `${resource}/firings`)).items.length).toBe(before)
    await api.call('POST', `${path}/suites/health/tests/SELF-CHECK/actions/enable`, {}, 2)
  })

  await test.step('Preview cron times, preserve edits through help, and use both themes', async () => {
    await row().getByRole('button', { name: 'Edit', exact: true }).click()
    await page.getByLabel('Cadence', { exact: true }).selectOption('cron')
    await page.getByLabel('Cron expression', { exact: true }).fill('0 2 * * *')
    await page.getByLabel('Timezone', { exact: true }).fill('America/New_York')
    await page.getByRole('button', { name: 'Preview schedule', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Schedule preview', exact: true })).toBeVisible()
    await page.getByRole('button', { name: 'Help', exact: true }).click()
    const help = page.getByRole('dialog', { name: 'Schedules help', exact: true })
    await expect(help).toContainText('Overlap and missed ticks')
    await page.keyboard.press('Escape')
    await expect(page.getByLabel('Timezone', { exact: true })).toHaveValue('America/New_York')
    for (const theme of ['light', 'dark']) {
      await page.getByLabel('Color theme').selectOption(theme)
      await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await testInfo.attach(`schedule-editor-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' })
    }
    await page.getByRole('button', { name: 'Save schedule', exact: true }).click()
    await expect(row()).toContainText('0 2 * * * · America/New_York')
    await expect(row()).toContainText('Paused')
  })

  await test.step('Discard the old project editor when changing workspace', async () => {
    await row().getByRole('button', { name: 'Edit', exact: true }).click()
    await page.getByLabel('Schedule name', { exact: true }).fill('Unsaved project change')
    await page.getByLabel('Workspace', { exact: true }).selectOption(other.project)
    await followNavigation(page, 'Schedules')
    await expect(page.getByRole('heading', { name: 'Edit schedule', exact: true })).toHaveCount(0)
    expect((await api.call('GET', resource)).data.name).toBe('Renamed availability probe')
  })
})
