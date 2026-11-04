import path from 'node:path'
import { execFileSync } from 'node:child_process'

export function dockerBuildBackend({ buildxPath, architecture, host, requireBuildKit = false }) {
  try {
    const command = buildxPath ? path.resolve(buildxPath) : 'docker'
    const prefix = buildxPath ? [] : ['buildx']
    const version = execFileSync(command, [...prefix, 'version'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim()
    const contextName = execFileSync('docker', ['context', 'show'], { encoding: 'utf8' }).trim()
    return { command, prefix: [...prefix, 'build', '--builder', contextName, '--load', '--provenance=false'], version }
  } catch {
    if (buildxPath || architecture !== host || requireBuildKit) throw new Error('This package requires Docker buildx. Install the plugin or select its trusted executable with --buildx=PATH.')
    return { command: 'docker', prefix: ['build'], version: 'docker-legacy' }
  }
}
