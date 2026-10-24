import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiClient } from './client'
import { TrackerAPI, huntInput } from './tracker'
import { hunt } from '../test/fixtures'
const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
afterEach(() => vi.unstubAllGlobals())
describe('typed record transport', () => {
  it('sends full JSON with null and zero on create and validates the returned identity', async () => {
    const fetcher = vi.fn().mockResolvedValue(json({ data: hunt }, 201))
    vi.stubGlobal('fetch', fetcher)
    await expect(new TrackerAPI().createHunt(huntInput(hunt))).resolves.toEqual(hunt)
    expect(fetcher).toHaveBeenCalledWith(
      '/api/v1/hunts',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(huntInput(hunt)),
        headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
      }),
    )
  })
  it('checks for a newer record before writing; stale data does not trigger PUT', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(json({ data: { ...hunt, updated_at: '2026-10-24T01:00:00Z' } }))
    vi.stubGlobal('fetch', fetcher)
    await expect(
      new TrackerAPI().updateHunt(1, huntInput(hunt), hunt.updated_at),
    ).rejects.toMatchObject({ code: 'stale_record' })
    expect(fetcher).toHaveBeenCalledTimes(1)
  })
  it('distinguishes failed preflight from an uncertain write so safe reads can be retried', async () => {
    const fetcher = vi.fn().mockRejectedValue(new TypeError('network'))
    vi.stubGlobal('fetch', fetcher)
    await expect(
      new TrackerAPI().updateHunt(1, huntInput(hunt), hunt.updated_at),
    ).rejects.toMatchObject({ name: 'BeforeWriteError' })
    expect(fetcher).toHaveBeenCalledTimes(1)
  })
  it('writes all editable fields after preflight and redacts HTTP conflict messages', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(json({ data: hunt }))
      .mockResolvedValueOnce(json({ error: { code: 'conflict', message: 'private-detail' } }, 409))
    vi.stubGlobal('fetch', fetcher)
    const error = await new TrackerAPI()
      .updateHunt(1, huntInput(hunt), hunt.updated_at)
      .catch((e) => e)
    expect(error).toMatchObject({ status: 409, code: 'conflict' })
    expect(String(error)).not.toContain('private-detail')
    expect(fetcher.mock.calls[1]?.[1]).toMatchObject({
      method: 'PUT',
      body: JSON.stringify(huntInput(hunt)),
    })
  })
  it.each([
    { ...hunt, encounter_count: Number.MAX_SAFE_INTEGER + 1 },
    { ...hunt, notes: undefined },
    { id: 1 },
  ])('rejects incomplete or unsafe successful records', async (row) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(json({ data: row })))
    await expect(new TrackerAPI().hunt(1)).rejects.toMatchObject({ kind: 'invalid_response' })
  })
  it('rejects malformed pagination rather than showing empty success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(json({ data: [], meta: { page: 0, page_size: 20, total: 0 } })),
    )
    await expect(new TrackerAPI(new ApiClient()).hunts()).rejects.toMatchObject({
      kind: 'invalid_response',
    })
  })
})
