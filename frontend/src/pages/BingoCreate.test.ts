import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, it, expect, vi } from 'vitest'
import BingoCreate from './BingoCreate.vue'
import { planner } from '../api/planner'
import { tracker } from '../api/tracker'
import { catalog } from '../test/fixtures'
import { bingoFixture } from '../test/bingo-fixture'
afterEach(() => vi.restoreAllMocks())
describe('Bingo creation', () => {
  it('reports insufficient unique targets instead of creating a partial 5x5', async () => {
    vi.spyOn(tracker, 'games').mockResolvedValue(catalog.games)
    const p = bingoFixture()
    p.coverage.requested = 25
    p.coverage.actual = 12
    p.coverage.shortage = true
    vi.spyOn(planner, 'pool').mockResolvedValue({
      rules: p.rules,
      coverage: p.coverage,
      candidates: [],
    })
    const create = vi.spyOn(planner, 'bingo')
    const w = mount(BingoCreate)
    await flushPromises()
    await w.get('fieldset input').setValue(true)
    await w.get('[aria-label="卡片尺寸"]').setValue('5')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(w.text()).toContain('需要 25 个唯一目标，当前只有 12 个')
    expect(create).not.toHaveBeenCalled()
    w.unmount()
  })
  it('creates a persistent card only after an explicit action with the stored calendar zone', async () => {
    vi.spyOn(tracker, 'games').mockResolvedValue(catalog.games)
    const p = bingoFixture()
    vi.spyOn(planner, 'pool').mockResolvedValue({
      rules: p.rules,
      coverage: p.coverage,
      candidates: [],
    })
    const create = vi.spyOn(planner, 'bingo').mockResolvedValue(p)
    const w = mount(BingoCreate)
    await flushPromises()
    expect(create).not.toHaveBeenCalled()
    await w.get('fieldset input').setValue(true)
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(create).toHaveBeenCalledWith(
      expect.objectContaining({
        size: 3,
        calendar_zone: expect.any(String),
        request_id: expect.any(String),
      }),
    )
    expect(location.hash).toBe('#/planner/plans/7')
    w.unmount()
  })
})
