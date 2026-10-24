import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, ApiError } from '../api/client'
import ServiceStatus from './ServiceStatus.vue'

afterEach(() => vi.restoreAllMocks())

describe('local service status', () => {
  it('shows loading, success and a usable refresh action', async () => {
    let finish!: (value: { status: 'ok' }) => void
    const health = vi.spyOn(api, 'getHealth').mockImplementationOnce(() => new Promise((resolve) => { finish = resolve }))
      .mockResolvedValue({ status: 'ok' })
    const wrapper = mount(ServiceStatus)
    expect(wrapper.get('[role="status"]').text()).toBe('正在检查')
    expect(wrapper.get('button').classes()).toContain('is-loading')
    finish({ status: 'ok' })
    await flushPromises()
    expect(wrapper.get('[role="status"]').text()).toBe('连接正常')
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(health).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it.each(['http', 'network', 'timeout', 'invalid_response'] as const)('shows %s failure and recovers on retry', async (kind) => {
    vi.spyOn(api, 'getHealth').mockRejectedValueOnce(new ApiError(kind, '测试错误提示'))
      .mockResolvedValue({ status: 'ok' })
    const wrapper = mount(ServiceStatus)
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('测试错误提示')
    expect(wrapper.get('button').text()).toBe('重试连接')
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(wrapper.get('[role="status"]').text()).toBe('连接正常')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('cancels an in-flight request when the component is removed', () => {
    let signal: AbortSignal | undefined
    vi.spyOn(api, 'getHealth').mockImplementation((options) => {
      signal = options?.signal
      return new Promise(() => {})
    })
    const wrapper = mount(ServiceStatus)
    expect(signal?.aborted).toBe(false)
    wrapper.unmount()
    expect(signal?.aborted).toBe(true)
  })
})
