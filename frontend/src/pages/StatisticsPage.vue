<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, reactive, ref } from 'vue'
import { ElButton } from 'element-plus'
import StatisticsChart from '../components/StatisticsChart.vue'
import Feedback from '../components/Feedback.vue'
import { statistics, type StatisticsResult } from '../api/statistics'
import { tracker } from '../api/tracker'
import { type Game } from '../api/models'
import { referenceOption } from '../domain/format'
import { statisticsCharts } from '../domain/statistics-charts'
const filters = reactive({
  collection_system: 'pokemon',
  game_id: '',
  method: '',
  source: '',
  status: '',
  from: '',
  to: '',
  count_unit: '',
  min_count: '',
  max_count: '',
  min_duration: '',
  max_duration: '',
})
const result = ref<StatisticsResult | null>(null),
  games = ref<Game[]>([]),
  loading = ref(true),
  error = ref(''),
  gameError = ref('')
const systems = computed(() => [
  ...new Set(['pokemon', ...games.value.map((g) => g.collection_system)]),
])
const availableGames = computed(() =>
  games.value.filter((g) => g.collection_system === filters.collection_system),
)
const charts = computed(() => (result.value ? statisticsCharts(result.value) : []))
let controller: AbortController | undefined
async function load() {
  controller?.abort()
  const current = new AbortController()
  controller = current
  result.value = null
  error.value = ''
  loading.value = true
  const q = new URLSearchParams()
  for (const [k, v] of Object.entries(filters)) {
    if (v.trim()) q.set(k, v.trim())
  }
  try {
    const v = await statistics.get(q.toString(), { signal: current.signal })
    if (!current.signal.aborted) {
      if ([...q.entries()].some(([k, x]) => v.filters[k] !== x))
        throw new Error('统计响应与筛选不一致，请重试。')
      result.value = v
    }
  } catch (e) {
    if (!current.signal.aborted) error.value = e instanceof Error ? e.message : '统计读取失败。'
  } finally {
    if (!current.signal.aborted) loading.value = false
  }
}
onMounted(() => {
  void load()
  void tracker
    .games()
    .then((v) => (games.value = v))
    .catch(() => (gameError.value = '游戏列表读取失败，暂不可按版本筛选；刷新页面可重试。'))
})
onBeforeUnmount(() => controller?.abort())
const num = (n: number | null) =>
  n === null ? '未知／不可计算' : n.toLocaleString('zh-CN', { maximumFractionDigits: 2 })
