<script setup lang="ts">
import { ElAlert } from 'element-plus'
</script>
<template>
  <ElAlert
    class="catalog-notice"
    title="本地资料仅覆盖部分物种与游戏；源中文名待核验，资料存在不代表可合法刷闪。"
    type="info"
    :closable="false"
    show-icon
  />
</template>
