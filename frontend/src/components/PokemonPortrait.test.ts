import { mount } from '@vue/test-utils'
import { expect, it } from 'vitest'
import PokemonPortrait from './PokemonPortrait.vue'
it('provides a named placeholder without requesting an unlicensed image', () => {
  const wrapper = mount(PokemonPortrait, { props: { name: 'Pikachu' } })
  expect(wrapper.find('img').exists()).toBe(false)
  expect(wrapper.get('[role="img"]').attributes('aria-label')).toContain('Pikachu 图片暂不可用')
  wrapper.unmount()
})
it('replaces a failed image and retries only when its source changes', async () => {
  const wrapper = mount(PokemonPortrait, { props: { name: 'Pikachu', src: '/missing.png' } })
  await wrapper.get('img').trigger('error')
  expect(wrapper.find('img').exists()).toBe(false)
  expect(wrapper.text()).toContain('图片暂不可用')
  await wrapper.setProps({ src: '/new-image.png' })
  expect(wrapper.get('img').attributes('src')).toBe('/new-image.png')
  wrapper.unmount()
})
