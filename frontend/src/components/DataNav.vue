<script setup lang="ts">
defineProps<{ active: 'import' | 'backup' }>()
</script>
<template>
  <nav aria-label="数据管理导航" class="data-nav">
    <a href="#/data/import" :aria-current="active === 'import' ? 'page' : undefined">Excel 导入</a
    ><a href="#/data/backup" :aria-current="active === 'backup' ? 'page' : undefined">导出与备份</a>
  </nav>
</template>
<style scoped>
.data-nav {
  display: flex;
  gap: 20px;
  margin-bottom: 20px;
}
.data-nav a[aria-current='page'] {
  font-weight: 700;
  color: #6577be;
}
</style>
