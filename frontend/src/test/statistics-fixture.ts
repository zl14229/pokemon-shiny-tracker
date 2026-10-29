import type { StatisticsResult } from '../api/statistics'
export function statisticsFixture(): StatisticsResult {
  return {
    filters: { collection_system: 'pokemon' },
    individuals: 0,
    hunts: 0,
    by_game: [],
    by_method: [],
    by_source: [],
    trend: [],
    effort: [],
    quality: {
      distinct_records: 0,
      candidate_groups: 0,
      candidate_records: 0,
      candidates: [],
      candidates_truncated: false,
      count_comparison: 'unavailable',
      note: '候选不等于重复个体',
    },
    coverage: {
      unknown_obtained_date: 0,
      partial_obtained_date: 0,
      unknown_start_date: 0,
      partial_start_date: 0,
      unplaced_records: 0,
      unplaced_hunts: 0,
      unknown_original_game: 0,
      unknown_method: 0,
    },
    rule: '合成测试；未知不补0',
  }
}
