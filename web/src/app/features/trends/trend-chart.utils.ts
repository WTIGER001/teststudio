import type { TrendPoint, TrendState } from './trends.models'

export const trendStates: { key: TrendState; label: string }[] = [
  { key: 'passed', label: 'Passed' }, { key: 'failed', label: 'Failed' }, { key: 'timedOut', label: 'Timed out' },
  { key: 'flaky', label: 'Flaky' }, { key: 'skipped', label: 'Skipped' }, { key: 'unknown', label: 'Unknown / incomplete' }
]

export function pointSummary(point: TrendPoint): string {
  if (point.gap) return `${point.runId}: ${point.gap}`
  return `${point.runId}: ${trendStates.map(state => `${point.counts[state.key]} ${state.label.toLowerCase()}`).join(', ')}. Total ${point.counts.total}.`
}

export function stackedChart(points: TrendPoint[]) {
  const left = 48, right = 884, top = 18, bottom = 254
  const maximum = Math.max(1, ...points.map(point => point.gap ? 0 : point.counts.total))
  const width = (right - left) / Math.max(1, points.length)
  const y = (value: number) => bottom - value / maximum * (bottom - top)
  const columns = points.map((point, index) => ({ point, index, x: left + index * width, width, summary: pointSummary(point) }))
  const lower = points.map(() => 0)
  const areas = trendStates.map(state => {
    const paths: string[] = []
    let upper: [number, number][] = [], under: [number, number][] = []
    let start = 0, end = 0
    const finish = () => {
      if (upper.length) {
        const polygon = [[start, upper[0][1]], ...upper, [end, upper.at(-1)![1]], [end, under.at(-1)![1]], ...under.slice().reverse(), [start, under[0][1]]]
        paths.push(`M${polygon.map(([x, y]) => `${x.toFixed(2)},${y.toFixed(2)}`).join(' L')} Z`)
      }
      upper = []; under = []
    }
    columns.forEach(column => {
      if (column.point.gap) { finish(); return }
      if (!upper.length) start = column.x
      end = column.x + width
      const base = lower[column.index], value = column.point.counts[state.key]
      upper.push([column.x + width / 2, y(base + value)])
      under.push([column.x + width / 2, y(base)])
      lower[column.index] += value
    })
    finish()
    return { ...state, path: paths.join(' ') }
  })
  const ticks = [...new Set([0, Math.ceil(maximum / 2), maximum])].map(value => ({ value, y: y(value) }))
  return { columns, areas, ticks, left, right, top, bottom, maximum }
}
