import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { TrendPoint } from './trends.models'
import { stackedChart, pointSummary } from './trend-chart.utils.ts'
import { defaultTrendFilters, readTrendFilters, trendFilterIssue, trendRequest } from './trends.utils.ts'

function point(run: string, passed: number, timedOut = 0, gap?: string): TrendPoint {
  return { runId: run, at: '2026-11-02T12:00:00Z', suiteId: 'suite', executionLocation: 'local', state: 'completed', outcome: 'passed', counts: { total: passed + timedOut, passed, failed: 0, timedOut, flaky: 0, skipped: 0, unknown: 0 }, gap }
}

test('stacked areas preserve the total height and break at missing history', () => {
  const chart = stackedChart([point('a', 2, 1), point('gap', 0, 0, 'Not selected'), point('b', 1, 1)])
  assert.equal(chart.maximum, 3)
  assert.equal(chart.areas.length, 6)
  assert.equal(chart.areas[0].path.match(/M/g)?.length, 2)
  assert.equal(chart.areas[2].path.match(/M/g)?.length, 2)
  assert.equal(chart.areas[2].path.includes('18.00'), true)
  assert.equal(chart.columns[1].point.gap, 'Not selected')
  assert.match(pointSummary(chart.columns[0].point), /1 timed out/)
  assert.match(pointSummary(chart.columns[1].point), /Not selected/)
  assert.equal(stackedChart([]).areas.every(area => area.path === ''), true)
})

test('UTC scope accepts real dates and retains exact category and test filters', () => {
  const filters = defaultTrendFilters(Date.parse('2026-11-02T23:00:00Z'))
  assert.equal(filters.from, '2026-10-04')
  assert.equal(filters.to, '2026-11-02')
  assert.equal(trendFilterIssue({ ...filters, from: '2026-02-30' }) !== '', true)
  assert.equal(trendFilterIssue({ ...filters, test: 'A' }) !== '', true)
  assert.equal(trendFilterIssue({ ...filters, from: '2020-01-01' }) !== '', true)
  const saved = readTrendFilters({ ...filters, suite: 'suite', test: 'A', category: 'API / health' }, 0)
  assert.equal(trendFilterIssue(saved), '')
  const query = new URLSearchParams(trendRequest(saved))
  assert.equal(query.get('until'), '2026-11-02T23:59:59.999Z')
  assert.equal(query.get('category'), 'API / health')
})

test('connected stacked polygons retain the integral of their logical counts', () => {
  const points = [point('a', 2, 1), point('b', 0, 2), point('c', 1, 0)]
  const chart = stackedChart(points)
  for (const state of ['passed', 'timedOut'] as const) {
    const path = chart.areas.find(area => area.key === state)!.path
    const coordinates = [...path.matchAll(/([\d.]+),([\d.]+)/g)].map(match => [Number(match[1]), Number(match[2])])
    let sum = 0
    coordinates.forEach(([x, y], index) => { const next = coordinates[(index + 1) % coordinates.length]; sum += x * next[1] - next[0] * y })
    const expected = points.reduce((total, item) => total + item.counts[state], 0) * (chart.right - chart.left) / points.length * (chart.bottom - chart.top) / chart.maximum
    assert.ok(Math.abs(Math.abs(sum / 2) - expected) < 2)
  }
})
