import type { ShinyRecord, ShinyWrite, Source, HoldingStatus } from '../api/models'
import { optionalText, optionalInteger, validateDate, textLimit, type Errors } from './forms'
export interface ShinyDraft {
  hunt_id: number | null
  original_species_id: number | null
  original_form_id: number | null
  current_species_id: number | null
  current_form_id: number | null
  original_game_id: number | null
  current_game_id: number | null
  current_location: string
  obtained_date: string
  source: Source
  holding_status: HoldingStatus
  nickname: string
  gender: string
  nature: string
  ability: string
  ball: string
  level: string
  notes: string
  change_type: '' | 'evolution' | 'correction'
  event_date: string
  history_note: string
}
export function shinyDraft(s?: ShinyRecord): ShinyDraft {
  return {
    hunt_id: s?.hunt_id ?? null,
    original_species_id: s?.original_species_id ?? null,
    original_form_id: s?.original_form_id ?? null,
    current_species_id: s?.current_species_id ?? null,
    current_form_id: s?.current_form_id ?? null,
    original_game_id: s?.original_game_id ?? null,
    current_game_id: s?.current_game_id ?? null,
    current_location: s?.current_location ?? '',
    obtained_date: s?.obtained_date ?? '',
    source: s?.source ?? 'unknown',
    holding_status: s?.holding_status ?? 'unknown',
    nickname: s?.nickname ?? '',
    gender: s?.gender ?? '',
    nature: s?.nature ?? '',
    ability: s?.ability ?? '',
    ball: s?.ball ?? '',
    level: s?.level == null ? '' : String(s.level),
    notes: s?.notes ?? '',
    change_type: '',
    event_date: '',
    history_note: '',
  }
}
export function validateShiny(
  d: ShinyDraft,
  original?: ShinyRecord,
): { errors: Errors; input: ShinyWrite | null } {
  const errors: Errors = {}
  if (!d.current_species_id || !Number.isSafeInteger(d.current_species_id))
    errors.current_species_id = '请选择当前物种。'
  if (d.original_form_id !== null && d.original_species_id === null)
    errors.original_species_id = '填写原始形态时必须明确原始物种。'
  if (d.hunt_id !== null && (!Number.isSafeInteger(d.hunt_id) || d.hunt_id <= 0))
    errors.hunt_id = '请选择有效的现有任务，或清空关联。'
  const level = optionalInteger(d.level, 'level', errors, 1, 100)
  validateDate(d.obtained_date.trim(), 'obtained_date', errors)
  validateDate(d.event_date.trim(), 'event_date', errors)
  for (const [key, max] of [
    ['nickname', 128],
    ['gender', 32],
    ['nature', 64],
    ['ability', 128],
    ['ball', 128],
    ['current_location', 256],
    ['notes', 5000],
    ['history_note', 1000],
  ] as const)
    textLimit(d[key], key, max, errors)
  const speciesChanged = original && original.current_species_id !== d.current_species_id
  if (speciesChanged && !d.change_type)
    errors.change_type = '请选择进化或资料纠错，不会自动推断变化原因。'
  if (Object.keys(errors).length) return { errors, input: null }
  return {
    errors,
    input: {
      hunt_id: d.hunt_id,
      original_species_id: d.original_species_id,
      original_form_id: d.original_form_id,
      current_species_id: d.current_species_id!,
      current_form_id: d.current_form_id,
      original_game_id: d.original_game_id,
      current_game_id: d.current_game_id,
      current_location: optionalText(d.current_location),
      obtained_date: optionalText(d.obtained_date),
      source: d.source,
      holding_status: d.holding_status,
      nickname: optionalText(d.nickname),
      gender: optionalText(d.gender),
      nature: optionalText(d.nature),
      ability: optionalText(d.ability),
      ball: optionalText(d.ball),
      level,
      notes: optionalText(d.notes),
      ...(speciesChanged && d.change_type ? { change_type: d.change_type } : {}),
      event_date: optionalText(d.event_date),
      history_note: optionalText(d.history_note),
    },
  }
}
export const historyLabels: Record<string, string> = {
  registered: '登记个体',
  evolution: '进化（用户记录）',
  form_change: '形态变化',
  location_change: '保存位置变化',
  holding_change: '持有状态变化',
  updated: '资料更新',
  deleted: '撤销登记',
}
