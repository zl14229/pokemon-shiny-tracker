<script setup lang="ts">
import { ElOption, ElSelect } from 'element-plus'
import type { Reference } from '../api/models'
import { referenceOption } from '../domain/format'
const props = defineProps<{
  modelValue: number | null
  items: Reference[]
  label: string
  disabled?: boolean
  required?: boolean
  placeholder?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: number | null] }>()
function update(value: number | undefined | null) {
  emit('update:modelValue', value ?? null)
}
</script>
<template>
  <ElSelect
    :model-value="modelValue"
    :aria-label="label"
    :disabled="disabled"
    filterable
    clearable
    :aria-required="required"
    :placeholder="placeholder ?? (required ? '请选择物种' : '未记录（可留空）')"
    @update:model-value="update"
  >
    <ElOption
      v-if="modelValue !== null && !props.items.some((item) => item.id === modelValue)"
      :value="modelValue"
      :label="`记录 #${modelValue}（资料暂不可用）`"
    />
    <ElOption
      v-for="item in items"
      :key="item.id"
      :value="item.id"
      :label="referenceOption(item)"
    />
  </ElSelect>
</template>
