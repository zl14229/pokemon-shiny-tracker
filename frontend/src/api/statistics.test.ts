import { it, expect } from 'vitest'
import { decodeStatistics } from './statistics'
import { statisticsFixture } from '../test/statistics-fixture'
it('validates true empty summaries and rejects missing/unsafe/contradictory counts', () => {
  expect(decodeStatistics({ data: statisticsFixture() }).effort).toEqual([])
  for (const mutate of [
    (v: any) => delete v.coverage.unplaced_records,
    (v: any) => (v.hunts = 1),
    (v: any) => (v.individuals = Number.MAX_SAFE_INTEGER + 1),
    (v: any) =>
      v.trend.push({ month: '2024-99', started_hunts: 0, individuals: 0, self_hunted: 0 }),
  ]) {
    const v = statisticsFixture()
    mutate(v)
    expect(() => decodeStatistics({ data: v })).toThrow()
  }
})
