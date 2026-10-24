import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { tracker } from '../api/tracker'
import { ApiError } from '../api/client'
import { catalog, hunt } from '../test/fixtures'
import HuntForm from './HuntForm.vue'
const ReferenceSelect = defineComponent({
  props: ['modelValue', 'items', 'label'],
  emits: ['update:modelValue'],
  template: `<select :aria-label="label" :value="modelValue" @change="$emit('update:modelValue', Number($event.target.value) || null)"><option value="">未记录</option><option v-for="item in items" :value="item.id">{{item.name}}</option></select>`,
})
afterEach(() => vi.restoreAllMocks())
function mountForm(id?: number) {
  vi.spyOn(tracker, 'catalog').mockResolvedValue(catalog)
  return mount(HuntForm, { props: { id }, global: { stubs: { ReferenceSelect } } })
}
describe('Hunt editor', () => {
  it('does not submit invalid input and preserves entered values', async () => {
    const create = vi.spyOn(tracker, 'createHunt'),
      wrapper = mountForm()
    await flushPromises()
    await wrapper.get('[aria-label="累计次数"]').setValue('-1')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(create).not.toHaveBeenCalled()
    await vi.waitFor(() => expect(wrapper.text()).toContain('请选择目标物种'))
    expect((wrapper.get('[aria-label="累计次数"]').element as HTMLInputElement).value).toBe('-1')
    wrapper.unmount()
  })
  it('retains nullable fields and loaded form identity during complete editing', async () => {
    vi.spyOn(tracker, 'hunt').mockResolvedValue(hunt)
    const update = vi
      .spyOn(tracker, 'updateHunt')
      .mockRejectedValue(new ApiError('http', '输入无效', 400, 'invalid_request'))
    const wrapper = mountForm(1)
    await flushPromises()
    await wrapper.get('[aria-label="任务备注"]').setValue('新的备注')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(update).toHaveBeenCalledWith(
      1,
      expect.objectContaining({
        form_id: 1,
        encounter_count: 0,
        game_id: null,
        start_date: '2024',
        duration_seconds: null,
        notes: '新的备注',
      }),
      hunt.updated_at,
    )
    expect(wrapper.text()).toContain('输入无效')
    wrapper.unmount()
  })
  it('blocks repeat writes after an ambiguous outcome instead of recreating records', async () => {
    const create = vi
      .spyOn(tracker, 'createHunt')
      .mockRejectedValue(new ApiError('network', '无法连接'))
    const wrapper = mountForm()
    await flushPromises()
    await wrapper.get('[aria-label="目标物种"]').setValue('1')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.text()).toContain('保存结果暂未确认')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(create).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('检查任务列表')
    wrapper.unmount()
  })
})
