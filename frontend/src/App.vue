<script setup lang="ts">
import { ref } from 'vue'
import { ElButton, ElCard, ElTag } from 'element-plus'
import ServiceStatus from './components/ServiceStatus.vue'

const showGuide = ref(false)
</script>

<template>
  <main class="app-shell">
    <header class="app-header">
      <span class="brand-mark" aria-hidden="true">✦</span>
      <div>
        <p class="eyebrow">属于你的闪光旅程</p>
        <h1>Pokémon Shiny Tracker</h1>
      </div>
    </header>

    <ElCard class="welcome-card" shadow="never">
      <ElTag type="info" effect="plain">初始化版本</ElTag>
      <h2>每一次相遇，都值得记录</h2>
      <p>在这里整理你的刷闪旅程与闪光收藏。</p>
      <p class="muted">记录与收藏功能正在后续阶段开发，当前尚不能保存个人数据。</p>
      <ElButton
        type="primary"
        :aria-expanded="showGuide"
        aria-controls="usage-guide"
        @click="showGuide = !showGuide"
      >
        {{ showGuide ? '收起说明' : '查看说明' }}
      </ElButton>
      <p v-if="showGuide" id="usage-guide" class="usage-guide">
        当前版本用于确认应用可以正常打开，尚不支持录入与保存收藏。
      </p>
    </ElCard>

    <ServiceStatus />
    <footer class="app-footer">本地运行 · 简体中文 · 个人收藏</footer>
  </main>
</template>
