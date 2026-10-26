<script setup lang="ts">
import { ref, watch } from 'vue'
const props = defineProps<{ name: string; src?: string }>()
const failed = ref(false)
watch(
  () => props.src,
  () => {
    failed.value = false
  },
)
</script>
<template>
  <div class="pokemon-portrait">
    <img
      v-if="src && !failed"
      :src="src"
      :alt="`${name} 图片`"
      loading="lazy"
      @error="failed = true"
    />
    <div v-else role="img" :aria-label="`${name} 图片暂不可用`" class="portrait-placeholder">
      <span aria-hidden="true" class="portrait-mark" /><small>图片暂不可用</small>
    </div>
  </div>
</template>
<style scoped>
.pokemon-portrait {
  width: 80px;
  height: 80px;
  flex: none;
  border-radius: 18px;
  background: #f0f2f8;
  display: grid;
  place-items: center;
}
.pokemon-portrait img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.portrait-placeholder {
  display: grid;
  place-items: center;
  color: #7a85a5;
}
.portrait-mark {
  display: block;
  width: 28px;
  height: 28px;
  background: #8a97ba;
  clip-path: polygon(50% 0, 62% 38%, 100% 50%, 62% 62%, 50% 100%, 38% 62%, 0 50%, 38% 38%);
}
.portrait-placeholder small {
  font-size: 10px;
  margin-top: 5px;
}
</style>
