import type { BingoProof, Plan, PlanTarget } from '../api/planner'
import { planFixture } from './planner-fixture'
export function bingoFixture(): Plan {
  const p = planFixture(),
    base = p.targets[0]!,
    start = new Date(Date.now() - 3600000).toISOString()
  p.id = 7
  p.kind = 'bingo'
  p.status = 'active'
  p.created_at = start
  p.updated_at = start
  p.rules.count = 9
  Object.assign(p.coverage, {
    verified_targets: 12,
    available: 12,
    actual: 9,
    requested: 9,
    shortage: false,
  })
  Object.assign(p.coverage.games[0]!, {
    reviewed_entries: 12,
    verified_direct: 12,
    available_targets: 12,
  })
  p.targets = Array.from({ length: 9 }, (_, i) => {
    const t = structuredClone(base),
      id = i + 1
    t.id = id
    t.plan_id = 7
    t.position = id
    t.target_key = `species:${id}`
    t.species_id = id
    t.snapshot.key = t.target_key
    t.snapshot.species_id = id
    t.snapshot.name = `测试目标${id}`
    for (const g of [...t.snapshot.games, t.snapshot.choice]) {
      g.evidence[0]!.id = id
      g.evidence[0]!.species_id = id
      g.evidence[0]!.form_id = id
    }
    return t
  })
  p.bingo = {
    size: 3,
    started_at: start,
    calendar_zone: 'UTC',
    lines: [],
    cells: p.targets.map((t) => ({ target_id: t.id, kind: 'none', reviews: [] })),
    events: [
      {
        id: 1,
        plan_id: 7,
        target_id: null,
        kind: 'created',
        summary: '创建 3×3 挑战（无免费格）',
        created_at: start,
      },
    ],
  }
  return p
}
export function completeFixtureCell(p: Plan, t: PlanTarget) {
  t.status = 'completed'
  t.shiny_record_id = 100 + t.id
  const proof: BingoProof = {
    method: 'manual_timestamp',
    record_id: t.shiny_record_id,
    obtained_date: null,
    obtained_at: new Date(Date.now() - 1800000).toISOString(),
    evidence_note: 'synthetic timestamp evidence',
    confirmed_at: new Date().toISOString(),
    original_species_id: t.species_id,
    original_form_id: t.snapshot.choice.evidence[0]!.form_id,
    original_game_id: t.game_id,
    source: 'self_hunted',
    availability: t.snapshot.choice.evidence[0]!,
  }
  const cell = p.bingo!.cells.find((c) => c.target_id === t.id)!
  cell.kind = 'completed'
  cell.proof = proof
}
