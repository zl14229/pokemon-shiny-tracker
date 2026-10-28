import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, it, expect, vi } from 'vitest'
import BingoCard from './BingoCard.vue'
import { ApiError } from '../api/client'
import { planner, parsePlan } from '../api/planner'
import { bingoFixture, completeFixtureCell } from '../test/bingo-fixture'
afterEach(() => vi.restoreAllMocks())
describe('Bingo card interactions', () => {
  it('retries an uncertain reroll with the same identity when the displayed goal is unchanged', async () => {
    const p = bingoFixture()
    vi.spyOn(planner, 'refresh').mockResolvedValue(p)
    const reroll = vi
      .spyOn(planner, 'reroll')
      .mockRejectedValueOnce(new ApiError('network', '未确认'))
      .mockResolvedValueOnce(p)
    const w = mount(BingoCard, { props: { plan: p } })
    await flushPromises()
    const button = w.findAll('button').find((b) => b.text() === '重抽未开始格')!
    await button.trigger('click')
    await flushPromises()
    await button.trigger('click')
    await flushPromises()
    expect(reroll.mock.calls[0]![2]).toEqual(reroll.mock.calls[1]![2])
    w.unmount()
  })
  it('uses a new reroll identity after recovery reveals a different displayed goal', async () => {
    const p = bingoFixture(),
      updated = structuredClone(p)
    const next = updated.targets[0]!
    next.target_key = 'species:99'
    next.species_id = 99
    next.snapshot.key = next.target_key
    next.snapshot.species_id = 99
    next.snapshot.name = '已重新抽取目标'
    for (const g of [...next.snapshot.games, next.snapshot.choice]) {
      g.evidence[0]!.species_id = 99
      g.evidence[0]!.form_id = 99
    }
    parsePlan(updated)
    const read = vi
      .spyOn(planner, 'refresh')
      .mockResolvedValueOnce(p)
      .mockResolvedValueOnce(updated)
    const reroll = vi
      .spyOn(planner, 'reroll')
      .mockRejectedValueOnce(new ApiError('network', '未确认'))
      .mockResolvedValueOnce(updated)
    const w = mount(BingoCard, { props: { plan: p } })
    await flushPromises()
    await w
      .findAll('button')
      .find((b) => b.text() === '重抽未开始格')!
      .trigger('click')
    await flushPromises()
    expect(read).toHaveBeenCalledTimes(2)
    await w.setProps({ plan: updated })
    await w
      .findAll('button')
      .find((b) => b.text() === '重抽未开始格')!
      .trigger('click')
    await flushPromises()
    expect(reroll.mock.calls[1]![2].expected_target_key).toBe('species:99')
    expect(reroll.mock.calls[1]![2].request_id).not.toBe(reroll.mock.calls[0]![2].request_id)
    w.unmount()
  })

  it('refreshes explicitly, keeps completed cells immutable and renders a derived line', async () => {
    const p = bingoFixture()
    for (const t of p.targets.slice(0, 3)) completeFixtureCell(p, t)
    p.bingo!.lines = [[1, 2, 3]]
    vi.spyOn(planner, 'refresh').mockResolvedValue(p)
    const reroll = vi.spyOn(planner, 'reroll')
    const w = mount(BingoCard, { props: { plan: p } })
    await flushPromises()
    expect(w.findAll('article')).toHaveLength(9)
    expect(w.text()).toMatch(/已连线\s*1/)
    expect(
      w
        .findAll('article')[0]!
        .findAll('button')
        .some((b) => b.text().includes('重抽')),
    ).toBe(false)
    expect(reroll).not.toHaveBeenCalled()
    w.unmount()
  })
  it('starts a pending cell with the exact displayed key', async () => {
    const p = bingoFixture()
    vi.spyOn(planner, 'refresh').mockResolvedValue(p)
    const start = vi.spyOn(planner, 'start').mockResolvedValue(p)
    const w = mount(BingoCard, { props: { plan: p } })
    await flushPromises()
    await w
      .findAll('button')
      .find((b) => b.text() === '开始此格')!
      .trigger('click')
    await flushPromises()
    expect(start).toHaveBeenCalledWith(7, 1, undefined, 'species:1')
    w.unmount()
  })
  it('requires explicit timestamp evidence and user confirmation for same-day outcomes', async () => {
    const p = bingoFixture()
    p.bingo!.cells[0] = {
      target_id: 1,
      kind: 'needs_review',
      reviews: [{ record_id: 101, code: 'same_day', reason: '同日需核验', needs_review: true }],
    }
    vi.spyOn(planner, 'refresh').mockResolvedValue(p)
    const verify = vi.spyOn(planner, 'verifyDate').mockResolvedValue(p)
    const w = mount(BingoCard, { props: { plan: p } })
    await flushPromises()
    await w
      .findAll('button')
      .find((b) => b.text() === '补充时间并人工核验')!
      .trigger('click')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(verify).not.toHaveBeenCalled()
    const d = new Date(Date.now() - 1800000)
    const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 19)
    await w.get('[aria-label="具体获得时间"]').setValue(local)
    await w.get('[aria-label="核验依据"]').setValue('synthetic personal timestamp note')
    await w.get('.manual-confirm input').setValue(true)
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(verify).toHaveBeenCalledWith(
      7,
      1,
      expect.objectContaining({
        record_id: 101,
        evidence_note: 'synthetic personal timestamp note',
        confirmed: true,
      }),
    )
    w.unmount()
  })
  it('offers no time-only override for unknown source', async () => {
    const p = bingoFixture()
    p.bingo!.cells[0] = {
      target_id: 1,
      kind: 'needs_review',
      reviews: [{ record_id: 101, code: 'source_missing', reason: '来源不明', needs_review: true }],
    }
    vi.spyOn(planner, 'refresh').mockResolvedValue(p)
    const w = mount(BingoCard, { props: { plan: p } })
    await flushPromises()
    expect(w.text()).toContain('来源不明')
    expect(w.findAll('button').some((b) => b.text() === '补充时间并人工核验')).toBe(false)
    w.unmount()
  })
})
