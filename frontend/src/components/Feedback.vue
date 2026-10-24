<script setup lang="ts">
import { ElAlert, ElButton } from 'element-plus'
defineProps<{ message: string; retry?: boolean }>()
defineEmits<{ retry: [] }>()
</script>
<template>
  <div v-if="message" class="feedback">
    <ElAlert :title="message" type="error" :closable="false" show-icon /><ElButton
      v-if="retry"
      @click="$emit('retry')"
      >重试</ElButton
    >
  </div>
</template>
