<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ElAlert, ElButton, ElCard, ElTag } from 'element-plus'
import { api, ApiError } from '../api/client'

const state = ref<'loading' | 'online' | 'error'>('loading')
const message = ref('')
let controller: AbortController | undefined
let generation = 0

async function check() {
  controller?.abort()
  controller = new AbortController()
  const current = ++generation
  state.value = 'loading'
  message.value = ''
  try {
    await api.getHealth({ signal: controller.signal })
    if (current === generation) state.value = 'online'
  } catch (error) {
    if (current !== generation) return
    state.value = 'error'
    message.value = error instanceof ApiError ? error.message : '服务检查失败，请稍后重试。'
  }
}

onMounted(check)
onBeforeUnmount(() => { ++generation; controller?.abort() })
</script>

<template>
  <ElCard class="service-card" shadow="never">
    <div class="service-heading">
      <h2>本地服务</h2>
      <ElTag :type="state === 'online' ? 'success' : state === 'error' ? 'danger' : 'info'" role="status" aria-live="polite">
        {{ state === 'online' ? '连接正常' : state === 'error' ? '连接失败' : '正在检查' }}
      </ElTag>
    </div>
    <p v-if="state === 'online'">本地服务可以响应请求。</p>
    <p v-else-if="state === 'loading'">正在连接本地服务，请稍候。</p>
    <ElAlert v-else :title="message" type="error" :closable="false" show-icon />
    <ElButton :loading="state === 'loading'" @click="check">
      {{ state === 'error' ? '重试连接' : '重新检查' }}
    </ElButton>
  </ElCard>
</template>
