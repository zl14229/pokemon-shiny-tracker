import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiClient } from './client'

afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers() })
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), {
  status, headers: { 'Content-Type': 'application/json; charset=utf-8' },
})

describe('API client', () => {
  it('calls the versioned same-origin endpoint and validates health', async () => {
    const fetcher = vi.fn().mockResolvedValue(json({ data: { status: 'ok' } }))
    vi.stubGlobal('fetch', fetcher)
    await expect(new ApiClient().getHealth()).resolves.toEqual({ status: 'ok' })
    expect(fetcher).toHaveBeenCalledWith('/api/v1/health', expect.objectContaining({
      method: 'GET', cache: 'no-store', headers: { Accept: 'application/json' },
    }))
  })

  it.each([
    [404, 'not_found'], [405, 'method_not_allowed'], [500, 'internal_error'],
  ])('preserves HTTP %s and machine error code, redacts raw messages', async (status, code) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(json({ error: { code, message: 'private-server-detail' } }, status)))
    const error = await new ApiClient().getHealth().catch((failure: unknown) => failure)
    expect(error).toMatchObject({ kind: 'http', status, code })
    expect(String(error)).not.toContain('private-server-detail')
  })

  it('handles a non-JSON proxy error without leaking its body', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>private-detail</html>', { status: 502 })))
    const error = await new ApiClient().getHealth().catch((failure: unknown) => failure)
    expect(error).toMatchObject({ kind: 'http', status: 502, code: null })
    expect(String(error)).not.toContain('private-detail')
  })

  it.each([
    new Response('<html>not the API</html>'),
    new Response('{broken', { headers: { 'Content-Type': 'application/json' } }),
    json({ data: { status: 'unknown' } }), json(null),
  ])('rejects malformed or unexpected success bodies', async (response) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response))
    await expect(new ApiClient().getHealth()).rejects.toMatchObject({ kind: 'invalid_response' })
  })

  it('normalizes network failures', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('private-network-detail')))
    await expect(new ApiClient().getHealth()).rejects.toMatchObject({ kind: 'network', status: null })
  })

  function pendingFetch() {
    vi.stubGlobal('fetch', vi.fn((_url, options: RequestInit) => new Promise((_resolve, reject) => {
      const fail = () => reject(new DOMException('aborted', 'AbortError'))
      if (options.signal?.aborted) fail()
      else options.signal?.addEventListener('abort', fail, { once: true })
    })))
  }

  it('aborts fetch on timeout and cleans up timers', async () => {
    vi.useFakeTimers()
    pendingFetch()
    const result = expect(new ApiClient().getHealth({ timeoutMs: 20 })).rejects.toMatchObject({ kind: 'timeout' })
    await vi.advanceTimersByTimeAsync(20)
    await result
    expect(vi.getTimerCount()).toBe(0)
  })

  it.each([false, true])('handles caller cancellation, already aborted: %s', async (alreadyAborted) => {
    pendingFetch()
    const controller = new AbortController()
    if (alreadyAborted) controller.abort()
    const result = expect(new ApiClient().getHealth({ signal: controller.signal })).rejects.toMatchObject({ kind: 'aborted' })
    controller.abort()
    await result
  })

  it('keeps the timeout active while reading the response body', async () => {
    vi.useFakeTimers()
    vi.stubGlobal('fetch', vi.fn((_url, options: RequestInit) => Promise.resolve({
      ok: true, status: 200, headers: new Headers({ 'Content-Type': 'application/json' }),
      text: () => new Promise((_resolve, reject) => {
        options.signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')), { once: true })
      }),
    })))
    const result = expect(new ApiClient().getHealth({ timeoutMs: 20 })).rejects.toMatchObject({ kind: 'timeout' })
    await vi.advanceTimersByTimeAsync(20)
    await result
  })
})

it('uploads multipart without forcing a JSON content type or stringifying the file', async () => {
  const fetcher = vi.fn().mockResolvedValue(json({data: true}))
  vi.stubGlobal('fetch', fetcher)
  const form = new FormData()
  form.set('file', new Blob(['anonymous']), 'anonymous.xlsx')
  await new ApiClient().postForm('/imports/inspect', form)
  expect(fetcher).toHaveBeenCalledWith('/api/v1/imports/inspect', expect.objectContaining({body: form, headers: {Accept: 'application/json'}}))
})
