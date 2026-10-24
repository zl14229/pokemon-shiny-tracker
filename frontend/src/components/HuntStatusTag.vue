<script setup lang="ts">
import { ElTag } from 'element-plus'
import type { HuntStatus } from '../api/models'
import { statusLabels } from '../domain/format'
defineProps<{ value: HuntStatus }>()
</script>
<template>
  <ElTag
    effect="light"
    :type="value === 'completed' ? 'success' : value === 'abandoned' ? 'info' : 'primary'"
    >{{ statusLabels[value] }}</ElTag
  >
</template>
