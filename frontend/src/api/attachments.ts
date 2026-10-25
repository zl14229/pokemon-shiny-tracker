import { api, type RequestOptions } from './client'
import { record, integer } from './models'
import { invalidResponse } from './tracker'
export interface Attachment {
  id: number
  shiny_record_id: number
  created_at: string
  kind: 'file' | 'link'
  name: string
  media_type: string
  sha256: string | null
  size_bytes: number | null
  url: string | null
}
export function safeURL(v: unknown): v is string {
  if (typeof v !== 'string') return false
  try {
    const u = new URL(v)
    return (
      (u.protocol === 'https:' || u.protocol === 'http:') &&
      !!u.hostname &&
      !u.username &&
      !u.password
    )
  } catch {
    return false
  }
}
export function isAttachment(v: unknown): v is Attachment {
  return (
    record(v) &&
    integer(v.id) &&
    v.id > 0 &&
    integer(v.shiny_record_id) &&
    v.shiny_record_id > 0 &&
    typeof v.name === 'string' &&
    typeof v.created_at === 'string' &&
    typeof v.media_type === 'string' &&
    ((v.kind === 'link' && safeURL(v.url) && v.sha256 === null && v.size_bytes === null) ||
      (v.kind === 'file' &&
        v.url === null &&
        integer(v.size_bytes) &&
        v.size_bytes > 0 &&
        v.size_bytes <= 10 * 1024 * 1024 &&
        typeof v.sha256 === 'string' &&
        /^[0-9a-f]{64}$/.test(v.sha256) &&
        [
          'image/png',
          'image/jpeg',
          'image/gif',
          'application/pdf',
          'text/plain; charset=utf-8',
        ].includes(v.media_type)))
  )
}
function entity(body: unknown): Attachment {
  if (!record(body) || !isAttachment(body.data)) return invalidResponse()
  return body.data
}
export async function listAttachments(id: number, options?: RequestOptions): Promise<Attachment[]> {
  const b = await api.getJSON(`/shiny-records/${id}/attachments`, options)
  if (!record(b) || !Array.isArray(b.data) || !b.data.every(isAttachment) || b.data.length > 100)
    return invalidResponse()
  return b.data
}
export async function uploadAttachment(
  id: number,
  file: File,
  options?: RequestOptions,
): Promise<Attachment> {
  const f = new FormData()
  f.set('file', file)
  return entity(
    await api.postForm(`/shiny-records/${id}/attachments`, f, { timeoutMs: 15000, ...options }),
  )
}
export async function linkAttachment(id: number, name: string, url: string): Promise<Attachment> {
  return entity(await api.postJSON(`/shiny-records/${id}/links`, { name, url }))
}
export async function deleteAttachment(id: number) {
  const b = await api.deleteJSON(`/attachments/${id}`)
  if (!record(b) || b.data !== true) return invalidResponse()
}
