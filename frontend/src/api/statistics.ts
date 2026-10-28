import { api, type RequestOptions } from './client'
import { id, integer, record } from './models'
import { invalidResponse } from './tracker'
export interface Distribution {
  key: string
  label: string
  hunts: number
  individuals: number
  self_hunted: number
}
export interface Effort {
  key: string
  game_id: number | null
  game: string
  method: string | null
  unit: string | null
  hunts: number
  known_count_hunts: number
  known_duration_hunts: number
  count: number | null
  seconds: number | null
  count_samples: number
  time_samples: number
  count_outcomes: number
  time_outcomes: number
  count_per_shiny: number | null
  seconds_per_shiny: number | null
  overflow: boolean
  note: string
}
export interface StatisticsResult {
  filters: Record<string, string>
  individuals: number
  hunts: number
  by_game: Distribution[]
  by_method: Distribution[]
  by_source: Distribution[]
  trend: { month: string; started_hunts: number; individuals: number; self_hunted: number }[]
  effort: Effort[]
  coverage: {
    unknown_obtained_date: number
    partial_obtained_date: number
    unknown_start_date: number
    partial_start_date: number
    unplaced_records: number
    unplaced_hunts: number
    unknown_original_game: number
    unknown_method: number
  }
  rule: string
}
const nullableCount = (v: unknown) => v === null || integer(v)
const ratio = (v: unknown) => v === null || (typeof v === 'number' && Number.isFinite(v) && v >= 0)
export function decodeStatistics(body: unknown): StatisticsResult {
  if (!record(body) || !record(body.data)) return invalidResponse()
  const v = body.data
  if (
    !record(v.filters) ||
    !Object.values(v.filters).every((x) => typeof x === 'string') ||
    typeof v.filters.collection_system !== 'string' ||
    !integer(v.individuals) ||
    !integer(v.hunts) ||
    typeof v.rule !== 'string' ||
    !record(v.coverage)
  )
    return invalidResponse()
  if (
    ![
      'unknown_obtained_date',
      'partial_obtained_date',
      'unknown_start_date',
      'partial_start_date',
      'unplaced_records',
      'unplaced_hunts',
      'unknown_original_game',
      'unknown_method',
    ].every((k) => integer(v.coverage && (v.coverage as Record<string, unknown>)[k]))
  )
    return invalidResponse()
  for (const field of ['by_game', 'by_method', 'by_source']) {
    const rows = v[field]
    if (
      !Array.isArray(rows) ||
      !rows.every(
        (r) =>
          record(r) &&
          typeof r.key === 'string' &&
          typeof r.label === 'string' &&
          ['hunts', 'individuals', 'self_hunted'].every((k) => integer(r[k])) &&
          (r.self_hunted as number) <= (r.individuals as number),
      ) ||
      new Set(rows.map((r) => r.key)).size !== rows.length ||
      rows.reduce((n, r) => n + r.individuals, 0) !== v.individuals ||
      (field !== 'by_source' && rows.reduce((n, r) => n + r.hunts, 0) !== v.hunts)
    )
      return invalidResponse()
  }
  if (
    !Array.isArray(v.trend) ||
    !v.trend.every(
      (m) =>
        record(m) &&
        typeof m.month === 'string' &&
        /^\d{4}-(0[1-9]|1[0-2])$/.test(m.month) &&
        integer(m.started_hunts) &&
        integer(m.individuals) &&
        integer(m.self_hunted) &&
        m.self_hunted <= m.individuals,
    ) ||
    new Set(v.trend.map((m) => m.month)).size !== v.trend.length
  )
    return invalidResponse()
  if (!Array.isArray(v.effort)) return invalidResponse()
  for (const e of v.effort) {
    if (
      !record(e) ||
      typeof e.key !== 'string' ||
      (e.game_id !== null && !id(e.game_id)) ||
      typeof e.game !== 'string' ||
      (e.method !== null && typeof e.method !== 'string') ||
      (e.unit !== null && typeof e.unit !== 'string') ||
      typeof e.note !== 'string' ||
      typeof e.overflow !== 'boolean' ||
      ![
        'hunts',
        'known_count_hunts',
        'known_duration_hunts',
        'count_samples',
        'time_samples',
        'count_outcomes',
        'time_outcomes',
      ].every((k) => integer(e[k])) ||
      !nullableCount(e.count) ||
      !nullableCount(e.seconds) ||
      !ratio(e.count_per_shiny) ||
      !ratio(e.seconds_per_shiny)
    )
      return invalidResponse()
    if (
      ['known_count_hunts', 'known_duration_hunts', 'count_samples', 'time_samples'].some(
        (k) => (e[k] as number) > (e.hunts as number),
      ) ||
      ((e.count_samples === 0 || e.count_outcomes === 0) && e.count_per_shiny !== null) ||
      ((e.time_samples === 0 || e.time_outcomes === 0) && e.seconds_per_shiny !== null) ||
      (e.overflow && (e.count !== null || e.seconds !== null))
    )
      return invalidResponse()
  }
  if (
    new Set(v.effort.map((e) => e.key)).size !== v.effort.length ||
    v.effort.reduce((n, e) => n + e.hunts, 0) !== v.hunts
  )
    return invalidResponse()
  const c = v.coverage as StatisticsResult['coverage']
  if (
    v.trend.reduce((n, m) => n + m.individuals, 0) + c.unplaced_records !== v.individuals ||
    v.trend.reduce((n, m) => n + m.started_hunts, 0) + c.unplaced_hunts !== v.hunts ||
    c.unknown_obtained_date + c.partial_obtained_date > v.individuals ||
    c.unknown_start_date + c.partial_start_date > v.hunts ||
    c.unknown_original_game > v.individuals ||
    c.unknown_method > v.hunts
  )
    return invalidResponse()
  return v as unknown as StatisticsResult
}
export const statistics = {
  async get(query: string, options?: RequestOptions) {
    return decodeStatistics(await api.getJSON(`/statistics${query ? '?' + query : ''}`, options))
  },
}
