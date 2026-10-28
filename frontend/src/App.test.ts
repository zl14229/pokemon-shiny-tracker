import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from './api/client'
import App from './App.vue'

afterEach(() => vi.restoreAllMocks())

describe('application shell', () => {
  it('renders Chinese guidance and usable Element Plus controls', async () => {
    vi.spyOn(api, 'getHealth').mockResolvedValue({ status: 'ok' })
    const wrapper = mount(App, { global: { stubs: { DashboardPage: true } } })
    expect(wrapper.find('h1').text()).toBe('Pokémon Shiny Tracker')
    expect(wrapper.text()).toContain('查看刷闪任务')
    const button = wrapper.get('button')
    expect(button.classes()).toContain('el-button')
    expect(button.attributes('aria-expanded')).toBe('false')
    await button.trigger('click')
    expect(wrapper.get('#usage-guide').text()).toContain('未知日期、次数与游戏可留空')
    expect(button.attributes('aria-expanded')).toBe('true')
    await button.trigger('click')
    expect(wrapper.find('#usage-guide').exists()).toBe(false)
    wrapper.unmount()
  })
})
