export type HuntStatus = 'in_progress' | 'completed' | 'abandoned'
export type Source = 'self_hunted' | 'trade' | 'event' | 'other' | 'unknown'
export type HoldingStatus = 'held' | 'traded' | 'released' | 'unknown'
export interface Base {
  id: number
  created_at: string
  updated_at: string
}
export interface Reference extends Base {
  slug: string
  name: string
  name_zh: string | null
  name_status: 'unknown' | 'source_provided' | 'verified' | 'conflict'
}
export interface Species extends Reference {
  national_dex: number
}
export interface Form extends Reference {
  species_id: number
}
export interface Game extends Reference {
  kind: string
  collection_system: string
}
export interface Catalog {
  species: Species[]
  forms: Form[]
  games: Game[]
}
export interface HuntInput {
  species_id: number
  form_id: number | null
  game_id: number | null
  method: string | null
  start_date: string | null
  end_date: string | null
  encounter_count: number | null
  count_unit: string | null
  duration_seconds: number | null
  status: HuntStatus
  notes: string | null
}
export interface Hunt extends Base, HuntInput {}
export interface ShinyInput {
  hunt_id: number | null
  original_species_id: number | null
  original_form_id: number | null
  current_species_id: number
  current_form_id: number | null
  original_game_id: number | null
  current_game_id: number | null
  current_location: string | null
  obtained_date: string | null
  source: Source
  holding_status: HoldingStatus
  nickname: string | null
  gender: string | null
  nature: string | null
  ability: string | null
  ball: string | null
  level: number | null
  notes: string | null
}
export interface ShinyRecord extends Base, ShinyInput {}
export interface Page<T> {
  data: T[]
  meta: { total: number; page: number; page_size: number }
}
export type Guard<T> = (value: unknown) => value is T
export const record = (v: unknown): v is Record<string, unknown> =>
  typeof v === 'object' && v !== null && !Array.isArray(v)
export const integer = (v: unknown): v is number =>
  typeof v === 'number' && Number.isSafeInteger(v) && v >= 0
export const id = (v: unknown): v is number => integer(v) && v > 0
const str = (v: unknown): v is string => typeof v === 'string'
const nullable = (v: unknown, guard: (value: unknown) => boolean) => v === null || guard(v)
const timestamp = (v: unknown) => str(v) && !Number.isNaN(Date.parse(v))
const base = (v: Record<string, unknown>) =>
  id(v.id) && timestamp(v.created_at) && timestamp(v.updated_at)
export const isReference: Guard<Reference> = (v): v is Reference =>
  record(v) &&
  base(v) &&
  str(v.slug) &&
  str(v.name) &&
  nullable(v.name_zh, str) &&
  ['unknown', 'source_provided', 'verified', 'conflict'].includes(String(v.name_status))
export const isSpecies: Guard<Species> = (v): v is Species =>
  isReference(v) && record(v) && id(v.national_dex)
export const isForm: Guard<Form> = (v): v is Form => isReference(v) && record(v) && id(v.species_id)
export const isGame: Guard<Game> = (v): v is Game =>
  isReference(v) && record(v) && str(v.kind) && str(v.collection_system)
export const isHunt: Guard<Hunt> = (v): v is Hunt =>
  record(v) &&
  base(v) &&
  id(v.species_id) &&
  ['form_id', 'game_id'].every((k) => nullable(v[k], id)) &&
  ['method', 'start_date', 'end_date', 'count_unit', 'notes'].every((k) => nullable(v[k], str)) &&
  ['encounter_count', 'duration_seconds'].every((k) => nullable(v[k], integer)) &&
  ['in_progress', 'completed', 'abandoned'].includes(String(v.status))
export const isShiny: Guard<ShinyRecord> = (v): v is ShinyRecord =>
  record(v) &&
  base(v) &&
  id(v.current_species_id) &&
  [
    'hunt_id',
    'original_species_id',
    'original_form_id',
    'current_form_id',
    'original_game_id',
    'current_game_id',
  ].every((k) => nullable(v[k], id)) &&
  [
    'current_location',
    'obtained_date',
    'nickname',
    'gender',
    'nature',
    'ability',
    'ball',
    'notes',
  ].every((k) => nullable(v[k], str)) &&
  nullable(v.level, (n) => integer(n) && n >= 1 && n <= 100) &&
  ['self_hunted', 'trade', 'event', 'other', 'unknown'].includes(String(v.source)) &&
  ['held', 'traded', 'released', 'unknown'].includes(String(v.holding_status))

export interface ShinyWrite extends ShinyInput {
  change_type?: 'evolution' | 'correction'
  event_date?: string | null
  history_note?: string | null
}
export interface History extends Base {
  shiny_record_id: number
  kind: string
  before_json: string
  after_json: string
  event_date: string | null
  note: string | null
}
export interface HistoryPage {
  data: History[]
  meta: { total: number; limit: number; offset: number; record_deleted: boolean }
}
export const isHistory: Guard<History> = (v): v is History =>
  record(v) &&
  base(v) &&
  id(v.shiny_record_id) &&
  [
    'registered',
    'evolution',
    'form_change',
    'location_change',
    'holding_change',
    'updated',
    'deleted',
  ].includes(String(v.kind)) &&
  [v.before_json, v.after_json].every((raw) => {
    if (typeof raw !== 'string') return false
    try {
      const snapshot: unknown = JSON.parse(raw)
      return snapshot === null || isShiny(snapshot)
    } catch {
      return false
    }
  }) &&
  nullable(v.event_date, str) &&
  nullable(v.note, str)
