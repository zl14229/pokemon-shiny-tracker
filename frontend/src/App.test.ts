import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import App from './App.vue'

describe('application shell', () => {
  it('renders Chinese guidance and usable Element Plus controls', async () => {
    const wrapper = mount(App)
    expect(wrapper.find('h1').text()).toBe('Pokémon Shiny Tracker')
    expect(wrapper.text()).toContain('当前尚不能保存个人数据')
    const button = wrapper.get('button')
    expect(button.classes()).toContain('el-button')
    expect(button.attributes('aria-expanded')).toBe('false')
    await button.trigger('click')
    expect(wrapper.get('#usage-guide').text()).toContain('尚不支持录入与保存收藏')
    expect(button.attributes('aria-expanded')).toBe('true')
    await button.trigger('click')
    expect(wrapper.find('#usage-guide').exists()).toBe(false)
    wrapper.unmount()
  })
})
