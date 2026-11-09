export async function inspectRecordingTarget(page, selector) {
  const picked = page.locator(selector)
  const details = await picked.evaluate(element => {
    const semantic = 'article,li,tr,[role="listitem"],[role="row"]'
    const shape = node => [node.tagName, node.getAttribute('role') ?? '', [...node.children].map(child => child.tagName).join(',')].join('|')
    let item = element.closest(semantic)
    if (!item) {
      for (let parent = element.parentElement, depth = 0; parent && depth < 6 && !['BODY', 'HTML', 'MAIN', 'NAV', 'HEADER', 'FOOTER'].includes(parent.tagName); parent = parent.parentElement, depth++) {
        const siblings = [...(parent.parentElement?.children ?? [])].filter(node => shape(node) === shape(parent))
        if (siblings.length > 1 && parent.querySelector('h1,h2,h3,h4,h5,h6,[role="heading"]') && parent.textContent.trim().length > element.textContent.trim().length) { item = parent; break }
      }
    }
    item ??= element
    const segments = []
    let parent = item.parentElement
    while (parent && parent.tagName !== 'HTML') {
      if (parent.id) { segments.unshift('#' + CSS.escape(parent.id)); break }
      const tag = parent.tagName.toLowerCase()
      const siblings = parent.parentElement ? [...parent.parentElement.children].filter(child => child.tagName === parent.tagName) : []
      segments.unshift(tag + (siblings.length > 1 ? ':nth-of-type(' + (siblings.indexOf(parent) + 1) + ')' : ''))
      parent = parent.parentElement
    }
    const role = item.getAttribute('role'), tag = item.tagName.toLowerCase()
    const peers = [...(item.parentElement?.children ?? [])].filter(node => shape(node) === shape(item))
    const classes = [...item.classList].filter(name => !['active', 'selected', 'hover', 'focus'].includes(name) && !name.startsWith('ng-') && peers.every(node => node.classList.contains(name))).slice(0, 3)
    segments.push(tag + (role ? '[role=' + JSON.stringify(role) + ']' : '') + classes.map(name => '.' + CSS.escape(name)).join(''))
    const capabilities = ['visible', 'text']
    if (element.matches('button,input,select,textarea,[role="button"]')) capabilities.push('enabled')
    if (element.matches('input[type="checkbox"],input[type="radio"]')) capabilities.push('checked')
    if (element.matches('textarea,select:not([multiple]),input:not([type="password"]):not([type="file"]):not([type="hidden"])')) capabilities.push('value')
    return { grouped: segments.join(' > '), label: ({ article: 'Articles', li: 'List items', tr: 'Table rows' })[tag] ?? (item !== element ? 'Similar cards' : 'Matching items'), elementLabel: (element.getAttribute('aria-label') || element.textContent?.trim() || element.getAttribute('placeholder') || element.tagName.toLowerCase()).slice(0, 160), text: (element.textContent ?? '').trim().slice(0, 200), capabilities, ...(capabilities.includes('value') ? { value: element.value.slice(0, 2048) } : {}) }
  }, undefined, { timeout: 5000 })
  const frames = selector.split(' >> internal:control=enter-frame >> ')
  return { ...details, selector, grouped: [...frames.slice(0, -1), details.grouped].join(' >> internal:control=enter-frame >> ') }
}

export async function recordingTargetView(page, target) {
  const picked = page.locator(target.selector)
  const box = await picked.boundingBox({ timeout: 1000 })
  const local = box ? await picked.evaluate(element => ({ x: element.getBoundingClientRect().x, y: element.getBoundingClientRect().y }), undefined, { timeout: 1000 }) : null
  const boxes = await page.locator(target.grouped).evaluateAll(elements => elements.map(element => { const r = element.getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width, height: r.height } }).filter(r => r.width > 0 && r.height > 0 && r.x + r.width > 0 && r.y + r.height > 0 && r.x < innerWidth && r.y < innerHeight).slice(0, 30))
  const bounds = boxes.map(r => ({ ...r, x: r.x + (box && local ? box.x - local.x : 0), y: r.y + (box && local ? box.y - local.y : 0) }))
  return { label: target.label, elementLabel: target.elementLabel, matches: await page.locator(target.grouped).filter({ visible: true }).count(), text: target.text, capabilities: target.capabilities, ...(target.value !== undefined ? { value: target.value } : {}), bounds, elementBounds: box ? [box] : [] }
}

