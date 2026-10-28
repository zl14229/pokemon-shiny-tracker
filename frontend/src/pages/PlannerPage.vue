<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import BingoCard from '../components/BingoCard.vue'
import { ElButton } from 'element-plus'
import { planner, type Plan, type PlannerRules, type PlanListItem } from '../api/planner'
import { tracker } from '../api/tracker'
import type { Game } from '../api/models'
import { errorText } from '../composables/resources'
import { go } from '../domain/navigation'
const props = defineProps<{ id?: number }>()
const games = shallowRef<Game[]>([]),
  saved = shallowRef<PlanListItem[]>([]),
  plan = shallowRef<Plan | null>(null)
const rules = ref<PlannerRules>({
  game_ids: [],
  dimension: 'species',
  pool: 'all_missing',
  count: 1,
  exclude_active: true,
})
const busy = ref(false),
  loading = ref(true),
  error = ref(''),
  notice = ref(''),
  offset = ref(0),
  total = ref(0)
let generation = 0,
  controller: AbortController | undefined,
  pending: { request_id: string; rules: PlannerRules } | undefined
const gameName = (g: Game) => (g.name_status !== 'conflict' && g.name_zh ? g.name_zh : g.name)
async function load() {
  const current = ++generation
  controller?.abort()
  controller = new AbortController()
  loading.value = true
  error.value = ''
  try {
    const [gs, ls, detail] = await Promise.all([
      tracker.games({ signal: controller.signal }),
      planner.list(offset.value, { signal: controller.signal }),
      props.id ? planner.get(props.id, { signal: controller.signal }) : Promise.resolve(null),
    ])
    if (current !== generation) return
    games.value = gs
    saved.value = ls.data
    total.value = ls.meta.total
    if (props.id) plan.value = detail
  } catch (e) {
    if (current === generation) error.value = errorText(e)
  } finally {
    if (current === generation) loading.value = false
  }
}
function changed() {
  if (busy.value) return
  pending = undefined
  plan.value = null
  notice.value = ''
  error.value = ''
}
async function draw() {
  if (busy.value) return
  error.value = ''
  notice.value = ''
  if (!rules.value.game_ids.length) {
    error.value = '请至少选择一个游戏版本。'
    return
  }
  if (!pending)
    pending = {
      request_id: crypto.randomUUID(),
      rules: { ...rules.value, game_ids: [...rules.value.game_ids] },
    }
  busy.value = true
  plan.value = null
  const current = generation
  try {
    const result = await planner.draw(pending)
    if (current !== generation) return
    plan.value = result
    pending = undefined
  } catch (e) {
    if (current === generation) error.value = errorText(e) + ' 可用相同筛选重试生成。'
  } finally {
    if (current === generation) busy.value = false
  }
}
async function save() {
  if (!plan.value || busy.value) return
  busy.value = true
  error.value = ''
  const current = generation
  try {
    const result = await planner.save(plan.value.id)
    if (current !== generation) return
    plan.value = result
    go(`/planner/plans/${result.id}`)
  } catch (e) {
    error.value = errorText(e)
  } finally {
    busy.value = false
  }
}
async function start(target: number) {
  if (!plan.value || busy.value) return
  busy.value = true
  error.value = ''
  notice.value = ''
  const current = generation
  try {
    const result = await planner.start(plan.value.id, target)
    if (current !== generation) return
    plan.value = result
    notice.value = '已关联 Hunt；重复操作会保留同一个 Hunt。'
  } catch (e) {
    error.value = errorText(e)
  } finally {
    busy.value = false
  }
}
async function page(delta: number) {
  offset.value += delta
  await load()
}
onMounted(load)
onBeforeUnmount(() => {
  generation++
  controller?.abort()
})
</script>
<template>
  <header class="page-heading">
    <div>
      <p class="eyebrow">HUNT PLANNER</p>
      <h2>下一次闪光目标</h2>
      <p class="muted">从已核实的本地候选中，找一个新的相遇。</p>
    </div>
    <a v-if="id" href="#/planner">返回生成器</a><a v-else href="#/planner/bingo">创建 Bingo 挑战</a>
  </header>
  <p v-if="error" role="alert">
    {{ error }}
    <ElButton v-if="loading === false && !busy" @click="load">重新读取</ElButton>
  </p>
  <p v-if="notice" role="status">{{ notice }}</p>
  <p v-if="loading" role="status">正在读取本地计划…</p>
  <template v-else>
    <section v-if="!id" class="surface planner-form">
      <form @submit.prevent="draw">
        <fieldset :disabled="busy">
          <legend>选择游戏版本（可多选）</legend>
          <label
            v-for="g in games.filter((g) => g.kind !== 'storage')"
            :key="g.id"
            class="game-option"
            ><input v-model="rules.game_ids" type="checkbox" :value="g.id" @change="changed" />{{
              gameName(g)
            }}</label
          >
        </fieldset>
        <div class="planner-controls">
          <label
            >抽取维度<select
              v-model="rules.dimension"
              aria-label="抽取维度"
              :disabled="busy"
              @change="changed"
            >
              <option value="species">物种</option>
              <option value="form">持久形态</option>
            </select></label
          >
          <label
            >候选池<select
              v-model="rules.pool"
              aria-label="候选池"
              :disabled="busy"
              @change="changed"
            >
              <option value="all_missing">全系列历史未获得</option>
              <option value="selected_missing">所选游戏历史未获得</option>
            </select></label
          >
          <label
            >目标数量<input
              v-model.number="rules.count"
              aria-label="目标数量"
              type="number"
              min="1"
              max="100"
              :disabled="busy"
              @input="changed"
          /></label>
        </div>
        <label class="exclude-option"
          ><input
            v-model="rules.exclude_active"
            type="checkbox"
            :disabled="busy"
            @change="changed"
          />排除进行中的相同目标（跨版本）</label
        >
        <p class="muted">
          历史按原始获得游戏统计。传送到某版本不会变成在那里获得；不同收藏体系须分别生成。
        </p>
        <ElButton type="primary" native-type="submit" :loading="busy">生成随机目标</ElButton>
      </form>
    </section>
    <BingoCard v-if="plan?.kind === 'bingo'" :plan="plan" @updated="plan = $event" />
    <section v-else-if="plan" class="surface" aria-label="生成结果">
      <h3>
        {{ plan.status === 'draft' ? '本次结果' : `已保存计划 #${plan.id}` }}
      </h3>
      <p>{{ plan.coverage.note }}</p>
      <p role="status">
        已核实唯一目标 {{ plan.coverage.verified_targets }} 个 · 排除已获得
        {{ plan.coverage.excluded_obtained }} 个 · 排除进行中 {{ plan.coverage.excluded_active }} 个
        · 可用候选 {{ plan.coverage.available }} 个
      </p>
      <p v-if="plan.coverage.shortage" class="planner-warning">
        候选数量不足：请求 {{ plan.coverage.requested }} 个，实际
        {{ plan.coverage.actual }} 个。不会加入未知、锁闪或仅传送目标。
      </p>
      <p v-if="!plan.targets.length">
        当前没有符合条件的已核实目标。请调整版本或等待资料核验；不会自动联网补全。
      </p>
      <ul class="coverage-list">
        <li v-for="g in plan.coverage.games" :key="g.game_id">
          {{ g.name }}：{{ g.reviewed_entries }} 条资料，{{
            g.verified_direct
          }}
          条已核实可直接刷闪，筛选后 {{ g.available_targets }} 个目标。
        </li>
      </ul>
      <p class="muted">
        资料版本：{{
          plan.coverage.versions.join('、') || '暂无已核实来源'
        }}。中文名称仍沿用来源状态，待交叉核验。
      </p>
      <p class="muted">
        先对唯一目标均匀抽取，再在该目标的可刷版本间均匀选择；每个目标只出现一次。
      </p>
      <div class="planner-targets">
        <article
          v-for="t in plan.targets"
          :key="t.id"
          class="planner-target"
          :aria-label="t.snapshot.name"
        >
          <h4>{{ t.position }} · {{ t.snapshot.name }}</h4>
          <p>
            {{ t.snapshot.choice.game_name }} ·
            {{ t.form_id ? '形态目标' : '物种目标' }}
          </p>
          <p v-if="t.snapshot.name_status !== 'verified'" class="muted">中文名待核实</p>
          <details>
            <summary>方法、条件与来源</summary>
            <div v-for="e in t.snapshot.choice.evidence" :key="e.id">
              <p>
                形态 #{{ e.form_id }} ·
                {{ e.fact.guide_status === 'complete' ? '已核对攻略' : '方法待完善（部分资料）' }}
              </p>
              <p v-if="!e.fact.methods.length">暂无已核实具体方法，概率未知。</p>
              <div v-for="(m, i) in e.fact.methods" :key="i">
                <p>{{ m.name }} · {{ m.location || '地点待完善' }}</p>
                <p>{{ m.conditions }}</p>
                <p>
                  遭遇率：{{
                    m.encounter_rate
                      ? `${m.encounter_rate.numerator}/${m.encounter_rate.denominator}`
                      : '未知'
                  }}；闪光率：{{
                    m.shiny_rate ? `${m.shiny_rate.numerator}/${m.shiny_rate.denominator}` : '未知'
                  }}
                </p>
                <p v-if="m.encounter_rate">遭遇率条件：{{ m.encounter_rate.conditions }}</p>
                <p v-if="m.shiny_rate">闪光率条件：{{ m.shiny_rate.conditions }}</p>
              </div>
              <p>
                <a :href="e.fact.evidence.url" target="_blank" rel="noopener noreferrer"
                  >{{ e.fact.evidence.source }} · 来源</a
                >
                · 核对 {{ e.fact.evidence.checked_at }} · {{ e.batch_version }}
              </p>
              <p v-for="(source, i) in e.fact.cross_checks" :key="i">
                <a :href="source.url" target="_blank" rel="noopener noreferrer"
                  >{{ source.source }} · 交叉核对</a
                >
              </p>
            </div>
          </details>
          <a v-if="t.hunt_id" :href="`#/hunts/${t.hunt_id}`">查看 Hunt #{{ t.hunt_id }}</a>
          <ElButton v-else-if="plan.status === 'active'" :disabled="busy" @click="start(t.id)"
            >转为 Hunt</ElButton
          >
        </article>
      </div>
      <ElButton
        v-if="plan.status === 'draft'"
        type="primary"
        :disabled="busy || !plan.targets.length"
        @click="save"
        >保存本次结果</ElButton
      >
      <p v-if="plan.status === 'draft'" class="muted">
        生成仅保留草稿快照；保存后进入计划历史。只有逐个点击“转为 Hunt”才会创建任务。
      </p>
    </section>
    <section class="surface" aria-label="已保存计划">
      <h3>已保存计划</h3>
      <p v-if="!saved.length">还没有已保存计划。</p>
      <ul>
        <li v-for="p in saved" :key="p.id">
          <a :href="`#/planner/plans/${p.id}`"
            >{{ p.kind === 'bingo' ? 'Bingo' : 'Random Hunt' }} #{{ p.id }}</a
          >
          · {{ p.target_count }} 个目标 · {{ p.created_at }}
        </li>
      </ul>
      <div class="pager">
        <ElButton :disabled="offset === 0 || busy" @click="page(-20)">上一页计划</ElButton
        ><span>{{ total ? offset + 1 : 0 }}–{{ Math.min(offset + 20, total) }} / {{ total }}</span
        ><ElButton :disabled="offset + 20 >= total || busy" @click="page(20)">下一页计划</ElButton>
      </div>
    </section>
  </template>
