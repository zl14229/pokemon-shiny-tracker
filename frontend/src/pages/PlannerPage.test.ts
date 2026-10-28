import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import PlannerPage from './PlannerPage.vue'
import { planner } from '../api/planner'
import { tracker } from '../api/tracker'
import { catalog } from '../test/fixtures'
import { planFixture } from '../test/planner-fixture'
import { ApiError } from '../api/client'
afterEach(() => vi.restoreAllMocks())
function setup() {
  vi.spyOn(tracker, 'games').mockResolvedValue(catalog.games)
  vi.spyOn(planner, 'list').mockResolvedValue({
    data: [],
    meta: { total: 0, limit: 20, offset: 0 },
  })
  return mount(PlannerPage)
}
describe('Random Hunt page', () => {
  it('requires a game and sends explicit missing/active rules without creating Hunts', async () => {
    const draw = vi.spyOn(planner, 'draw').mockResolvedValue(planFixture())
    const start = vi.spyOn(planner, 'start')
    const w = setup()
    await flushPromises()
    await w.get('form').trigger('submit')
    expect(w.text()).toContain('请至少选择')
    expect(draw).not.toHaveBeenCalled()
    await w.get('fieldset input').setValue(true)
    await w.get('[aria-label="目标数量"]').setValue(2)
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(draw).toHaveBeenCalledWith(
      expect.objectContaining({
        rules: {
          game_ids: [1],
          dimension: 'species',
          pool: 'all_missing',
          count: 2,
          exclude_active: true,
        },
      }),
    )
    expect(w.text()).toContain('可用候选 2 个')
    expect(w.text()).toContain('方法待完善')
    expect(w.findAll('article')).toHaveLength(2)
    expect(start).not.toHaveBeenCalled()
    w.unmount()
  })
  it('reports empty verified coverage and disables saving', async () => {
    const p = planFixture()
    p.targets = []
    Object.assign(p.coverage, {
      verified_targets: 0,
      available: 0,
      actual: 0,
      shortage: true,
    })
    vi.spyOn(planner, 'draw').mockResolvedValue(p)
    const w = setup()
    await flushPromises()
    await w.get('fieldset input').setValue(true)
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(w.text()).toContain('当前没有符合条件的已核实目标')
    expect(w.text()).toContain('不会加入未知')
    expect(w.findAll('article')).toHaveLength(0)
    const save = w.findAll('button').find((b) => b.text() === '保存本次结果')!
    expect(save.attributes('disabled')).toBeDefined()
    w.unmount()
  })
  it('keeps the same request identity after ambiguous generation failure', async () => {
    const draw = vi
      .spyOn(planner, 'draw')
      .mockRejectedValueOnce(new ApiError('network', '网络中断'))
      .mockResolvedValueOnce(planFixture())
    const w = setup()
    await flushPromises()
    await w.get('fieldset input').setValue(true)
    await w.get('form').trigger('submit')
    await flushPromises()
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(draw.mock.calls[0]![0]).toEqual(draw.mock.calls[1]![0])
    w.unmount()
  })
  it('loads a saved plan and creates a Hunt only after an explicit click', async () => {
    const p = planFixture()
    p.status = 'active'
    vi.spyOn(planner, 'get').mockResolvedValue(p)
    const result = structuredClone(p)
    result.targets[0]!.hunt_id = 3
    result.targets[0]!.status = 'started'
    const start = vi.spyOn(planner, 'start').mockResolvedValue(result)
    vi.spyOn(tracker, 'games').mockResolvedValue(catalog.games)
    vi.spyOn(planner, 'list').mockResolvedValue({
      data: [],
      meta: { total: 0, limit: 20, offset: 0 },
    })
    const w = mount(PlannerPage, { props: { id: 1 } })
    await flushPromises()
    expect(start).not.toHaveBeenCalled()
    await w
      .findAll('button')
      .find((b) => b.text() === '转为 Hunt')!
      .trigger('click')
    await flushPromises()
    expect(start).toHaveBeenCalledWith(1, 1)
    expect(w.get('a[href="#/hunts/3"]').text()).toContain('Hunt #3')
    w.unmount()
  })
})
