import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import ShinyForm from './ShinyForm.vue'
import { tracker } from '../api/tracker'
import { ApiError } from '../api/client'
import { catalog, shiny } from '../test/fixtures'
const ReferenceSelect = defineComponent({
  props: ['modelValue', 'items', 'label'],
  emits: ['update:modelValue'],
  template: `<select :aria-label="label" :value="modelValue" @change="$emit('update:modelValue',Number($event.target.value)||null)"><option value="">未记录</option><option v-for="i in items" :value="i.id">{{i.name}}</option></select>`,
})
const HuntPicker = defineComponent({
  props: ['modelValue'],
  emits: ['update:modelValue'],
  template: `<input aria-label="关联 Hunt" :value="modelValue" @input="$emit('update:modelValue',Number($event.target.value)||null)"/>`,
})
afterEach(() => vi.restoreAllMocks())
function setup(id?: number) {
  vi.spyOn(tracker, 'catalog').mockResolvedValue(catalog)
  return mount(ShinyForm, { props: { id }, global: { stubs: { ReferenceSelect, HuntPicker } } })
}
describe('individual editor', () => {
  it('creates an independent record and sends nullable facts unchanged', async () => {
    const create = vi
      .spyOn(tracker, 'createShiny')
      .mockRejectedValue(new ApiError('http', '请检查输入', 400, 'invalid_request'))
    const w = setup()
    await flushPromises()
    await w.get('[aria-label="当前物种"]').setValue('1')
    await w.get('[aria-label="球种"]').setValue('精灵球')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(create).toHaveBeenCalledWith(
      expect.objectContaining({
        hunt_id: null,
        current_species_id: 1,
        original_species_id: null,
        source: 'unknown',
        holding_status: 'unknown',
        original_game_id: null,
        ball: '精灵球',
        level: null,
      }),
    )
    w.unmount()
  })
  it('preserves all known details on PUT and keeps draft after a conflict', async () => {
    const original = {
      ...shiny,
      ball: '高级球',
      nature: '胆小',
      level: 50,
      original_game_id: 1,
      obtained_date: '2018',
    }
    vi.spyOn(tracker, 'shiny').mockResolvedValue(original)
    const update = vi
      .spyOn(tracker, 'updateShiny')
      .mockRejectedValue(new ApiError('http', '任务关联冲突', 409, 'conflict'))
    const w = setup(1)
    await flushPromises()
    await w.get('[aria-label="个体备注"]').setValue('编辑草稿')
    await w.get('[aria-label="关联 Hunt"]').setValue('')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(update).toHaveBeenCalledWith(
      1,
      expect.objectContaining({
        ball: '高级球',
        nature: '胆小',
        level: 50,
        original_game_id: 1,
        obtained_date: '2018',
        hunt_id: null,
        notes: '编辑草稿',
      }),
      original.updated_at,
    )
    expect(w.text()).toContain('无法移走已完成任务的最后有效成果')
    expect((w.get('[aria-label="个体备注"]').element as HTMLTextAreaElement).value).toBe('编辑草稿')
    w.unmount()
  })
  it('does not write invalid levels or silently guess missing current species', async () => {
    const create = vi.spyOn(tracker, 'createShiny'),
      w = setup()
    await flushPromises()
    await w.get('[aria-label="等级"]').setValue('0')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(create).not.toHaveBeenCalled()
    expect(w.text()).toContain('请检查表单中的输入')
    w.unmount()
  })
})
