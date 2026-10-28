import { api, type RequestOptions } from './client'
import { decodeDex, type DexResult } from './dex'
import { record, integer } from './models'
import { invalidResponse } from './tracker'

export interface DashboardDex {
  dimension: 'species' | 'form'
  basis: 'historical' | 'current'
  summary: DexResult['summary']
  coverage: DexResult['coverage']
}
export interface DashboardResult {
  collection_system: string
  collection: {
    historical_individuals: number
    held_individuals: number
    unknown_holding_individuals: number
    self_hunted_individuals: number
    unknown_source_individuals: number
    unknown_original_game_individuals: number
  }
  hunts: {
    total: number
    in_progress: number
    completed: number
    abandoned: number
    unknown_game: number
    unknown_count: number
    unknown_duration: number
  }
  dex: DashboardDex[]
  rule: string
}
export function decodeDashboard(body: unknown): DashboardResult {
  if (!record(body) || !record(body.data)) return invalidResponse()
  const v = body.data
  if (
    typeof v.collection_system !== 'string' ||
    !v.collection_system ||
    typeof v.rule !== 'string' ||
    !record(v.collection) ||
    !record(v.hunts) ||
    !Array.isArray(v.dex) ||
    v.dex.length !== 4
  )
    return invalidResponse()
  const c = v.collection,
    h = v.hunts
  if (
    ![
      'historical_individuals',
      'held_individuals',
      'unknown_holding_individuals',
      'self_hunted_individuals',
      'unknown_source_individuals',
      'unknown_original_game_individuals',
    ].every((k) => integer(c[k])) ||
    ![
      'total',
      'in_progress',
      'completed',
      'abandoned',
      'unknown_game',
      'unknown_count',
      'unknown_duration',
    ].every((k) => integer(h[k]))
  )
    return invalidResponse()
  if (
    (h.in_progress as number) + (h.completed as number) + (h.abandoned as number) !== h.total ||
    ['unknown_game', 'unknown_count', 'unknown_duration'].some(
      (k) => (h[k] as number) > (h.total as number),
    ) ||
    [
      'self_hunted_individuals',
      'unknown_source_individuals',
      'unknown_original_game_individuals',
    ].some((k) => (c[k] as number) > (c.historical_individuals as number))
  )
    return invalidResponse()
  const seen = new Set<string>()
  for (const d of v.dex) {
    if (!record(d)) return invalidResponse()
    const parsed = decodeDex({
      data: [],
      meta: { total: 0, page: 1, page_size: 1 },
      filters: {
        dimension: d.dimension,
        basis: d.basis,
        scope: 'all',
        display: 'all',
        collection_system: v.collection_system,
      },
      summary: d.summary,
      coverage: d.coverage,
    })
    const key = `${parsed.filters.dimension}/${parsed.filters.basis}`
    if (seen.has(key)) return invalidResponse()
    seen.add(key)
  }
  return v as unknown as DashboardResult
}
export const dashboard = {
  async get(system = 'pokemon', options?: RequestOptions) {
    const v = decodeDashboard(
      await api.getJSON(`/dashboard?collection_system=${encodeURIComponent(system)}`, options),
    )
    if (v.collection_system !== system) return invalidResponse()
    return v
  },
}
