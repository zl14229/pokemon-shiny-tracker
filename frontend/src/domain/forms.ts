import type { Hunt, HuntInput, HuntStatus } from '../api/models'
export type Errors = Record<string, string>
export const optionalText = (v: string) => v.trim() || null
export function dateBounds(v: string): [number, number] | null {
  if (!/^\d{4}(-\d{2}(-\d{2})?)?$/.test(v)) return null
  const [y, m = 1, d = 1] = v.split('-').map(Number) as [number, number?, number?]
  if (y < 1 || y > 9999 || m < 1 || m > 12) return null
  const days = [
    31,
    y % 4 === 0 && (y % 100 !== 0 || y % 400 === 0) ? 29 : 28,
    31,
    30,
    31,
    30,
    31,
    31,
    30,
    31,
    30,
    31,
  ]
  if (d < 1 || d > days[m - 1]!) return null
  return [
    y * 10000 + m * 100 + d,
    v.length === 4
      ? y * 10000 + 1231
      : v.length === 7
        ? y * 10000 + m * 100 + days[m - 1]!
        : y * 10000 + m * 100 + d,
  ]
}
export function optionalInteger(
  v: string,
  name: string,
  errors: Errors,
  min = 0,
  max = Number.MAX_SAFE_INTEGER,
): number | null {
  if (!v.trim()) return null
  const n = Number(v)
  if (!/^\d+$/.test(v.trim()) || !Number.isSafeInteger(n) || n < min || n > max) {
    errors[name] = `请输入 ${min} 至 ${max} 的整数，未知可留空。`
    return null
  }
  return n
}
export function validateDate(value: string, key: string, errors: Errors) {
  if (value && !dateBounds(value))
    errors[key] = '请输入有效的年、年月或日期，例如 2024、2024-06 或 2024-06-15。'
}
export function textLimit(value: string, key: string, max: number, errors: Errors) {
  if (Array.from(value.trim()).length > max) errors[key] = `最多 ${max} 个字符。`
}
export interface HuntDraft {
  species_id: number | null
  form_id: number | null
  game_id: number | null
  method: string
  start_date: string
  end_date: string
  encounter_count: string
  count_unit: string
  duration_seconds: string
  status: HuntStatus
  notes: string
}
export function huntDraft(h?: Hunt): HuntDraft {
  return {
    species_id: h?.species_id ?? null,
    form_id: h?.form_id ?? null,
    game_id: h?.game_id ?? null,
    method: h?.method ?? '',
    start_date: h?.start_date ?? '',
    end_date: h?.end_date ?? '',
    encounter_count: h?.encounter_count == null ? '' : String(h.encounter_count),
    count_unit: h?.count_unit ?? '',
    duration_seconds: h?.duration_seconds == null ? '' : String(h.duration_seconds),
    status: h?.status ?? 'in_progress',
    notes: h?.notes ?? '',
  }
}
export function validateHunt(d: HuntDraft): { errors: Errors; input: HuntInput | null } {
  const errors: Errors = {}
  if (!d.species_id || !Number.isSafeInteger(d.species_id)) errors.species_id = '请选择目标物种。'
  const count = optionalInteger(d.encounter_count, 'encounter_count', errors)
  const seconds = optionalInteger(d.duration_seconds, 'duration_seconds', errors)
  if (d.encounter_count.trim() && !d.count_unit.trim())
    errors.count_unit = '已知次数必须填写计数单位，例如 eggs 或 encounters。'
  validateDate(d.start_date.trim(), 'start_date', errors)
  validateDate(d.end_date.trim(), 'end_date', errors)
  const start = dateBounds(d.start_date.trim()),
    end = dateBounds(d.end_date.trim())
  if (start && end && start[0] > end[1]) errors.end_date = '结束日期不能早于开始日期。'
  textLimit(d.method, 'method', 128, errors)
  textLimit(d.count_unit, 'count_unit', 32, errors)
  textLimit(d.notes, 'notes', 5000, errors)
  if (Object.keys(errors).length) return { errors, input: null }
  return {
    errors,
    input: {
      species_id: d.species_id!,
      form_id: d.form_id,
      game_id: d.game_id,
      method: optionalText(d.method),
      start_date: optionalText(d.start_date),
      end_date: optionalText(d.end_date),
      encounter_count: count,
      count_unit: optionalText(d.count_unit),
      duration_seconds: seconds,
      status: d.status,
      notes: optionalText(d.notes),
    },
  }
}
