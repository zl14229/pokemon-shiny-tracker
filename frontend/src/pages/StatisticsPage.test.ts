import { mount, flushPromises } from '@vue/test-utils'
import { afterEach, it, expect, vi } from 'vitest'
import StatisticsPage from './StatisticsPage.vue'
import { statistics } from '../api/statistics'
import { tracker } from '../api/tracker'
import { statisticsFixture } from '../test/statistics-fixture'
afterEach(() => vi.restoreAllMocks())
const options = { global: { stubs: { StatisticsChart: true } } }
it('shows loading and real empty records, with partial-date semantics', async () => {
  vi.spyOn(tracker, 'games').mockResolvedValue([])
  vi.spyOn(statistics, 'get').mockResolvedValue(statisticsFixture())
  const w = mount(StatisticsPage, options)
  expect(w.text()).toContain('正在汇总')
  await flushPromises()
  expect(w.text()).toContain('此筛选下暂无记录')
  expect(w.text()).toContain('不代表已完整记录零投入')
  w.unmount()
})
it('applies a snapshot of filters and clears old charts after a failure', async () => {
  vi.spyOn(tracker, 'games').mockResolvedValue([])
  const get = vi
    .spyOn(statistics, 'get')
    .mockResolvedValueOnce(statisticsFixture())
    .mockRejectedValueOnce(new Error('连接失败'))
    .mockResolvedValue({
      ...statisticsFixture(),
      filters: { collection_system: 'pokemon', method: 'wild' },
    })
  const w = mount(StatisticsPage, options)
  await flushPromises()
  await w.get('[aria-label="统计方法"]').setValue('wild')
  await w.get('form').trigger('submit')
  await flushPromises()
  expect(w.text()).toContain('连接失败')
  expect(w.find('.statistics-grid').exists()).toBe(false)
  expect(get.mock.calls[1]?.[0]).toContain('method=wild')
  await w.findAll('button').at(-1)!.trigger('click')
  await flushPromises()
  expect(w.find('.statistics-grid').exists()).toBe(true)
  w.unmount()
})