export async function recordingCheck(page, target, command) {
  const counts = ['at-least', 'at-most', 'exactly']
  const texts = ['text', 'exact-text', 'value']
  const states = { visible: ['toBeVisible', true, 'is visible'] }
  const allowed = ['visible', 'enabled', 'disabled', 'checked', 'unchecked', ...counts, ...texts]
  if (!allowed.includes(command.key)) throw new Error('Unsupported check')
  const grouped = counts.includes(command.key) && command.scope !== 'element'
  const locator = `page.locator(${JSON.stringify(grouped ? target.grouped : target.selector)})${counts.includes(command.key) ? ".filter({ visible: true })" : ""}`
  const picked = page.locator(target.selector)
  if (counts.includes(command.key)) {
    if (!Number.isInteger(command.count) || command.count < (command.key === 'at-least' ? 1 : 0) || command.count > 10000) throw new Error('Enter a valid item count')
    const matches = await page.locator(grouped ? target.grouped : target.selector).filter({ visible: true }).count()
    if (command.key === 'at-least' ? matches < command.count : command.key === 'at-most' ? matches > command.count : matches !== command.count) return { message: `There are ${matches} matching items now. Change the expected count or choose different items before adding this check.` }
    const operator = command.key === 'at-least' ? 'at least' : command.key === 'at-most' ? 'at most' : 'exactly'
    return { kind: 'assertCount', description: `Check ${grouped ? target.label.toLowerCase() : 'the selected item'}: ${operator} ${command.count}`, statement: command.key === 'exactly' ? `await expect(${locator}).toHaveCount(${command.count});` : `await expect.poll(() => ${locator}.count()).${command.key === 'at-least' ? 'toBeGreaterThanOrEqual' : 'toBeLessThanOrEqual'}(${command.count});` }
  }
  if (texts.includes(command.key)) {
    if (typeof command.text !== 'string' || command.text.length > 2048 || command.key !== 'value' && !command.text.length) throw new Error('Enter the expected text')
    if (command.key === 'value' && !target.capabilities.includes('value')) throw new Error('Select a field with a readable value')
    const current = command.key === 'value' ? await picked.inputValue() : (await picked.textContent()) ?? ''
    const normalize = value => value.replace(/\s+/g, ' ').trim()
    const matches = command.key === 'value' ? current === command.text : command.key === 'text' ? normalize(current).includes(normalize(command.text)) : normalize(current) === normalize(command.text)
    if (!matches) return { message: 'The selected item does not match that expected text or value now. Review it before adding this check.' }
    return { kind: command.key === 'value' ? 'assertValue' : 'assertText', description: `Check the selected ${command.key === 'value' ? 'field value equals' : command.key === 'exact-text' ? 'text equals' : 'item contains'} “${command.text}”`, statement: `await expect(${locator}).${command.key === 'value' ? 'toHaveValue' : command.key === 'exact-text' ? 'toHaveText' : 'toContainText'}(${JSON.stringify(command.text)});` }
  }
  if (['checked', 'unchecked'].includes(command.key)) {
    if (!target.capabilities.includes('checked')) throw new Error('Select a checkbox or radio button')
    const checked = await picked.isChecked()
    states[command.key] = ['toBeChecked', command.key === 'checked' ? checked : !checked, command.key === 'checked' ? 'is checked' : 'is unchecked']
  }
  if (['enabled', 'disabled'].includes(command.key) && !target.capabilities.includes('enabled')) throw new Error('Select a form control')
  if (['enabled', 'disabled'].includes(command.key)) {
    const enabled = await picked.isEnabled()
    states[command.key] = [command.key === 'enabled' ? 'toBeEnabled' : 'toBeDisabled', command.key === 'enabled' ? enabled : !enabled, `is ${command.key}`]
  }
  if (command.key === 'visible') states.visible[1] = await picked.isVisible()
  const [method, matches, description] = states[command.key]
  if (!matches) return { message: `The selected item does not match “${description}” now. Review its state before adding this check.` }
  return { kind: 'assertState', description: `Check the selected item ${description}`, statement: `await expect(${locator})${command.key === 'unchecked' ? '.not' : ''}.${method}();` }
}

export async function recordingSelectorAtPoint(page, playwright, progress, x, y) {
  let frame = page.mainFrame(), point = { x, y }
  for (let depth = 0; depth < 16; depth++) {
    const handle = await frame.evaluateHandle(({ x, y }) => {
      let element = document.elementFromPoint(x, y)
      while (element?.shadowRoot?.elementFromPoint) {
        const child = element.shadowRoot.elementFromPoint(x, y)
        if (!child || child === element) break
        element = child
      }
      return element
    }, point)
    try {
      const element = handle.asElement()
      if (!element) throw new Error('Choose an item inside the website')
      const child = await element.contentFrame()
      if (child) {
        const box = await element.boundingBox()
        const border = await element.evaluate(node => ({ x: node.clientLeft, y: node.clientTop }))
        if (!box) throw new Error('The frame is no longer visible')
        frame = child; point = { x: x - box.x - border.x, y: y - box.y - border.y }
        continue
      }
      const backend = playwright._connection.toImpl(element)
      const parts = [await backend.evaluateInUtility(([injected, node]) => injected.generateSelectorSimple(node), undefined)]
      for (let owner = backend._frame; owner.parentFrame(); owner = owner.parentFrame()) {
        const ownerElement = await owner.frameElement(progress)
        try { parts.unshift(await ownerElement.evaluateInUtility(([injected, node]) => injected.generateSelectorSimple(node), undefined)) }
        finally { ownerElement.dispose() }
      }
      return parts.join(' >> internal:control=enter-frame >> ')
    } finally { await handle.dispose() }
  }
  throw new Error('Choose an item in a supported website frame')
}