</template>
<style scoped>
.planner-controls {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 18px;
  margin: 20px 0;
}
.planner-controls label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
select,
input[type='number'] {
  padding: 9px;
  border: 1px solid #cbd3df;
  border-radius: 7px;
  background: white;
  font: inherit;
  max-width: 100%;
  color: inherit;
}
fieldset {
  border: 1px solid #dce1eb;
  border-radius: 10px;
  padding: 14px;
}
.game-option {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 8px 20px 8px 0;
}
.exclude-option {
  display: flex;
  gap: 8px;
  align-items: center;
}
.planner-targets {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 260px), 1fr));
  gap: 16px;
  margin: 20px 0;
}
.planner-target {
  min-width: 0;
  border: 1px solid #dce1eb;
  border-radius: 12px;
  padding: 18px;
  background: #fafbfe;
  overflow-wrap: anywhere;
}
.planner-target h4 {
  font-size: 17px;
  margin-bottom: 6px;
}
.planner-target p {
  margin: 8px 0;
  font-size: 13px;
}
.planner-warning {
  color: #8a5a09;
  background: #fff6dd;
  padding: 12px;
  border-radius: 8px;
}
.coverage-list {
  padding-left: 20px;
  overflow-wrap: anywhere;
}
summary {
  cursor: pointer;
  color: #435da7;
  margin: 12px 0;
}
</style>
