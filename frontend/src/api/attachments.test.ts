import { afterEach, expect, it, vi } from 'vitest'
import {
  isAttachment,
  safeURL,
  uploadAttachment,
  linkAttachment,
  deleteAttachment,
} from './attachments'
const base = {
  id: 1,
  shiny_record_id: 2,
  name: '匿名.txt',
  created_at: '2026-01-01T00:00:00Z',
  kind: 'file',
  media_type: 'text/plain; charset=utf-8',
  sha256: 'a'.repeat(64),
  size_bytes: 3,
  url: null,
}
const reply = (data: unknown) =>
  new Response(JSON.stringify({ data }), { headers: { 'Content-Type': 'application/json' } })
afterEach(() => vi.unstubAllGlobals())
it('guards metadata and prevents active URL protocols', () => {
  expect(isAttachment(base)).toBe(true)
  expect(
    isAttachment({
      ...base,
      kind: 'link',
      sha256: null,
      size_bytes: null,
      url: 'javascript:alert(1)',
    }),
  ).toBe(false)
  expect(safeURL('https://user:secret@example.com')).toBe(false)
  expect(safeURL('https://example.com/proof')).toBe(true)
})
it('rejects unsafe integers and unsupported file MIME', () => {
  expect(isAttachment({ ...base, size_bytes: 9007199254740992 })).toBe(false)
  expect(isAttachment({ ...base, media_type: 'image/svg+xml' })).toBe(false)
})
it('uploads a real multipart File without JSON encoding', async () => {
  const fetcher = vi.fn().mockResolvedValue(reply(base))
  vi.stubGlobal('fetch', fetcher)
  await uploadAttachment(2, new File(['abc'], '匿名.txt'))
  const init = fetcher.mock.calls[0]![1]
  expect(init.body).toBeInstanceOf(FormData)
  expect(init.headers).toEqual({ Accept: 'application/json' })
})
it('does not accept a malformed link write response', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(reply({ ...base, kind: 'link', url: 'file:///private' })),
  )
  await expect(linkAttachment(2, 'proof', 'https://example.com')).rejects.toMatchObject({
    kind: 'invalid_response',
  })
})
it('validates deletion acknowledgement', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(reply(false)))
  await expect(deleteAttachment(1)).rejects.toMatchObject({ kind: 'invalid_response' })
})
