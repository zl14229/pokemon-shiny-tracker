import { mount, flushPromises } from '@vue/test-utils'
import { afterEach, it, expect, vi } from 'vitest'
import DashboardPage from './DashboardPage.vue'
import { dashboard } from '../api/dashboard'
import { tracker } from '../api/tracker'
import { dashboardFixture } from '../test/dashboard-fixture'
afterEach(() => vi.restoreAllMocks())
it('shows empty collection separately from unknown denominator', async () => {
  vi.spyOn(tracker, 'games').mockResolvedValue([])
  vi.spyOn(dashboard, 'get').mockResolvedValue(dashboardFixture())
  const w = mount(DashboardPage)
  expect(w.text()).toContain('正在读取')
  await flushPromises()
  expect(w.text()).toContain('还没有个人记录')
  expect(w.text()).toContain('分母待核实')
  expect(w.text()).not.toContain('0.0%')
  w.unmount()
})
it('clears stale totals on error and supports retry', async () => {
  vi.spyOn(tracker, 'games').mockResolvedValue([])
  const get = vi
    .spyOn(dashboard, 'get')
    .mockResolvedValueOnce(dashboardFixture())
    .mockRejectedValueOnce(new Error('本地连接失败'))
    .mockResolvedValue(dashboardFixture())
  const w = mount(DashboardPage)
  await flushPromises()
  await w.get('button').trigger('click')
  await flushPromises()
  expect(w.text()).toContain('本地连接失败')
  expect(w.find('[data-testid="historical-individuals"]').exists()).toBe(false)
  await w.findAll('button').at(-1)!.trigger('click')
  await flushPromises()
  expect(get).toHaveBeenCalledTimes(3)
  expect(w.find('[data-testid="historical-individuals"]').exists()).toBe(true)
  w.unmount()
})
