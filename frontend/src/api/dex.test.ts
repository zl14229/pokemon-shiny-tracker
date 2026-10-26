import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from './client'
import { decodeDex, dex, sourceLink } from './dex'
import { dexFixture } from '../test/dex-fixture'
afterEach(() => vi.restoreAllMocks())
describe('dex evidence response', () => {
  it('reads local endpoint and preserves unknown completion as null', async () => {
    const body = dexFixture(),
      get = vi.spyOn(api, 'getJSON').mockResolvedValue(body)
    expect((await dex.list('dimension=species')).summary.percentage).toBeNull()
    expect(get).toHaveBeenCalledWith('/dex?dimension=species', undefined)
  })
  it.each([0, 100])('rejects invented zero-denominator percentage %s', (percentage) => {
    const body = dexFixture()
    body.summary.percentage = percentage
    expect(() => decodeDex(body)).toThrow()
  })
  it('rejects a numerator outside its denominator and unsafe IDs', () => {
    const body = dexFixture()
    body.summary.eligible_targets = 1
    body.summary.obtained_eligible = 2
    expect(() => decodeDex(body)).toThrow()
    const inconsistent = dexFixture()
    inconsistent.data[0]!.eligible = true
    expect(() => decodeDex(inconsistent)).toThrow()
    const unsafe = dexFixture()
    unsafe.data[0]!.id = Number.MAX_SAFE_INTEGER + 1
    expect(() => decodeDex(unsafe)).toThrow()
  })
  it('accepts current-game holding evidence with no original-game history and rejects stale obtained flags', () => {
    const body = dexFixture()
    body.filters.basis = 'current'
    body.data[0]!.held_count = 1
    body.data[0]!.obtained = true
    body.data[0]!.record_ids = [7]
    body.summary.observed_targets = 1
    body.summary.observed_outside_denominator = 1
    expect(decodeDex(body).data[0]!.historical_count).toBe(0)
    body.data[0]!.record_ids = []
    expect(() => decodeDex(body)).toThrow()
  })
  it('permits only HTTP source links without embedded credentials', () => {
    expect(sourceLink('https://example.test/source')).toBe('https://example.test/source')
    for (const url of [
      'javascript:alert(1)',
      'file:///private',
      'https://user:password@example.test',
      'bad',
    ])
      expect(sourceLink(url)).toBeUndefined()
  })
})
