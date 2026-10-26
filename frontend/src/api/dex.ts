import { api, type RequestOptions } from './client'
import { id, integer, record } from './models'
import { invalidResponse } from './tracker'

export interface DexTarget {
  id: number
  species_id: number
  form_id: number | null
  national_dex: number
  slug: string
  name: string
  names: Record<string, string>
  name_status: string
  cross_check_status: string
  inclusion: string
  inclusion_reason: string
  source_url: string
  cross_check_url: string
  revision: string
  availability: string
  eligible: boolean
  historical_count: number
  held_count: number
  unknown_holding_count: number
  obtained: boolean
  record_ids: number[]
}
export interface DexResult {
  data: DexTarget[]
  meta: { total: number; page: number; page_size: number }
  filters: {
    dimension: string
    basis: string
    scope: string
    collection_system: string
    display: string
  }
  summary: {
    known_targets: number
    eligible_targets: number
    obtained_eligible: number
    missing_eligible: number
    observed_targets: number
    observed_outside_denominator: number
    percentage: number | null
    denominator_rule: string
  }
  coverage: {
    status: string
    species: number
    forms: number
    included_forms: number
    unknown_forms: number
    selected_games: number
    unclassified_games: number
    verified_availability: number
    unknown_availability_targets: number
    pending_chinese_names: number
    revisions: string[]
    availability_versions: string[]
    note: string
  }
}
const oneOf = (v: unknown, values: string[]) => typeof v === 'string' && values.includes(v)
const counts = (v: Record<string, unknown>, fields: string[]) =>
  fields.every((key) => integer(v[key]))
function isTarget(v: unknown): v is DexTarget {
  return (
    record(v) &&
    id(v.id) &&
    id(v.species_id) &&
    (v.form_id === null || id(v.form_id)) &&
    id(v.national_dex) &&
    v.name !== '' &&
    v.slug !== '' &&
    ['slug', 'name', 'inclusion_reason', 'source_url', 'cross_check_url', 'revision'].every(
      (k) => typeof v[k] === 'string',
    ) &&
    record(v.names) &&
    Object.values(v.names).every((n) => typeof n === 'string') &&
    oneOf(v.name_status, ['unknown', 'source_provided', 'verified', 'conflict']) &&
    oneOf(v.cross_check_status, ['unknown', 'verified', 'conflict']) &&
    oneOf(v.inclusion, ['included', 'excluded', 'unknown']) &&
    oneOf(v.availability, [
      'confirmed_available',
      'confirmed_locked',
      'transfer_only',
      'unknown',
    ]) &&
    typeof v.eligible === 'boolean' &&
    typeof v.obtained === 'boolean' &&
    counts(v, ['historical_count', 'held_count', 'unknown_holding_count']) &&
    Array.isArray(v.record_ids) &&
    v.record_ids.every(id) &&
    new Set(v.record_ids).size === v.record_ids.length
  )
}
export function decodeDex(body: unknown): DexResult {
  if (
    !record(body) ||
    !Array.isArray(body.data) ||
    !body.data.every(isTarget) ||
    !record(body.meta) ||
    !counts(body.meta, ['total', 'page', 'page_size']) ||
    !integer(body.meta.total) ||
    !id(body.meta.page) ||
    !id(body.meta.page_size) ||
    body.meta.page_size > 100 ||
    body.data.length > body.meta.page_size ||
    body.data.length > body.meta.total ||
    !record(body.filters) ||
    !oneOf(body.filters.dimension, ['species', 'form']) ||
    !oneOf(body.filters.basis, ['historical', 'current']) ||
    !oneOf(body.filters.scope, ['all', 'mainline', 'game']) ||
    typeof body.filters.collection_system !== 'string' ||
    !oneOf(body.filters.display, ['all', 'legal']) ||
    !record(body.summary) ||
    !record(body.coverage)
  )
    return invalidResponse()
  const s = body.summary,
    c = body.coverage
  if (
    !counts(s, [
      'known_targets',
      'eligible_targets',
      'obtained_eligible',
      'missing_eligible',
      'observed_targets',
      'observed_outside_denominator',
    ]) ||
    typeof s.denominator_rule !== 'string' ||
    typeof c.status !== 'string' ||
    typeof c.note !== 'string' ||
    !counts(c, [
      'species',
      'forms',
      'included_forms',
      'unknown_forms',
      'selected_games',
      'unclassified_games',
      'verified_availability',
      'unknown_availability_targets',
      'pending_chinese_names',
    ]) ||
    !Array.isArray(c.revisions) ||
    !c.revisions.every((v) => typeof v === 'string') ||
    !Array.isArray(c.availability_versions) ||
    !c.availability_versions.every((v) => typeof v === 'string')
  )
    return invalidResponse()
  const eligible = s.eligible_targets as number,
    obtained = s.obtained_eligible as number
  if (
    obtained > eligible ||
    eligible > (s.known_targets as number) ||
    s.missing_eligible !== eligible - obtained ||
    (s.observed_outside_denominator as number) + obtained !== s.observed_targets ||
    body.meta.total > (s.known_targets as number) ||
    (s.observed_targets as number) > (s.known_targets as number) ||
    (s.observed_outside_denominator as number) > (s.observed_targets as number) ||
    (eligible === 0
      ? s.percentage !== null
      : typeof s.percentage !== 'number' ||
        !Number.isFinite(s.percentage) ||
        Math.abs(s.percentage - (100 * obtained) / eligible) > 1e-6)
  )
    return invalidResponse()
  for (const row of body.data) {
    const n = body.filters.basis === 'current' ? row.held_count : row.historical_count
    const observed = n > 0
    if (
      row.obtained !== observed ||
      row.record_ids.length !== n ||
      (body.filters.dimension === 'species'
        ? row.form_id !== null || row.id !== row.species_id
        : row.form_id !== row.id) ||
      (row.eligible &&
        (eligible === 0 ||
          row.availability !== 'confirmed_available' ||
          row.inclusion !== 'included')) ||
      (row.eligible && observed && obtained === 0)
    )
      return invalidResponse()
  }
  return body as unknown as DexResult
}
export const dex = {
  async list(query: string, options?: RequestOptions): Promise<DexResult> {
    return decodeDex(await api.getJSON(`/dex${query ? `?${query}` : ''}`, options))
  },
}
export function sourceLink(value: string): string | undefined {
  try {
    const url = new URL(value)
    return ['https:', 'http:'].includes(url.protocol) && !url.username && !url.password
      ? url.href
      : undefined
  } catch {
    return undefined
  }
}
