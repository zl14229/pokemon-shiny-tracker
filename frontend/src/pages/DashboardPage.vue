<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { ElButton } from 'element-plus'
import { dashboard, type DashboardResult } from '../api/dashboard'
import { tracker } from '../api/tracker'
import Feedback from '../components/Feedback.vue'
const result = ref<DashboardResult | null>(null),
  loading = ref(true),
  error = ref(''),
  gameError = ref(''),
  system = ref('pokemon'),
  systems = ref(['pokemon'])
let controller: AbortController | undefined
async function loadGames() {
  try {
    const games = await tracker.games()
    systems.value = [...new Set(['pokemon', ...games.map((g) => g.collection_system)])]
    gameError.value = ''
  } catch {
    gameError.value = '游戏列表读取失败，其他收藏体系暂不可选；可刷新总览重试。'
  }
}
async function load() {
  void loadGames()
  controller?.abort()
  const current = new AbortController()
  controller = current
  loading.value = true
  result.value = null
  error.value = ''
  try {
    const v = await dashboard.get(system.value, { signal: current.signal })
    if (!current.signal.aborted) result.value = v
  } catch (e) {
    if (!current.signal.aborted)
      error.value = e instanceof Error ? e.message : '总览读取失败，请重试。'
  } finally {
    if (!current.signal.aborted) loading.value = false
  }
}
onMounted(() => {
  void load()
})
onBeforeUnmount(() => controller?.abort())
const count = (n: number) => n.toLocaleString('zh-CN')
</script>
<template>
  <section aria-label="收藏总览" class="dashboard">
    <div class="dashboard-heading">
      <h3>收藏总览</h3>
      <div>
        <label
          >收藏体系
          <select v-model="system" aria-label="总览收藏体系" :disabled="loading" @change="load">
            <option v-for="s in systems" :key="s" :value="s">
              {{
                s === 'pokemon' ? '宝可梦（普通收藏）' : s === 'sleep' ? 'Pokémon Sleep（独立）' : s
              }}
            </option>
          </select></label
        ><ElButton :disabled="loading" @click="load">刷新总览</ElButton>
      </div>
    </div>
    <p v-if="loading" role="status">正在读取本地收藏总览…</p>
    <Feedback :message="error" retry @retry="load" />
    <p v-if="gameError" class="muted">{{ gameError }}</p>
    <template v-if="result && !loading">
      <p
        v-if="result.collection.historical_individuals === 0 && result.hunts.total === 0"
        class="surface"
      >
        当前体系还没有个人记录。可以先创建刷闪任务，或登记已有收藏；资料库不等于个人收藏。
      </p>
      <div class="dashboard-metrics">
        <article class="surface">
          <span>历史曾获得个体</span
          ><strong data-testid="historical-individuals">{{
            count(result.collection.historical_individuals)
          }}</strong>
          <p>每只永久身份只计一次</p>
        </article>
        <article class="surface">
          <span>当前确认持有个体</span
          ><strong data-testid="held-individuals">{{
            count(result.collection.held_individuals)
          }}</strong>
          <p>持有状态未知 {{ count(result.collection.unknown_holding_individuals) }} 只</p>
        </article>
        <article class="surface">
          <span>亲自刷闪个体</span
          ><strong>{{ count(result.collection.self_hunted_individuals) }}</strong>
          <p>来源未知 {{ count(result.collection.unknown_source_individuals) }} 只</p>
        </article>
      </div>
      <section class="surface dashboard-hunts" aria-label="刷闪任务概况">
        <h3>刷闪任务</h3>
        <p>
          进行中 <b>{{ result.hunts.in_progress }}</b> · 已完成
          <b>{{ result.hunts.completed }}</b> · 已放弃 <b>{{ result.hunts.abandoned }}</b> · 共
          {{ result.hunts.total }} 个
        </p>
        <p class="muted">
          次数未知 {{ result.hunts.unknown_count }} 个；耗时未知
          {{ result.hunts.unknown_duration }} 个；游戏未知
          {{ result.hunts.unknown_game }} 个。未知值不会按 0 计算效率。
        </p>
        <a href="#/hunts">查看刷闪任务 →</a>
      </section>
      <h3>双层图鉴 · 已核实子集</h3>
      <div class="dashboard-dex">
        <article
          v-for="d in result.dex"
          :key="`${d.dimension}/${d.basis}`"
          class="surface"
          :aria-label="`${d.dimension === 'species' ? '物种' : '形态'}${d.basis === 'historical' ? '历史曾获得' : '当前持有'}`"
        >
          <h4>
            {{ d.dimension === 'species' ? '物种' : '持久形态' }} ·
            {{ d.basis === 'historical' ? '历史曾获得' : '当前持有' }}
          </h4>
          <strong>{{
            d.summary.percentage === null ? '分母待核实' : `${d.summary.percentage.toFixed(1)}%`
          }}</strong>
          <p>{{ d.summary.obtained_eligible }} / {{ d.summary.eligible_targets }} 个已核实目标</p>
          <p>
            已记录 {{ d.summary.observed_targets }} 个目标，其中
            {{ d.summary.observed_outside_denominator }} 个在分母外。
          </p>
          <a
            :href="`#/dex?dimension=${d.dimension}&basis=${d.basis}&collection_system=${encodeURIComponent(result.collection_system)}`"
            >查看图鉴与缺失 →</a
          >
        </article>
      </div>
      <details class="surface dashboard-rules">
        <summary>统计口径与资料覆盖</summary>
        <p>{{ result.rule }}</p>
        <p>{{ result.dex[0]?.coverage.note }}</p>
        <p>
          本地物种 {{ result.dex[0]?.coverage.species }}，形态
          {{ result.dex[0]?.coverage.forms }}；获取资料
          {{ result.dex[0]?.coverage.verified_availability }} 条。未知原始获得游戏的个体
          {{ result.collection.unknown_original_game_individuals }} 只，不推断其获得版本。
        </p>
        <p>
          资料版本：{{
            result.dex[0]?.coverage.revisions.join('、') || '尚无资料版本'
          }}；可获取性版本：{{
            result.dex[0]?.coverage.availability_versions.join('、') || '尚无已核实事实'
          }}。仅代表本地子集，不能据此宣称全游戏完成。
        </p>
      </details>
    </template>
  </section>
</template>
<style scoped>
.dashboard {
  margin-bottom: 28px;
}
.dashboard-heading,
.dashboard-heading > div {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-wrap: wrap;
  justify-content: space-between;
}
select {
  max-width: 100%;
  padding: 8px;
  border: 1px solid #dce1ec;
  border-radius: 8px;
  background: white;
  color: #263443;
}
.dashboard-metrics,
.dashboard-dex {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin: 18px 0;
}
.dashboard-dex {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
article {
  min-width: 0;
  overflow-wrap: anywhere;
}
article strong {
  display: block;
  font-size: 28px;
  color: #435da7;
  margin: 12px 0;
}
article p,
.dashboard-rules {
  font-size: 13px;
}
.dashboard-hunts {
  margin: 20px 0;
}
.dashboard-rules {
  overflow-wrap: anywhere;
}
summary {
  cursor: pointer;
}
@media (max-width: 700px) {
  .dashboard-metrics,
  .dashboard-dex {
    grid-template-columns: 1fr;
  }
}
</style>
