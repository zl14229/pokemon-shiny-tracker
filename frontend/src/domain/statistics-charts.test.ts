import { it, expect } from 'vitest'
import { statisticsCharts } from './statistics-charts'
import { statisticsFixture } from '../test/statistics-fixture'
import type { Effort } from '../api/statistics'
it('does not chart incomparable units or replace unknown effort with zero', () => {
  const v = statisticsFixture()
  v.effort = [
    { game: 'A', method: 'wild', unit: 'eggs', count: 100, seconds: null, seconds_per_shiny: null },
    {
      game: 'A',
      method: 'wild',
      unit: 'encounters',
      count: 200,
      seconds: 0,
      seconds_per_shiny: null,
    },
  ] as Effort[]
  const charts = statisticsCharts(v)
  expect((charts[4]!.option.series as any)[0].data).toEqual([null, null])
  expect((charts[5]!.option.series as any)[0].data).toEqual([null, 0])
  expect((charts[6]!.option.series as any)[0].data).toEqual([null, null])
})
