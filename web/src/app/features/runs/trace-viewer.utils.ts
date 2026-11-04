export function privateTraceURL(value: string | undefined, artifactURL: string): string {
  if (!value) return ''
  const viewer = new URL(value), artifact = new URL(artifactURL)
  const token = /^\/access\/([A-Za-z0-9_-]{43})$/.exec(artifact.pathname)?.[1]
  if (!token || !['http:', 'https:'].includes(viewer.protocol) || viewer.origin !== artifact.origin || viewer.username || viewer.password || viewer.hash || viewer.pathname !== `/trace/${token}/index.html` || new URL(viewer.searchParams.get('trace') ?? '').href !== artifact.href || !Number.isFinite(Date.parse(viewer.searchParams.get('expires') ?? ''))) throw new Error('The private trace viewer address does not match its artifact access grant.')
  return viewer.href
}
