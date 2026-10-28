<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, reactive, ref } from 'vue'
import { ElButton } from 'element-plus'
import { planner, type Plan, type PlanTarget } from '../api/planner'
import { errorText } from '../composables/resources'
const props = defineProps<{ plan: Plan }>(),
  emit = defineEmits<{ updated: [plan: Plan] }>()
const busy = ref(false),
  error = ref(''),
  notice = ref(''),
  manualTarget = ref(0),
  manualRecord = ref(0),
  manualTime = ref(''),
  manualNote = ref(''),
  confirmed = ref(false)
const existing = reactive<Record<number, number | undefined>>({}),
  rerollKeys = new Map<number, { request_id: string; expected_target_key: string }>()
let alive = true
const info = computed(() => props.plan.bingo!),
  active = computed(() => props.plan.status === 'active')
const statuses: Record<string, string> = {
  pending: '未开始',
  started: '进行中',
  completed: '已完成',
}
const count = computed(() => props.plan.targets.filter((t) => t.status === 'completed').length)
const cell = (id: number) => info.value.cells.find((c) => c.target_id === id)
const dateReviews = (id: number) =>
  cell(id)?.reviews.filter((r) =>
    ['same_day', 'date_missing', 'date_precision'].includes(r.code),
  ) ?? []
const proofDate = (id: number) => {
  const p = cell(id)?.proof
  return p?.method === 'manual_timestamp'
    ? `用户核验获得时间：${p.obtained_at}`
    : `获得日期：${p?.obtained_date || '未知'}`
}
async function action(work: () => Promise<Plan>, message = '') {
  if (busy.value) return
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    const result = await work()
    if (alive) {
      emit('updated', result)
      notice.value = message
      return true
    }
  } catch (e) {
    if (alive) {
      error.value = errorText(e)
      try {
        const current = await planner.refresh(props.plan.id)
        if (alive) {
          emit('updated', current)
          if (current.targets.find((t) => t.id === manualTarget.value)?.status === 'completed')
            manualTarget.value = 0
        }
      } catch {
        /* Keep the existing snapshot visible; never invent successful evidence. */
      }
    }
  } finally {
    if (alive) busy.value = false
  }
  return false
}
async function close(status: 'completed' | 'abandoned') {
  await action(
    () => planner.close(props.plan.id, status),
    status === 'completed' ? '已结束连线挑战，历史保留。' : '已放弃挑战，目标与历史保留。',
  )
}
async function refresh() {
  await action(() => planner.refresh(props.plan.id), '已核对本地新成果；证据不足的记录继续待核验。')
}
async function start(t: PlanTarget, link = false) {
  const id = link ? existing[t.id] : undefined
  if (link && (!id || !Number.isSafeInteger(id) || id < 1)) {
    error.value = '请输入有效的已有任务编号。'
    return
  }
  await action(
    () => planner.start(props.plan.id, t.id, id, t.target_key),
    '已关联刷闪任务，未重复建档。',
  )
}
async function reroll(t: PlanTarget) {
  if (rerollKeys.get(t.id)?.expected_target_key !== t.target_key) {
    rerollKeys.set(t.id, { request_id: crypto.randomUUID(), expected_target_key: t.target_key })
  }
  if (
    await action(
      () => planner.reroll(props.plan.id, t.id, rerollKeys.get(t.id)!),
      '未开始格子已重新抽取，原目标保留在历史中。',
    )
  )
    rerollKeys.delete(t.id)
}
function openManual(t: PlanTarget) {
  manualTarget.value = t.id
  manualRecord.value = dateReviews(t.id)[0]?.record_id ?? 0
  manualTime.value = ''
  manualNote.value = ''
  confirmed.value = false
  error.value = ''
}
async function verify() {
  const at = new Date(manualTime.value)
  if (
    !manualRecord.value ||
    !Number.isFinite(at.getTime()) ||
    at.getTime() <= Date.parse(info.value.started_at) ||
    at.getTime() > Date.now()
  ) {
    error.value = '获得时间必须晚于挑战开始，且不能在未来。'
    return
  }
  if (!confirmed.value || !manualNote.value.trim()) {
    error.value = '请填写核验依据并确认亲自合法获得。'
    return
  }
  if (
    await action(
      () =>
        planner.verifyDate(props.plan.id, manualTarget.value, {
          record_id: manualRecord.value,
          obtained_at: at.toISOString(),
          evidence_note: manualNote.value,
          confirmed: true,
        }),
      '已保存人工时间核验和完成证据。',
    )
  )
    manualTarget.value = 0
}
onMounted(refresh)
onBeforeUnmount(() => {
  alive = false
})
</script>
<template>
  <section v-if="info" class="surface bingo-card" aria-label="Bingo 挑战">
    <header class="bingo-heading">
      <div>
        <h3>Bingo #{{ plan.id }} · {{ info.size }}×{{ info.size }}</h3>
        <p>
          {{
            plan.status === 'active'
              ? '挑战进行中'
              : plan.status === 'completed'
                ? '挑战已完成 · 历史保留'
                : '挑战已放弃 · 历史保留'
          }}
        </p>
      </div>
      <a href="#/planner/bingo">创建另一张卡片</a>
    </header>
    <p role="status">
      已完成 {{ count }} / {{ info.size * info.size }} 格 · 已连线 {{ info.lines.length }} 条
    </p>
    <p class="muted">
      创建时间：{{ info.started_at }} · 日历时区
      {{ info.calendar_zone }}。无免费格；横向、纵向和两条对角线均计连线。
    </p>
    <p class="muted">
      默认仅用挑战后亲自合法获得的新个体；原始目标、形态与获得游戏必须有据可查。旧收藏、交换、活动、单纯传送不计入。只有日期且与创建同日的记录先待核验。
    </p>
    <p v-if="error" role="alert">{{ error }}</p>
    <p v-if="notice" role="status">{{ notice }}</p>
    <div v-if="active" class="bingo-actions">
      <ElButton :disabled="busy" @click="refresh">核对新成果</ElButton
      ><ElButton :disabled="busy || !info.lines.length" @click="close('completed')"
        >结束已连线挑战</ElButton
      ><ElButton :disabled="busy" @click="close('abandoned')">放弃挑战并保留历史</ElButton>
    </div>
    <p v-if="info.lines.length">
      连线：<span v-for="(line, i) in info.lines" :key="i"
        >{{ line.join(' → ') }}{{ i !== info.lines.length - 1 ? '；' : '' }}</span
      >
    </p>
    <div class="bingo-board-scroll" tabindex="0" aria-label="Bingo 格子，可横向滚动">
      <div
        class="bingo-board"
        :style="{ '--size': info.size, '--board-width': info.size === 3 ? '660px' : '1100px' }"
      >
        <article
          v-for="t in plan.targets"
          :key="t.id"
          class="bingo-cell"
          :class="{ 'is-completed': t.status === 'completed' }"
          :aria-label="`格子 ${t.position} ${t.snapshot.name}`"
        >
          <span class="cell-number">{{ t.position }}</span>
          <h4>{{ t.snapshot.name }}</h4>
          <p>{{ t.snapshot.choice.game_name }} · {{ statuses[t.status] }}</p>
          <p v-if="cell(t.id)?.kind === 'needs_review'" class="needs-review">
            有记录待核验，尚未完成
          </p>
          <a v-if="t.hunt_id" :href="`#/hunts/${t.hunt_id}`">查看 Hunt #{{ t.hunt_id }}</a>
          <template v-if="t.status === 'completed'"
            ><p>
              <a :href="`#/collection/${t.shiny_record_id}`">完成个体 #{{ t.shiny_record_id }}</a>
            </p>
            <p>{{ proofDate(t.id) }}</p>
            <details v-if="cell(t.id)?.proof">
              <summary>完成证据</summary>
              <p>{{ cell(t.id)?.proof?.evidence_note }}</p>
              <p>来源：亲自刷闪 · 原始获得游戏 #{{ cell(t.id)?.proof?.original_game_id }}</p>
              <p>核验时间 {{ cell(t.id)?.proof?.confirmed_at }}</p>
              <p>资料批次 {{ cell(t.id)?.proof?.availability.batch_version }}</p>
            </details></template
          >
          <template v-if="active && t.status === 'pending'"
            ><ElButton :disabled="busy" @click="start(t)">开始此格</ElButton
            ><ElButton :disabled="busy" @click="reroll(t)">重抽未开始格</ElButton>
            <details>
              <summary>关联已有任务</summary>
              <label
                >任务编号<input
                  v-model.number="existing[t.id]"
                  :aria-label="`格子 ${t.position} 已有任务编号`"
                  type="number"
                  min="1"
                  :disabled="busy" /></label
              ><ElButton :disabled="busy" @click="start(t, true)">关联此任务</ElButton>
            </details></template
          >
          <details v-if="cell(t.id)?.reviews.length">
            <summary>未计入的记录与原因</summary>
            <p v-for="r in cell(t.id)?.reviews" :key="r.record_id">
              <a :href="`#/collection/${r.record_id}`">个体 #{{ r.record_id }}</a
              >：{{ r.reason }}
            </p>
          </details>
          <ElButton
            v-if="active && t.status !== 'completed' && dateReviews(t.id).length"
            :disabled="busy"
            @click="openManual(t)"
            >补充时间并人工核验</ElButton
          >
        </article>
      </div>
    </div>
    <form
      v-if="manualTarget && active"
      class="manual-evidence"
      aria-label="人工时间核验"
      @submit.prevent="verify"
    >
      <h4>
        格子 {{ plan.targets.find((t) => t.id === manualTarget)?.position }} · 补充获得时间证据
      </h4>
      <p>
        仅解决时间精度问题；来源、原始目标、游戏、形态和合法性仍会核对。不会修改个人记录或允许旧收藏、交换及传送替代新获得。
      </p>
      <label
        >核验个体<select v-model.number="manualRecord" aria-label="核验个体" :disabled="busy">
          <option v-for="r in dateReviews(manualTarget)" :key="r.record_id" :value="r.record_id">
            个体 #{{ r.record_id }}
          </option>
        </select></label
      ><label
        >具体获得时间（本机时区）<input
          v-model="manualTime"
          type="datetime-local"
          step="1"
          aria-label="具体获得时间"
          :disabled="busy"
          required /></label
      ><label
        >核验依据<textarea
          v-model="manualNote"
          aria-label="核验依据"
          maxlength="500"
          :disabled="busy"
          required
          placeholder="例如当时记录的时间或自己的视频时间"
        /></label
      ><label class="manual-confirm"
        ><input
          v-model="confirmed"
          type="checkbox"
          :disabled="busy"
        />我已核对该个体在挑战创建后亲自合法获得，时间与原记录一致</label
      ><ElButton type="primary" native-type="submit" :disabled="busy">确认时间证据并完成</ElButton
      ><ElButton :disabled="busy" @click="manualTarget = 0">取消核验</ElButton>
    </form>
    <details>
      <summary>创建时的候选覆盖与资料</summary>
      <p>{{ plan.coverage.note }}</p>
      <p>
        创建时 {{ plan.coverage.available }} 个可用唯一目标，抽取
        {{ plan.coverage.actual }} 个；中文名称仍待交叉核验。
      </p>
      <p>{{ plan.coverage.versions.join('、') }}</p>
    </details>
    <details class="bingo-history">
      <summary>挑战历史（{{ info.events.length }} 条）</summary>
      <ol>
        <li v-for="event in info.events" :key="event.id">
          {{ event.summary }} · {{ event.created_at }}
        </li>
      </ol>
    </details>
  </section>
