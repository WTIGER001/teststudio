import { test, expect, followNavigation } from './fixtures'

test('Notification channels, durable delivery, and retained attempts', { tag: '@STUDIO-NOTIFY-001' }, async ({ page, api }, testInfo) => {
  const { project, path } = await api.scaffold('notification-selftests')
  const other = await api.scaffold()
  await api.createCheck(path)
  const base = `${path}/notifications`, url = `/projects/${project}/notifications`
  const channels = () => page.getByRole('table', { name: 'Project notification channels', exact: true })
  const channelRow = (id: string) => channels().getByRole('row').filter({ has: page.getByText(id, { exact: true }) })
  const deliveries = () => page.getByRole('table', { name: 'Notification delivery history', exact: true })
  const deliveryRow = (id: string) => deliveries().getByRole('row').filter({ has: page.getByText(id, { exact: true }) })
  const readDelivery = async (run: string) => (await api.call('GET', `${base}/deliveries?run=${run}`)).items[0]
  const finished = async (run: string, state: string) => {
    let delivery: any
    await expect.poll(async () => { delivery = await readDelivery(run); return delivery?.data.state }, { timeout: 18000, intervals: [100, 250, 500] }).toBe(state)
    return delivery
  }
  const refresh = async () => { await page.getByRole('button', { name: 'Refresh deliveries', exact: true }).click() }
  const create = async (id: string, reference: string, transport = 'webhook') => {
    await page.getByRole('button', { name: '+ New channel', exact: true }).click()
    await page.getByLabel('Channel name', { exact: true }).fill(`Channel ${id}`)
    await page.getByLabel('Channel identifier', { exact: true }).fill(id)
    await page.getByLabel('Transport', { exact: true }).selectOption(transport)
    await page.getByLabel('Secret reference', { exact: true }).selectOption(reference)
    await page.getByLabel('Suite identifiers (optional)', { exact: true }).fill('health')
    await page.getByLabel('Environment identifiers (optional)', { exact: true }).fill('local')
    await page.getByLabel('Stages (optional)', { exact: true }).fill('local')
    await page.getByLabel('Automatic delivery attempts', { exact: true }).fill('3')
    await page.getByLabel('Passed', { exact: true }).check()
    await page.getByLabel('Enable this channel', { exact: true }).check()
    await page.getByRole('button', { name: 'Save notification channel', exact: true }).click()
    await expect(channelRow(id)).toContainText('Enabled')
  }
  let first: any

  await test.step('Configure a scoped channel using registered secret names and validate trigger selection', async () => {
    await page.goto(url)
    await expect(page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'Notifications', exact: true })).toBeVisible()
    const refs = (await api.call('GET', `${base}/references`)).items
    expect(refs).toHaveLength(3)
    for (const ref of refs) expect(Object.keys(ref).sort()).toEqual(['id', 'name'])
    await create('availability', 'retry-receiver')
    await channelRow('availability').getByRole('button', { name: 'Edit', exact: true }).click()
    for (const label of ['Manual runs', 'Scheduled runs', 'Failure reruns']) await page.getByLabel(label, { exact: true }).uncheck()
    await page.getByRole('button', { name: 'Save notification channel', exact: true }).click()
    await expect(page.getByRole('alert')).toContainText('Select at least one run trigger')
    for (const label of ['Manual runs', 'Scheduled runs', 'Failure reruns']) await page.getByLabel(label, { exact: true }).check()
    await page.getByRole('button', { name: 'Save notification channel', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Edit notification channel', exact: true })).toHaveCount(0)
  })

  await test.step('Deliver a manual completion after a real transient response and retain stable attempt history', async () => {
    const run = await api.run(path)
    first = await finished(run.id, 'delivered')
    expect(first.data.attempts).toBe(2)
    expect(first.data.message.counts).toMatchObject({ passed: 1, failed: 0, timedOut: 0, total: 1 })
    await refresh()
    await expect(deliveryRow(first.id)).toContainText('delivered')
    await deliveryRow(first.id).getByRole('button', { name: 'Attempts', exact: true }).click()
    const attempts = page.getByRole('table', { name: 'Retained delivery attempt history', exact: true })
    await expect(attempts).toContainText('HTTP 503'); await expect(attempts).toContainText('HTTP 204')
    const state = await (await page.request.get('http://127.0.0.1:8050/state')).json()
    const received = state.requests.filter((item: any) => item.deliveryId === first.id)
    expect(received.map((item: any) => item.attempt)).toEqual([1, 2])
    expect(received.every((item: any) => !item.secretSeenInMessage && item.runId === run.id)).toBe(true)
    expect(JSON.stringify(await api.call('GET', `${base}/deliveries/${first.id}/attempts`))).not.toContain('provider diagnostic')
    await deliveryRow(first.id).getByRole('link', { name: run.id, exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Run details', exact: true })).toBeVisible()
    await expect(deliveries().getByRole('row')).toHaveCount(2)
    await expect(deliveryRow(first.id)).toContainText('delivered')
    expect((await api.call('GET', `${path}/runs/${run.id}`)).data.outcome).toBe('passed')
    await page.goto(url)
    await channelRow('availability').getByRole('button', { name: 'Pause', exact: true }).click()
    await expect(channelRow('availability')).toContainText('Paused')
  })

  await test.step('Retry a permanent Discord delivery failure without rerunning its passing test', async () => {
    await create('discord', 'failure-receiver', 'discord')
    const run = await api.run(path), failed = await finished(run.id, 'failed')
    expect(failed.data.httpStatus).toBe(400)
    await refresh(); await expect(deliveryRow(failed.id)).toContainText('HTTP 400')
    expect((await api.call('GET', `${path}/runs/${run.id}`)).data.outcome).toBe('passed')
    expect((await page.request.post('http://127.0.0.1:8050/control', { data: { acceptFailure: true } })).ok()).toBe(true)
    await deliveryRow(failed.id).getByRole('button', { name: 'Retry delivery', exact: true }).click()
    const delivered = await finished(run.id, 'delivered')
    expect(delivered.id).toBe(failed.id); expect(delivered.data.attempts).toBe(2)
    const state = await (await page.request.get('http://127.0.0.1:8050/state')).json()
    const received = state.requests.filter((item: any) => item.deliveryId === failed.id)
    expect(received).toHaveLength(2)
    expect(received.every((item: any) => item.discordConfirmed && item.mentionsSuppressed && !item.secretSeenInMessage)).toBe(true)
    await channelRow('discord').getByRole('button', { name: 'Pause', exact: true }).click()
    await expect(channelRow('discord')).toContainText('Paused')
  })

  await test.step('Stop waiting delivery retries while preserving the original test outcome', async () => {
    await create('waiting', 'waiting-receiver')
    const run = await api.run(path), waiting = await finished(run.id, 'retrying')
    await refresh(); await expect(deliveryRow(waiting.id)).toContainText('retrying')
    await deliveryRow(waiting.id).getByRole('button', { name: 'Stop retries', exact: true }).click()
    await expect(deliveryRow(waiting.id)).toContainText('cancelled')
    expect((await api.call('GET', `${path}/runs/${run.id}`)).data.outcome).toBe('passed')
    expect((await api.call('GET', `${base}/deliveries/${waiting.id}`)).data.attempts).toBe(1)
    await channelRow('waiting').getByRole('button', { name: 'Pause', exact: true }).click()
    await expect(channelRow('waiting')).toContainText('Paused')
  })

  await test.step('Use the same delivery path for an actual scheduled run', async () => {
    await channelRow('availability').getByRole('button', { name: 'Enable', exact: true }).click()
    await expect(channelRow('availability')).toContainText('Enabled')
    const schedule = `${path}/schedules/completion-check`
    await api.call('POST', `${path}/schedules`, { id: 'completion-check', name: 'Completion check', enabled: true, kind: 'interval', intervalSeconds: 5, overlap: 'skip', request: { suiteId: 'health', environmentId: 'local', executionLocation: 'local', timeoutSeconds: 30 } })
    let saved: any
    await expect.poll(async () => { saved = await api.call('GET', schedule); return !!saved.data.lastRunId }, { timeout: 12000, intervals: [100, 250] }).toBe(true)
    const { id, name, kind, intervalSeconds, cron, timezone, overlap, request, lastRunId } = saved.data
    await api.call('PUT', schedule, { id, name, kind, intervalSeconds, cron, timezone, overlap, request, enabled: false }, saved.revision)
    const delivery = await finished(lastRunId, 'delivered')
    expect(delivery.data.message.trigger).toBe('schedule'); expect(delivery.data.attempts).toBe(2)
    expect((await api.call('GET', `${base}/deliveries?run=${lastRunId}`)).items).toHaveLength(1)
    await refresh(); await expect(deliveryRow(delivery.id)).toContainText('schedule')
    await channelRow('availability').getByRole('button', { name: 'Pause', exact: true }).click()
    await expect(channelRow('availability')).toContainText('Paused')
  })

  await test.step('Preserve history through rename, archive, and paused restoration', async () => {
    await channelRow('availability').getByRole('button', { name: 'Edit', exact: true }).click()
    await page.getByLabel('Channel name', { exact: true }).fill('Renamed availability')
    await page.getByRole('button', { name: 'Save notification channel', exact: true }).click()
    await expect(channelRow('availability')).toContainText('Renamed availability')
    expect((await api.call('GET', `${base}/deliveries/${first.id}`)).data.channelName).toBe('Channel availability')
    await channelRow('availability').getByRole('button', { name: 'Archive', exact: true }).click()
    await expect(channelRow('availability')).toHaveCount(0)
    await page.getByLabel('Include archived channels', { exact: true }).check()
    await expect(channelRow('availability')).toContainText('Archived')
    await channelRow('availability').getByRole('button', { name: 'Restore', exact: true }).click()
    await expect(channelRow('availability')).toContainText('Paused')
    await expect(deliveryRow(first.id)).toContainText('Channel availability')
  })

  await test.step('Keep page help, themes, mobile layout, and project scope consistent', async () => {
    await channelRow('availability').getByRole('button', { name: 'Edit', exact: true }).click()
    await page.getByLabel('Channel name', { exact: true }).fill('Unsaved channel edit')
    await page.getByRole('button', { name: 'Help', exact: true }).click()
    await expect(page.getByRole('dialog', { name: 'Notifications help', exact: true })).toContainText('Delivery and uncertainty')
    await page.keyboard.press('Escape')
    await expect(page.getByLabel('Channel name', { exact: true })).toHaveValue('Unsaved channel edit')
    for (const theme of ['light', 'dark']) {
      await page.getByLabel('Color theme').selectOption(theme)
      await page.setViewportSize(theme === 'light' ? { width: 1280, height: 900 } : { width: 390, height: 844 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await testInfo.attach(`notifications-${theme}`, { body: await page.screenshot({ fullPage: true, animations: 'disabled' }), contentType: 'image/png' })
    }
    await page.getByLabel('Workspace', { exact: true }).selectOption(other.project)
    await followNavigation(page, 'Notifications')
    await expect(page.getByRole('heading', { name: 'Edit notification channel', exact: true })).toHaveCount(0)
    await page.getByRole('button', { name: '+ New channel', exact: true }).click()
    await expect(page.getByText(/No notification secret references are registered/)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Save notification channel', exact: true })).toBeDisabled()
    expect((await api.call('GET', `${base}/channels/availability`)).data.name).toBe('Renamed availability')
  })
})
