import type { DashboardResult } from '../api/dashboard'
import { dexFixture } from './dex-fixture'
export function dashboardFixture(): DashboardResult {
  return {
    collection_system: 'pokemon',
    collection: {
      historical_individuals: 0,
      held_individuals: 0,
      unknown_holding_individuals: 0,
      self_hunted_individuals: 0,
      unknown_source_individuals: 0,
      unknown_original_game_individuals: 0,
    },
    hunts: {
      total: 0,
      in_progress: 0,
      completed: 0,
      abandoned: 0,
      unknown_game: 0,
      unknown_count: 0,
      unknown_duration: 0,
    },
    dex: (['species', 'form'] as const).flatMap((dimension) =>
      (['historical', 'current'] as const).map((basis) => ({
        dimension,
        basis,
        summary: dexFixture().summary,
        coverage: dexFixture().coverage,
      })),
    ),
    rule: '永久个体身份；合成测试资料',
  }
}
