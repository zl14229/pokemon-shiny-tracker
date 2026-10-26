import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, expect, it, vi } from 'vitest'
import { dex, decodeDex } from '../api/dex'
import { tracker } from '../api/tracker'
import { ApiError } from '../api/client'
import { dexFixture } from '../test/dex-fixture'
import DexPage from './DexPage.vue'
afterEach(() => vi.restoreAllMocks())
it('shows unknown denominator and source gaps without claiming completion', async () => {
  vi.spyOn(dex, 'list').mockResolvedValue(dexFixture())
  vi.spyOn(tracker, 'games').mockResolvedValue([])
  const wrapper = mount(DexPage)
  await flushPromises()
  expect(wrapper.text()).toContain('分母待核实')
  expect(wrapper.text()).toContain('中文名缺失')
  expect(wrapper.text()).not.toContain('100.0%')
  wrapper.unmount()
})
it('clears old targets on a failed filter read, then retries local data', async () => {
  const list = vi
    .spyOn(dex, 'list')
    .mockResolvedValueOnce(dexFixture())
    .mockRejectedValueOnce(new ApiError('network', '无法连接本地服务'))
    .mockResolvedValue(dexFixture())
  vi.spyOn(tracker, 'games').mockResolvedValue([])
  const wrapper = mount(DexPage, { props: { query: '' } })
  await flushPromises()
  expect(wrapper.find('article').exists()).toBe(true)
  await wrapper.setProps({ query: 'basis=current' })
  await flushPromises()
  expect(wrapper.find('article').exists()).toBe(false)
  expect(wrapper.text()).toContain('无法连接本地服务')
  const retry = wrapper.findAll('button').find((b) => b.text() === '重试')!
  await retry.trigger('click')
  await flushPromises()
  expect(list).toHaveBeenCalledTimes(3)
  expect(wrapper.find('article').exists()).toBe(true)
  wrapper.unmount()
})

it('hides previous targets immediately while a filter hash navigation is pending', async () => {
  vi.spyOn(dex, 'list').mockResolvedValue(dexFixture())
  vi.spyOn(tracker, 'games').mockResolvedValue([])
  const wrapper = mount(DexPage, { props: { query: '' } })
  await flushPromises()
  await wrapper.get('form').trigger('submit')
  expect(wrapper.find('article').exists()).toBe(false)
  expect(wrapper.text()).toContain('正在读取本地图鉴')
  await wrapper.setProps({ query: 'dimension=species' })
  await flushPromises()
  expect(wrapper.find('article').exists()).toBe(true)
  wrapper.unmount()
})

it('shows a proven subset percentage only when a valid numerator and denominator exist', async () => {
  const body = dexFixture()
  Object.assign(body.summary, {
    eligible_targets: 1,
    obtained_eligible: 1,
    observed_targets: 1,
    percentage: 100,
  })
  Object.assign(body.data[0]!, {
    eligible: true,
    availability: 'confirmed_available',
    historical_count: 1,
    obtained: true,
    record_ids: [7],
  })
  Object.assign(body.coverage, {
    selected_games: 1,
    verified_availability: 1,
    unknown_availability_targets: 0,
    availability_versions: ['synthetic-fixture'],
  })
  vi.spyOn(dex, 'list').mockResolvedValue(decodeDex(body))
  vi.spyOn(tracker, 'games').mockResolvedValue([])
  const wrapper = mount(DexPage)
  await flushPromises()
  expect(wrapper.text()).toContain('100.0%')
  expect(wrapper.text()).toContain('1 / 1 个已核实目标')
  expect(wrapper.text()).not.toContain('分母待核实')
  wrapper.unmount()
})