</script>
<template>
  <header class="page-heading">
    <div>
      <p class="eyebrow">记录里的轨迹</p>
      <h2>统计中心</h2>
      <p class="muted">只分析本地已记录数据，未知值不替换为零。</p>
    </div>
  </header>
  <form class="surface statistics-filters" @submit.prevent="load">
    <label
      >收藏体系<select
        v-model="filters.collection_system"
        aria-label="统计收藏体系"
        :disabled="loading"
        @change="filters.game_id = ''"
      >
        <option v-for="s in systems" :key="s" :value="s">
          {{ s === 'pokemon' ? '宝可梦（普通收藏）' : s === 'sleep' ? 'Pokémon Sleep（独立）' : s }}
        </option>
      </select></label
    >
    <label
      >原始获得／目标游戏<select
        v-model="filters.game_id"
        aria-label="统计游戏"
        :disabled="loading || !!gameError"
      >
        <option value="">全部游戏（含未知）</option>
        <option v-for="g in availableGames" :key="g.id" :value="String(g.id)">
          {{ referenceOption(g) }}
        </option>
      </select></label
    >
    <label
      >关联Hunt方法<input
        v-model="filters.method"
        aria-label="统计方法"
        maxlength="128"
        placeholder="精确匹配已记录方法"
        :disabled="loading"
    /></label>
    <label
      >成果来源<select v-model="filters.source" aria-label="统计来源" :disabled="loading">
        <option value="">全部来源</option>
        <option value="self_hunted">亲自刷闪</option>
        <option value="trade">交换</option>
        <option value="event">活动</option>
        <option value="other">其他</option>
        <option value="unknown">未知</option>
      </select></label
    >
    <label
      >Hunt状态<select v-model="filters.status" aria-label="统计任务状态" :disabled="loading">
        <option value="">全部状态</option>
        <option value="in_progress">进行中</option>
        <option value="completed">已完成</option>
        <option value="abandoned">已放弃</option>
      </select></label
    >
    <label
      >开始／获得日期从<input
        v-model="filters.from"
        aria-label="统计起始日期"
        type="date"
        :disabled="loading" /></label
    ><label
      >日期至<input v-model="filters.to" aria-label="统计结束日期" type="date" :disabled="loading"
    /></label>
    <label
      >计数单位<input
        v-model="filters.count_unit"
        aria-label="统计计数单位"
        maxlength="32"
        placeholder="次数比较必填原单位"
        :disabled="loading"
    /></label>
    <label
      v-for="(label, key) in {
        min_count: '最少累计次数',
        max_count: '最多累计次数',
        min_duration: '最少累计秒数',
        max_duration: '最多累计秒数',
      }"
      :key="key"
      >{{ label
      }}<input
        v-model="filters[key]"
        :aria-label="label"
        inputmode="numeric"
        pattern="[0-9]+"
        :disabled="loading"
    /></label>
    <ElButton native-type="submit" type="primary" :disabled="loading">应用统计筛选</ElButton>
  </form>
  <p v-if="gameError" class="muted">{{ gameError }}</p>
  <Feedback :message="error" retry @retry="load" />
  <p v-if="loading" role="status">正在汇总本地统计…</p>
  <template v-if="result && !loading">
    <section class="surface statistics-summary" aria-label="统计结果">
      <h3>当前结果</h3>
      <p v-if="new Set(result.effort.map((e) => e.unit)).size > 1 && !result.filters.count_unit">
        当前含不同计数单位，次数图暂不比较；请选择单一计数单位，或在明细中分别查看。
      </p>
      <p data-testid="statistics-totals">
        {{ result.hunts }} 个Hunt · {{ result.individuals }} 只获得个体
      </p>
      <p v-if="!result.hunts && !result.individuals">
        此筛选下暂无记录，可以调整条件。资料库中的目标不会计作个人成果。
      </p>
      <p>
        日期未知：成果 {{ result.coverage.unknown_obtained_date }}、Hunt
        {{ result.coverage.unknown_start_date }}；部分精度日期：成果
        {{ result.coverage.partial_obtained_date }}、Hunt
        {{ result.coverage.partial_start_date }}。无法放入月趋势：成果
        {{ result.coverage.unplaced_records }}、Hunt {{ result.coverage.unplaced_hunts }}。
      </p>
      <p v-if="result.filters.from || result.filters.to">
        日期筛选排除未知日期，部分日期重叠只表示可能在区间内。累计投入无法拆分到子期间，因此效率图表和数值留空。
      </p>
      <details>
        <summary>统计口径</summary>
        <p>{{ result.rule }}</p>
        <p>
          游戏图按成果原始获得游戏；方法图按关联Hunt。来源筛选只作用于成果，Hunt总数仍是任务队列。月份图缺少的月份不代表已完整记录零投入。当前可获取性资料只覆盖M5有限子集，这里不推算真实闪光概率。
        </p>
      </details>
    </section>
    <div class="statistics-grid">
      <article v-for="c in charts" :key="c.title" class="surface">
        <h3>{{ c.title }}</h3>
        <StatisticsChart :title="c.title" :option="c.option" />
      </article>
    </div>
    <section class="surface">
      <h3>投入与效率明细</h3>
      <p>
        合计仅已知值；每个分组独立，不能把单位含义不同的次数相加比较。效率只使用符合条件的已完成Hunt，不代表总体或真实概率。
      </p>
      <div class="statistics-table">
        <table>
          <thead>
            <tr>
              <th>游戏／方法／单位</th>
              <th>次数与覆盖</th>
              <th>耗时与覆盖</th>
              <th>次数／自刷个体</th>
              <th>秒／自刷个体</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="e in result.effort" :key="e.key">
              <td>{{ e.game }}／{{ e.method ?? '方法未知' }}／{{ e.unit ?? '单位未知' }}</td>
              <td>{{ num(e.count) }}（{{ e.known_count_hunts }}/{{ e.hunts }} Hunt）</td>
              <td>{{ num(e.seconds) }} 秒（{{ e.known_duration_hunts }}/{{ e.hunts }} Hunt）</td>
              <td>
                {{ num(e.count_per_shiny) }}<br />{{ e.count_samples }} Hunt／{{ e.count_outcomes }}
                成果
              </td>
              <td>
                {{ num(e.seconds_per_shiny) }}<br />{{ e.time_samples }} Hunt／{{ e.time_outcomes }}
                成果
              </td>
            </tr>
            <tr v-if="!result.effort.length">
              <td colspan="5">暂无匹配的投入记录</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-if="result.effort.some((e) => e.overflow)">
        部分累计值超出安全显示范围，已保留未知，不绘制失真的数值。
      </p>
    </section>
  </template>
</template>
<style scoped>
.statistics-filters {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 20px;
}
.statistics-filters label {
  display: grid;
  gap: 6px;
  font-size: 12px;
  color: #62718a;
}
.statistics-filters input,
.statistics-filters select {
  min-width: 0;
  width: 100%;
  box-sizing: border-box;
  min-height: 40px;
  padding: 8px;
  border: 1px solid #dce1ec;
  border-radius: 8px;
  background: white;
  color: #263443;
  font: inherit;
}
.statistics-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
  margin: 22px 0;
}
.statistics-grid article {
  min-width: 0;
}
.statistics-grid h3 {
  font-size: 15px;
}
.statistics-summary p,
details {
  font-size: 13px;
  overflow-wrap: anywhere;
}
.statistics-table {
  overflow: auto;
}
table {
  border-collapse: collapse;
  width: 100%;
  min-width: 750px;
  font-size: 12px;
}
th,
td {
  text-align: left;
  border-bottom: 1px solid #e4e8ef;
  padding: 12px;
  max-width: 260px;
  overflow-wrap: anywhere;
}
summary {
  cursor: pointer;
}
@media (max-width: 1000px) {
  .statistics-filters {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .statistics-grid {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 450px) {
  .statistics-filters {
    grid-template-columns: 1fr;
  }
}
</style>
