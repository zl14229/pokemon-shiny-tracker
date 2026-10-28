<script setup lang="ts">
import { onMounted, ref, shallowRef, onBeforeUnmount } from 'vue'
import { ElButton } from 'element-plus'
import { planner, type PlannerRules, type Coverage } from '../api/planner'
import { tracker } from '../api/tracker'
import type { Game } from '../api/models'
import { errorText } from '../composables/resources'
import { go } from '../domain/navigation'
const games = shallowRef<Game[]>([]),
  coverage = shallowRef<Coverage | null>(null),
  size = ref<3 | 5>(3),
  busy = ref(false),
  loading = ref(true),
  error = ref('')
const rules = ref<PlannerRules>({
  game_ids: [],
  dimension: 'species',
  pool: 'all_missing',
  count: 9,
  exclude_active: true,
})
let alive = true,
  pending: Parameters<typeof planner.bingo>[0] | undefined
const zone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
const gameName = (g: Game) => (g.name_status !== 'conflict' && g.name_zh ? g.name_zh : g.name)
function change() {
  coverage.value = null
  pending = undefined
  error.value = ''
}
async function readPool() {
  const r = { ...rules.value, game_ids: [...rules.value.game_ids], count: size.value * size.value }
  if (!r.game_ids.length) throw new Error('choose game')
  const p = await planner.pool(r)
  if (alive) coverage.value = p.coverage
  return p
}
async function check() {
  if (busy.value) return
  error.value = ''
  if (!rules.value.game_ids.length) {
    error.value = '请至少选择一个游戏版本。'
    return
  }
  busy.value = true
  try {
    await readPool()
  } catch (e) {
    if (alive) error.value = errorText(e)
  } finally {
    if (alive) busy.value = false
  }
}
async function create() {
  if (busy.value) return
  error.value = ''
  if (!rules.value.game_ids.length) {
    error.value = '请至少选择一个游戏版本。'
    return
  }
  busy.value = true
  try {
    if (!pending) {
      const p = await readPool()
      if (p.coverage.shortage) {
        error.value = `无法创建完整卡片：需要 ${size.value * size.value} 个唯一目标，当前只有 ${p.coverage.available} 个。`
        return
      }
      pending = {
        request_id: crypto.randomUUID(),
        rules: p.rules,
        size: size.value,
        calendar_zone: zone,
      }
    }
    const result = await planner.bingo(pending)
    if (alive) {
      pending = undefined
      go(`/planner/plans/${result.id}`)
    }
  } catch (e) {
    if (alive) error.value = errorText(e) + ' 创建结果未确认时，请用相同条件重试或查看已保存计划。'
  } finally {
    if (alive) busy.value = false
  }
}
onMounted(async () => {
  try {
    const result = await tracker.games()
    if (alive) games.value = result
  } catch (e) {
    if (alive) error.value = errorText(e)
  } finally {
    if (alive) loading.value = false
  }
})
onBeforeUnmount(() => {
  alive = false
})
</script>
<template>
  <header class="page-heading">
    <div>
      <p class="eyebrow">HUNT BINGO</p>
      <h2>创建 Bingo 挑战</h2>
      <p class="muted">每个格子都是新的相遇，没有免费格。</p>
    </div>
    <a href="#/planner">返回 Hunt Planner</a>
  </header>
  <p v-if="error" role="alert">{{ error }}</p>
  <p v-if="loading" role="status">正在读取本地版本…</p>
  <section v-else class="surface bingo-create">
    <form @submit.prevent="create">
      <fieldset :disabled="busy">
        <legend>挑战版本（可多选）</legend>
        <label v-for="g in games.filter((g) => g.kind !== 'storage')" :key="g.id"
          ><input v-model="rules.game_ids" type="checkbox" :value="g.id" @change="change" />{{
            gameName(g)
          }}</label
        >
      </fieldset>
      <div class="bingo-controls">
        <label
          >卡片尺寸<select
            v-model.number="size"
            aria-label="卡片尺寸"
            :disabled="busy"
            @change="change"
          >
            <option :value="3">3×3 · 9 个目标</option>
            <option :value="5">5×5 · 25 个目标</option>
          </select></label
        >
        <label
          >挑战维度<select
            v-model="rules.dimension"
            aria-label="挑战维度"
            :disabled="busy"
            @change="change"
          >
            <option value="species">物种</option>
            <option value="form">持久形态</option>
          </select></label
        >
        <label
          >挑战候选池<select
            v-model="rules.pool"
            aria-label="挑战候选池"
            :disabled="busy"
            @change="change"
          >
            <option value="all_missing">全系列历史未获得</option>
            <option value="selected_missing">所选游戏历史未获得</option>
          </select></label
        >
      </div>
      <label
        ><input
          v-model="rules.exclude_active"
          type="checkbox"
          :disabled="busy"
          @change="change"
        />排除进行中的相同目标</label
      >
      <p class="muted">
        仅使用本地已核实的直接可刷子集。候选不足不会以重复、未知、锁闪或仅传送目标补格。
      </p>
      <p class="muted">
        日历时区：{{
          zone
        }}。自动完成要求挑战后亲自合法获得的新个体；同日只有日期时先待核验，可补充具体时间依据。旧收藏、交换、活动和单纯传送不自动计入。
      </p>
      <ElButton :disabled="busy" @click="check">检查已核实候选</ElButton
      ><ElButton type="primary" native-type="submit" :loading="busy">创建 Bingo</ElButton>
    </form>
  </section>
  <section v-if="coverage" class="surface" aria-label="Bingo 候选覆盖">
    <h3>本地已核实候选</h3>
    <p>
      唯一可刷目标 {{ coverage.verified_targets }} 个 · 当前可用 {{ coverage.available }} 个 · 需要
      {{ size * size }} 个
    </p>
    <p>{{ coverage.note }}</p>
    <p v-if="coverage.shortage">候选数量不足，不能创建完整卡片。</p>
    <ul>
      <li v-for="g in coverage.games" :key="g.game_id">
        {{ g.name }}：{{ g.verified_direct }} 条已核实可刷资料，筛选后
        {{ g.available_targets }} 个目标。
      </li>
    </ul>
  </section>
</template>
<style scoped>
fieldset {
  border: 1px solid #dce1eb;
  border-radius: 8px;
  padding: 12px;
}
fieldset label {
  display: inline-flex;
  gap: 6px;
  margin: 6px 20px 6px 0;
}
.bingo-controls {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 18px;
  margin: 20px 0;
}
.bingo-controls label {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 6px;
}
select {
  font: inherit;
  padding: 9px;
  border: 1px solid #cbd3df;
  border-radius: 7px;
  background: white;
  color: inherit;
  max-width: 100%;
}
</style>
