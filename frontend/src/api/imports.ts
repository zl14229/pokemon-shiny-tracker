import { api } from './client'
import { record, integer } from './models'
import type { ShinyInput } from './models'
import { invalidResponse } from './tracker'

export interface Cell {
  value: string
  formula: boolean
  issue?: string
}
export interface ExcelRow {
  number: number
  cells: Record<string, Cell>
}
export interface ExcelSheet {
  name: string
  rows: ExcelRow[]
}
export interface Mapping {
  sheet: string
  header_row: number
  columns: Record<string, string>
  values: Record<string, Record<string, string>>
}
export interface PreviewRow {
  number: number
  input: ShinyInput
  errors: string[]
  warnings: string[]
  candidates: number[]
  raw: Record<string, Cell>
}
export interface Preview {
  token: string
  expires_at: string
  file_sha256: string
  mapping: Mapping
  rows: PreviewRow[]
}
function data(body: unknown): unknown {
  if (!record(body) || !('data' in body)) return invalidResponse()
  return body.data
}
function cell(v: unknown): v is Cell {
  return (
    record(v) &&
    typeof v.value === 'string' &&
    typeof v.formula === 'boolean' &&
    (v.issue === undefined || typeof v.issue === 'string')
  )
}
function row(v: unknown): v is ExcelRow {
  return record(v) && integer(v.number) && record(v.cells) && Object.values(v.cells).every(cell)
}
function upload(file: File, mapping?: Mapping) {
  const form = new FormData()
  form.set('file', file)
  if (mapping) form.set('mapping', JSON.stringify(mapping))
  return form
}
export async function inspectWorkbook(file: File): Promise<ExcelSheet[]> {
  const v = data(await api.postForm('/imports/inspect', upload(file), { timeoutMs: 15000 }))
  if (
    !record(v) ||
    !Array.isArray(v.sheets) ||
    !v.sheets.every(
      (s) => record(s) && typeof s.name === 'string' && Array.isArray(s.rows) && s.rows.every(row),
    )
  )
    return invalidResponse()
  return v.sheets as ExcelSheet[]
}
export async function previewWorkbook(file: File, mapping: Mapping): Promise<Preview> {
  const v = data(
    await api.postForm('/imports/preview', upload(file, mapping), { timeoutMs: 15000 }),
  )
  if (
    !record(v) ||
    typeof v.token !== 'string' ||
    typeof v.expires_at !== 'string' ||
    !Array.isArray(v.rows) ||
    !v.rows.every(
      (r) =>
        record(r) &&
        integer(r.number) &&
        record(r.input) &&
        Array.isArray(r.errors) &&
        r.errors.every((x) => typeof x === 'string') &&
        Array.isArray(r.warnings) &&
        r.warnings.every((x) => typeof x === 'string') &&
        Array.isArray(r.candidates) &&
        r.candidates.every(integer) &&
        record(r.raw) &&
        Object.values(r.raw).every(cell),
    )
  )
    return invalidResponse()
  return v as unknown as Preview
}
export async function discardPreview(token: string) {
  await api.postJSON('/imports/discard', { token })
}
export async function confirmImport(
  token: string,
  rows: number[],
  ack: boolean,
): Promise<{ batch: { id: number }; rows: { row_number: number; shiny_record_id: number }[] }> {
  const v = data(
    await api.postJSON(
      '/imports/confirm',
      { token, rows, acknowledge_warnings: ack },
      { timeoutMs: 15000 },
    ),
  )
  if (
    !record(v) ||
    !record(v.batch) ||
    !integer(v.batch.id) ||
    v.batch.id < 1 ||
    !Array.isArray(v.rows) ||
    !v.rows.every(
      (r) =>
        record(r) &&
        integer(r.row_number) &&
        rows.includes(r.row_number) &&
        integer(r.shiny_record_id) &&
        r.shiny_record_id > 0,
    ) ||
    v.rows.length !== rows.length ||
    new Set(v.rows.map((r) => r.row_number)).size !== rows.length ||
    new Set(v.rows.map((r) => r.shiny_record_id)).size !== rows.length
  )
    return invalidResponse()
  return v as { batch: { id: number }; rows: { row_number: number; shiny_record_id: number }[] }
}

export async function importBatches(): Promise<
  { id: number; created_at: string; sheet: string; selected_json: string }[]
> {
  const v = data(await api.getJSON('/imports/batches'))
  if (
    !Array.isArray(v) ||
    !v.every(
      (b) =>
        record(b) &&
        integer(b.id) &&
        b.id > 0 &&
        typeof b.created_at === 'string' &&
        typeof b.sheet === 'string' &&
        typeof b.selected_json === 'string',
    )
  )
    return invalidResponse()
  return v as { id: number; created_at: string; sheet: string; selected_json: string }[]
}
export async function batchRecords(
  id: number,
): Promise<{ row_number: number; shiny_record_id: number }[]> {
  const v = data(await api.getJSON(`/imports/batches/${id}`))
  if (
    !record(v) ||
    !Array.isArray(v.rows) ||
    !v.rows.every((r) => record(r) && integer(r.row_number) && integer(r.shiny_record_id))
  )
    return invalidResponse()
  return v.rows as { row_number: number; shiny_record_id: number }[]
}
