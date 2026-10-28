import { describe, it, expect } from 'vitest'
import { decodeDashboard } from './dashboard'
import { dashboardFixture } from '../test/dashboard-fixture'
describe('dashboard response', () => {
  it('preserves unknown denominator and true empty counts', () => {
    const v = decodeDashboard({ data: dashboardFixture() })
    expect(v.dex[0]?.summary.percentage).toBeNull()
    expect(v.collection.historical_individuals).toBe(0)
  })
  it('rejects missing counts, contradictory totals and duplicate dimensions', () => {
    for (const change of [
      (v: any) => {
        delete v.hunts.unknown_count
      },
      (v: any) => {
        v.hunts.total = 1
      },
      (v: any) => {
        v.dex[1] = v.dex[0]
      },
      (v: any) => {
        v.dex[0].summary.percentage = 0
      },
      (v: any) => {
        v.collection.historical_individuals = -1
      },
    ]) {
      const v = dashboardFixture()
      change(v)
      expect(() => decodeDashboard({ data: v })).toThrow()
    }
  })
})
