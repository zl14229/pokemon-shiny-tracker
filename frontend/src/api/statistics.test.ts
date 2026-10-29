import { it, expect } from 'vitest'
import { decodeStatistics } from './statistics'
import { statisticsFixture } from '../test/statistics-fixture'
import type { Effort } from './statistics'
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

function oneSample() {
  const v = statisticsFixture()
  v.individuals = 1
  v.hunts = 1
  v.by_game = [{ key: '1', label: 'A', hunts: 1, individuals: 1, self_hunted: 1 }]
  v.by_method = [{ key: 'wild', label: 'wild', hunts: 1, individuals: 1, self_hunted: 1 }]
  v.by_source = [
    { key: 'self_hunted', label: 'self_hunted', hunts: 0, individuals: 1, self_hunted: 1 },
  ]
  v.coverage.unknown_obtained_date = 1
  v.coverage.unknown_start_date = 1
  v.coverage.unplaced_records = 1
  v.coverage.unplaced_hunts = 1
  v.quality.distinct_records = 1
  v.quality.count_comparison = 'single_context'
  v.effort = [
    {
      key: '1/wild/encounters',
      game_id: 1,
      game: 'A',
      method: 'wild',
      unit: 'encounters',
      hunts: 1,
      known_count_hunts: 1,
      known_duration_hunts: 0,
      count: 0,
      seconds: null,
      count_samples: 1,
      time_samples: 0,
      count_outcomes: 1,
      time_outcomes: 0,
      count_per_shiny: 0,
      seconds_per_shiny: null,
      overflow: false,
      note: 'fixture',
      count_sample_status: 'single_sample',
      time_sample_status: 'no_eligible_samples',
      median_count_per_shiny: null,
      median_seconds_per_shiny: null,
      exclusions: { missing_duration: 1 },
    } satisfies Effort,
  ]
  return v
}
it('keeps known zero separate from missing time and rejects fabricated efficiency evidence', () => {
  expect(decodeStatistics({ data: oneSample() }).effort[0]?.count_per_shiny).toBe(0)
  expect(decodeStatistics({ data: oneSample() }).effort[0]?.seconds).toBeNull()
  for (const mutate of [
    (e: any) => (e.seconds = 0),
    (e: any) => (e.seconds_per_shiny = 0),
    (e: any) => (e.median_count_per_shiny = 0),
    (e: any) => (e.count_sample_status = 'descriptive_subset'),
    (e: any) => (e.exclusions.missing_duration = 2),
  ]) {
    const v = oneSample()
    mutate(v.effort[0])
    expect(() => decodeStatistics({ data: v })).toThrow()
  }
})

it('rejects contradictory identity diagnostics rather than subtracting candidates', () => {
  for (const change of [
    (v: any) => (v.quality.distinct_records = 1),
    (v: any) => (v.quality.candidate_groups = 1),
    (v: any) => (v.quality.candidates_truncated = true),
    (v: any) => (v.quality.count_comparison = 'single_context'),
  ]) {
    const v = statisticsFixture()
    change(v)
    expect(() => decodeStatistics({ data: v })).toThrow()
  }
})
