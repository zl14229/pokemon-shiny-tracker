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
  count_sample_status: 'no_eligible_samples' | 'single_sample' | 'descriptive_subset'
  time_sample_status: 'no_eligible_samples' | 'single_sample' | 'descriptive_subset'
  median_count_per_shiny: number | null
  median_seconds_per_shiny: number | null
  exclusions: Record<string, number>
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
  quality: {
    distinct_records: number
    candidate_groups: number
    candidate_records: number
    candidates: { record_ids: number[]; count: number }[]
    candidates_truncated: boolean
    count_comparison: string
    note: string
  }
}
export const efficiencyReasons: Record<string, string> = {
  unfinished_hunt: '未完成或已放弃',
  unknown_game: '游戏未知',
  unknown_method: '方法未知',
  date_filter: '日期子期间不能拆分累计投入',
  source_filter: '当前非自刷来源筛选',
  no_outcomes: '无关联成果',
  outcome_evidence: '关联成果来源或原始游戏不匹配',
  missing_count: '次数未知',
  missing_unit: '单位未知',
  missing_duration: '耗时未知',
  zero_duration: '耗时为零，不能计算效率',
  unsafe_count: '次数超出安全显示范围',
  unsafe_duration: '耗时超出安全显示范围',
}
const nullableCount = (v: unknown) => v === null || integer(v)
const ratio = (v: unknown) =>
  v === null ||
  (typeof v === 'number' && Number.isFinite(v) && v >= 0 && v <= Number.MAX_SAFE_INTEGER)
const status = (n: number) =>
  n === 0 ? 'no_eligible_samples' : n === 1 ? 'single_sample' : 'descriptive_subset'
function validQuality(q: unknown, individuals: number, contexts: number): boolean {
  if (
    !record(q) ||
    !integer(q.distinct_records) ||
    q.distinct_records !== individuals ||
    !integer(q.candidate_groups) ||
    !integer(q.candidate_records) ||
    q.candidate_records > individuals ||
    q.candidate_records < 2 * q.candidate_groups ||
    !Array.isArray(q.candidates) ||
    q.candidates.length !== Math.min(q.candidate_groups, 20) ||
    typeof q.candidates_truncated !== 'boolean' ||
    typeof q.note !== 'string' ||
    q.count_comparison !==
      (contexts === 0 ? 'unavailable' : contexts === 1 ? 'single_context' : 'separate_contexts')
  )
    return false
  const ids = new Set<number>()
  let seen = 0,
    truncated = q.candidate_groups > 20
  for (const c of q.candidates) {
    if (
      !record(c) ||
      !integer(c.count) ||
      c.count < 2 ||
      !Array.isArray(c.record_ids) ||
      c.record_ids.length !== Math.min(c.count, 100) ||
      !c.record_ids.every(id)
    )
      return false
    seen += c.count
    if (c.count > 100) truncated = true
    for (const n of c.record_ids) {
      if (ids.has(n)) return false
      ids.add(n)
    }
  }
  return (
    seen <= q.candidate_records &&
    (q.candidates_truncated || seen === q.candidate_records) &&
    q.candidates_truncated === truncated
  )
}
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
        Number(m.month.slice(0, 4)) > 0 &&
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
      !ratio(e.median_count_per_shiny) ||
      !ratio(e.median_seconds_per_shiny) ||
      e.count_sample_status !== status(e.count_samples as number) ||
      e.time_sample_status !== status(e.time_samples as number) ||
      ((e.count_samples as number) < 2
        ? e.median_count_per_shiny !== null
        : e.median_count_per_shiny === null) ||
      ((e.time_samples as number) < 2
        ? e.median_seconds_per_shiny !== null
        : e.median_seconds_per_shiny === null) ||
      !record(e.exclusions) ||
      !Object.entries(e.exclusions).every(
        ([k, n]) =>
          Object.hasOwn(efficiencyReasons, k) && integer(n) && n > 0 && n <= (e.hunts as number),
      ) ||
      (e.count_samples as number) > (e.known_count_hunts as number) ||
      (e.time_samples as number) > (e.known_duration_hunts as number) ||
      (e.count_outcomes as number) < (e.count_samples as number) ||
      (e.time_outcomes as number) < (e.time_samples as number) ||
      (e.count_samples === 0) !== (e.count_outcomes === 0) ||
      (e.time_samples === 0) !== (e.time_outcomes === 0)
    )
      return invalidResponse()
    if (
      (e.known_count_hunts === 0 && e.count !== null) ||
      (e.known_duration_hunts === 0 && e.seconds !== null) ||
      (!e.overflow &&
        (((e.known_count_hunts as number) > 0 && e.count === null) ||
          ((e.known_duration_hunts as number) > 0 && e.seconds === null) ||
          ((e.count_samples as number) > 0 && e.count_per_shiny === null) ||
          ((e.time_samples as number) > 0 && e.seconds_per_shiny === null)))
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
  if (!validQuality(v.quality, v.individuals as number, v.effort.length)) return invalidResponse()
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
