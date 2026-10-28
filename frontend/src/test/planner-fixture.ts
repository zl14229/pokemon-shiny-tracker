import type { Plan, Choice, Candidate } from '../api/planner'
export function planFixture(): Plan {
  const evidence = {
    source: 'synthetic test only',
    url: 'https://example.org/fact',
    version: 'test',
    scope: 'test game/form only',
    checked_at: '2026-10-29T00:00:00Z',
  }
  const choices = (id: number): Choice[] => [
    {
      game_id: 1,
      game_name: '测试版本',
      evidence: [
        {
          id,
          game_id: 1,
          species_id: id,
          form_id: id,
          batch_version: 'synthetic',
          fact: {
            verification: 'verified',
            availability: 'confirmed_available',
            direct_hunt: 'yes',
            guide_status: 'unknown',
            evidence,
            cross_checks: [],
            methods: [],
          },
        },
      ],
    },
  ]
  const candidate = (id: number): Candidate => ({
    key: `species:${id}`,
    species_id: id,
    form_id: null,
    name: `测试目标${id}`,
    name_status: 'unknown',
    games: choices(id),
  })
  return {
    id: 1,
    kind: 'random',
    status: 'draft',
    created_at: '2026-10-29T00:00:00Z',
    updated_at: '2026-10-29T00:00:00Z',
    rules: {
      game_ids: [1],
      dimension: 'species',
      pool: 'all_missing',
      count: 2,
      exclude_active: true,
    },
    coverage: {
      status: 'partial',
      note: '仅限本地已核实子集',
      games: [
        {
          game_id: 1,
          name: '测试版本',
          reviewed_entries: 2,
          verified_direct: 2,
          available_targets: 2,
        },
      ],
      versions: ['synthetic'],
      verified_targets: 2,
      excluded_obtained: 0,
      excluded_active: 0,
      available: 2,
      requested: 2,
      actual: 2,
      shortage: false,
    },
    targets: [1, 2].map((id) => ({
      id,
      plan_id: 1,
      position: id,
      target_key: `species:${id}`,
      species_id: id,
      form_id: null,
      game_id: 1,
      status: 'pending',
      hunt_id: null,
      shiny_record_id: null,
      snapshot: { ...candidate(id), choice: choices(id)[0]! },
    })),
  }
}