</template>
<style scoped>
.bingo-heading {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.bingo-heading h3 {
  margin-bottom: 6px;
}
.bingo-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin: 14px 0;
}
.bingo-board-scroll {
  overflow-x: auto;
  max-width: 100%;
  margin: 20px 0;
  border-radius: 12px;
}
.bingo-board {
  display: grid;
  grid-template-columns: repeat(var(--size), minmax(0, 1fr));
  min-width: var(--board-width);
  gap: 12px;
}
.bingo-cell {
  padding: 16px;
  border: 1px solid #dce1eb;
  border-radius: 12px;
  background: #fafbfe;
  min-width: 0;
  overflow-wrap: anywhere;
}
.bingo-cell.is-completed {
  background: #edf7ed;
  border-color: #aacdaa;
}
.bingo-cell h4 {
  font-size: 17px;
  margin: 6px 0;
}
.bingo-cell p {
  font-size: 12px;
  margin: 8px 0;
}
.cell-number {
  font-size: 12px;
  color: #78869a;
}
.needs-review {
  color: #8a5a09;
}
.bingo-cell .el-button {
  margin: 4px 4px 4px 0;
}
.bingo-cell input {
  width: 100%;
  min-width: 0;
}
summary {
  color: #435da7;
  cursor: pointer;
  margin: 10px 0;
}
.manual-evidence {
  padding: 18px;
  border: 1px solid #ccd5ec;
  border-radius: 12px;
  margin: 20px 0;
}
.manual-evidence > label {
  display: block;
  max-width: 100%;
  overflow-wrap: anywhere;
  margin: 12px 0;
}
.manual-evidence input:not([type='checkbox']),
.manual-evidence select,
.manual-evidence textarea {
  display: block;
  width: 100%;
  max-width: 500px;
  font: inherit;
  padding: 8px;
  border: 1px solid #cbd3df;
  border-radius: 6px;
  color: inherit;
  background: white;
}
.manual-confirm {
  font-size: 14px;
}
.bingo-history ol {
  padding-left: 22px;
  overflow-wrap: anywhere;
}
</style>
