<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { ElButton, ElOption, ElSelect } from 'element-plus'
import type { Catalog, Hunt } from '../api/models'
import { tracker } from '../api/tracker'
import { errorText } from '../composables/resources'
import { referenceName, statusLabels } from '../domain/format'
import Feedback from './Feedback.vue'
const props = defineProps<{ modelValue: number | null; catalog: Catalog; disabled?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [id: number | null] }>()
const rows = shallowRef<Hunt[]>([]),
  selected = shallowRef<Hunt>(),
  loading = ref(false),
  error = ref(''),
  total = ref(0),
  currentPage = ref(1)
let query = '',
  generation = 0,
  controller: AbortController | undefined,
  alive = true
const selectedController = new AbortController()
async function load(search = query, page = 1) {
  query = search
  const current = ++generation
  controller?.abort()
  controller = new AbortController()
  loading.value = true
  error.value = ''
  try {
    const result = await tracker.hunts(
      new URLSearchParams({
        ...(search.trim() ? { q: search.trim() } : {}),
        page: String(page),
        page_size: '20',
        sort: 'id',
        order: 'desc',
      }).toString(),
      { signal: controller.signal },
    )
    if (alive && current === generation) {
      rows.value = page === 1 ? result.data : [...rows.value, ...result.data]
      total.value = result.meta.total
      currentPage.value = page
    }
  } catch (e) {
    if (alive && current === generation) error.value = errorText(e)
  } finally {
    if (alive && current === generation) loading.value = false
  }
}
async function selection() {
  if (props.modelValue === null) {
    selected.value = undefined
    return
  }
  const id = props.modelValue
  try {
    const h = await tracker.hunt(id, { signal: selectedController.signal })
    if (alive && props.modelValue === id) selected.value = h
  } catch (e) {
    if (alive && props.modelValue === id) error.value = errorText(e)
  }
}
function update(value: number | null | undefined) {
  emit('update:modelValue', value ?? null)
}
function title(h: Hunt) {
  return `#${h.id} · ${referenceName(props.catalog.species, h.species_id)} · ${statusLabels[h.status]}`
}
onMounted(() => {
  void load()
  void selection()
})
watch(() => props.modelValue, selection)
onBeforeUnmount(() => {
  alive = false
  ++generation
  controller?.abort()
  selectedController.abort()
})
</script>
<template>
  <div class="picker">
    <ElSelect
      :model-value="modelValue"
      aria-label="关联 Hunt"
      placeholder="不关联任务（可留空）"
      filterable
      remote
      clearable
      :disabled="disabled"
      :loading="loading"
      :remote-method="load"
      @update:model-value="update"
    >
      <ElOption
        v-if="modelValue !== null && !rows.some((h) => h.id === modelValue)"
        :value="modelValue"
        :label="selected?.id === modelValue ? title(selected) : `任务 #${modelValue}（信息未加载）`"
      />
      <ElOption v-for="h in rows" :key="h.id" :value="h.id" :label="title(h)" />
      <template #footer
        ><ElButton
          v-if="currentPage * 20 < total"
          :loading="loading"
          @click="load(query, currentPage + 1)"
          >加载更多任务</ElButton
        ><span v-else class="field-hint">可搜索物种、方法或备注</span></template
      > </ElSelect
    ><ElButton
      v-if="modelValue !== null"
      text
      type="primary"
      :disabled="disabled"
      @click="update(null)"
      >清空关联</ElButton
    >
    <Feedback :message="error" retry @retry="load()" />
  </div>
</template>
