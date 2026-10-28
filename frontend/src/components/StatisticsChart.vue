<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { init, use, type EChartsType, type EChartsCoreOption } from 'echarts/core'
import { BarChart, LineChart, PieChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent, AriaComponent } from 'echarts/components'
import { SVGRenderer } from 'echarts/renderers'
use([
  BarChart,
  LineChart,
  PieChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  AriaComponent,
  SVGRenderer,
])
const props = defineProps<{ title: string; option: EChartsCoreOption }>()
const container = ref<HTMLDivElement>()
let chart: EChartsType | undefined, observer: ResizeObserver | undefined
function update() {
  chart?.setOption(
    {
      ...props.option,
      animation: false,
      aria: { enabled: true },
      tooltip: { trigger: 'item', renderMode: 'richText', confine: true },
    },
    true,
  )
}
onMounted(() => {
  if (!container.value) return
  chart = init(container.value, undefined, { renderer: 'svg' })
  update()
  observer = new ResizeObserver(() => chart?.resize())
  observer.observe(container.value)
})
watch(() => props.option, update, { deep: true })
onBeforeUnmount(() => {
  observer?.disconnect()
  chart?.dispose()
  chart = undefined
})
</script>
<template><div ref="container" class="statistics-chart" role="img" :aria-label="title" /></template>
<style scoped>
.statistics-chart {
  height: 330px;
  width: 100%;
  min-width: 0;
  overflow: hidden;
}
</style>
