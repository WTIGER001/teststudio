export function parseVersion(value) {
  const match = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([a-z][a-z0-9-]*))?$/.exec(value.trim())
  if (!match || match[4] === 'stable') throw new Error('Use MAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH-TAG. Tags use lowercase letters, numbers, and hyphens.')
  const [major, minor, patch] = match.slice(1, 4).map(Number), tag = match[4] ?? 'stable'
  if (![major, minor, patch].every(Number.isSafeInteger)) throw new Error('Version numbers exceed the supported integer range.')
  return { version: match[0], displayVersion: `${major}.${minor}.${patch}.${tag}`, major, minor, patch, tag }
}

export function nextVersion(current, action, tag) {
  const value = parseVersion(current)
  if (action === 'major') { value.major++; value.minor = 0; value.patch = 0 }
  else if (action === 'minor') { value.minor++; value.patch = 0 }
  else if (action === 'patch') value.patch++
  else if (action === 'tag' && /^[a-z][a-z0-9-]*$/.test(tag ?? '')) value.tag = tag
  else throw new Error('Use major, minor, patch, or tag TAG.')
  return parseVersion(`${value.major}.${value.minor}.${value.patch}${value.tag === 'stable' ? '' : '-' + value.tag}`).version
}

export function buildLinkerFlags(info) {
  return `-X github.com/wtiger001/teststudio/internal/buildinfo.Manifest=${Buffer.from(JSON.stringify(info)).toString('base64')}`
}
