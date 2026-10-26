<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, shallowRef, watch } from 'vue'
import { ElButton } from 'element-plus'
import { dex, sourceLink, type DexResult } from '../api/dex'
import { tracker } from '../api/tracker'
import type { Game } from '../api/models'
import { referenceOption } from '../domain/format'
import { go } from '../domain/navigation'
import { errorText } from '../composables/resources'
import Feedback from '../components/Feedback.vue'
import PokemonPortrait from '../components/PokemonPortrait.vue'
const props = defineProps<{ query?: string }>()
const defaults = () => ({
  dimension: 'species',
  basis: 'historical',
  scope: 'all',
  game_id: '',
  collection_system: 'pokemon',
  display: 'all',
  acquired: 'all',
  holding: 'all',
  q: '',
})
const filters = reactive(defaults()),
  result = shallowRef<DexResult>(),
  games = shallowRef<Game[]>([])
const loading = ref(false),
  error = ref(''),
  gameError = ref('')
let generation = 0,
  controller: AbortController | undefined
function readFilters() {
  const params = new URLSearchParams(props.query)
  const values = defaults()
  for (const key of Object.keys(values) as (keyof typeof values)[])
    if (params.has(key)) values[key] = params.get(key)!
  Object.assign(filters, values)
}
async function load() {
  const run = ++generation
  controller?.abort()
  controller = new AbortController()
  const signal = controller.signal
  loading.value = true
  error.value = ''
  gameError.value = ''
  result.value = undefined
  const params = new URLSearchParams(props.query)
  if (!params.has('page_size')) params.set('page_size', '24')
  const [data, versions] = await Promise.allSettled([
    dex.list(params.toString(), { signal }),
    tracker.games({ signal }),
  ])
  if (run !== generation) return
  if (data.status === 'fulfilled') result.value = data.value
  else error.value = errorText(data.reason)
  if (versions.status === 'fulfilled') games.value = versions.value
  else gameError.value = '游戏选项暂不可用，可重试读取本地资料。'
  loading.value = false
}
function navigate(params: URLSearchParams) {
  const query = params.toString()
  if (query === (props.query ?? '')) void load()
  else {
    loading.value = true
    result.value = undefined
    go(`/dex${query ? `?${query}` : ''}`)
  }
}
function apply() {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(filters))
    if (value && (key !== 'game_id' || filters.scope === 'game')) params.set(key, value)
  if (filters.scope === 'game') {
    const game = games.value.find((g) => String(g.id) === filters.game_id)
    if (game) params.set('collection_system', game.collection_system)
  }
  params.set('page_size', '24')
  navigate(params)
}
function reset() {
  Object.assign(filters, defaults())
  navigate(new URLSearchParams())
}
function missing() {
  const params = new URLSearchParams(props.query)
  params.set('display', 'legal')
  params.set('acquired', 'missing')
  params.set('holding', 'all')
  params.delete('page')
  navigate(params)
}
function pageTo(page: number) {
  const params = new URLSearchParams(props.query)
  params.set('page', String(page))
  navigate(params)
}
const systems = computed(() => [
  ...new Set(['pokemon', ...games.value.map((g) => g.collection_system)]),
])
const count = (n: number) => n.toLocaleString('zh-CN')
const availabilityLabels: Record<string, string> = {
  unknown: '合法性待核实',
  confirmed_available: '已核实范围内可获取',
  confirmed_locked: '已核实范围内锁闪',
  transfer_only: '已核实范围内仅传送',
}
watch(
  () => props.query,
  () => {
    readFilters()
    void load()
  },
)
onMounted(() => {
  readFilters()
  void load()
})
onBeforeUnmount(() => {
  ++generation
  controller?.abort()
})
</script>
<template>
  <header class="page-heading">
    <div>
      <p class="eyebrow">全国图鉴 · 本地资料</p>
      <h2>闪光图鉴</h2>
      <p class="muted">看见每一份收藏，也看清仍待核实的空白。</p>
    </div>
    <ElButton tag="a" href="#/collection">查看个体收藏</ElButton>
  </header>
  <form class="surface dex-filters" aria-label="图鉴筛选" @submit.prevent="apply">
    <label
      >统计维度<select v-model="filters.dimension" aria-label="统计维度" :disabled="loading">
        <option value="species">物种</option>
        <option value="form">形态</option>
      </select></label
    >
    <label
      >收藏口径<select v-model="filters.basis" aria-label="收藏口径" :disabled="loading">
        <option value="historical">历史曾获得</option>
        <option value="current">当前仍持有</option>
      </select></label
    >
    <label
      >游戏范围<select v-model="filters.scope" aria-label="游戏范围" :disabled="loading">
        <option value="all">全系列（当前体系）</option>
        <option value="mainline">主系列（已分类版本）</option>
        <option value="game">指定游戏</option>
      </select></label
    >
    <label v-if="filters.scope === 'game'"
      >指定游戏<select
        v-model="filters.game_id"
        aria-label="指定游戏"
        required
        :disabled="loading || !!gameError"
      >
        <option value="">请选择游戏</option>
        <option v-for="g in games" :key="g.id" :value="String(g.id)">
          {{ referenceOption(g) }}{{ g.kind === 'unknown' ? ' · 类别待核实' : '' }}
        </option>
      </select></label
    >
    <label v-else
      >收藏体系<select
        v-model="filters.collection_system"
        aria-label="收藏体系"
        :disabled="loading"
      >
        <option v-for="system in systems" :key="system" :value="system">
          {{
            system === 'pokemon'
              ? '宝可梦（普通收藏）'
              : system === 'sleep'
                ? 'Pokémon Sleep（独立）'
                : system
          }}
        </option>
      </select></label
    >
    <label
      >记录状态<select v-model="filters.acquired" aria-label="记录状态" :disabled="loading">
        <option value="all">全部</option>
        <option value="obtained">已记录获得</option>
        <option value="missing">未记录获得</option>
      </select></label
    >
    <label
      >持有筛选<select v-model="filters.holding" aria-label="持有筛选" :disabled="loading">
        <option value="all">全部持有状态</option>
        <option value="held">有确认持有</option>
        <option value="no_confirmed_held">无确认持有（可含未知）</option>
        <option value="unknown">有持有状态未知</option>
      </select></label
    >
    <label
      >目标范围<select v-model="filters.display" aria-label="目标范围" :disabled="loading">
        <option value="all">完整本地资料</option>
        <option value="legal">已核实合法子集</option>
      </select></label
    >
    <label class="dex-search"
      >搜索图鉴<input
        v-model="filters.q"
        aria-label="搜索图鉴"
        type="search"
        maxlength="100"
        placeholder="中文、其他语言、编号或标识"
        :disabled="loading"
    /></label>
    <div class="dex-actions">
      <ElButton native-type="submit" type="primary" :disabled="loading">应用筛选</ElButton
      ><ElButton :disabled="loading" @click="reset">重置</ElButton>
    </div>
  </form>
  <Feedback :message="error || gameError" retry @retry="load" />
  <p v-if="loading" role="status">正在读取本地图鉴…</p>
  <template v-if="result && !loading">
    <section class="dex-progress" aria-label="收藏完成度">
      <div class="surface">
        <span class="muted">已核实子集完成度</span
        ><strong>{{
          result.summary.percentage === null
            ? '分母待核实'
            : `${result.summary.percentage.toFixed(1)}%`
        }}</strong>
        <p>
          {{ result.summary.obtained_eligible }} /
          {{ result.summary.eligible_targets }} 个已核实目标
        </p>
      </div>
      <div class="surface">
        <span class="muted"
          >{{
            result.filters.basis === 'historical' ? '历史已获得目标' : '当前持有目标'
          }}（含分母外）</span
        ><strong>{{ count(result.summary.observed_targets) }}</strong>
        <p>其中 {{ count(result.summary.observed_outside_denominator) }} 个不纳入当前分母</p>
      </div>
      <div class="surface">
        <span class="muted">已核实子集缺失</span
        ><strong>{{
          result.summary.eligible_targets === 0
            ? '待资料核验'
            : count(result.summary.missing_eligible)
        }}</strong
        ><ElButton @click="missing">查看已核实缺失</ElButton>
      </div>
    </section>
    <section class="dex-coverage surface" aria-label="资料覆盖说明">
      <strong>资料尚未完整核验</strong>
      <p>
        本地 {{ count(result.coverage.species) }} 个物种、{{
          count(result.coverage.forms)
        }}
        个形态；{{ result.coverage.unknown_forms }} 个形态口径待核实。当前范围已核验获取资料
        {{ result.coverage.verified_availability }} 条，中文待核实
        {{ count(result.coverage.pending_chinese_names) }} 个目标。
      </p>
      <p v-if="result.coverage.selected_games === 0">
        当前范围没有已分类且适用的游戏，不能宣称游戏覆盖完整。
      </p>
      <p v-if="result.coverage.unclassified_games">
        本地有
        {{ result.coverage.unclassified_games }} 个版本类别待核实；主系列筛选只使用已分类版本。
      </p>
      <details>
        <summary>统计口径与来源</summary>
        <p>
          历史按原始获得游戏，当前按保存游戏；仅确认持有的当前形态进入当前收藏。转移不新增个体，未知原始形态不推断。Sleep
          等体系独立统计。
        </p>
        <p>
          分母仅含本地已核实可直接合法获取的持久目标，未知、争议、仅传送、锁闪及未纳入形态排除；缺失清单只代表这一子集。搜索、记录和持有筛选不改变范围总进度。
        </p>
        <p>{{ result.coverage.note }}</p>
        <p>资料版本：{{ result.coverage.revisions.join('、') || '尚无已审核快照' }}</p>
        <p>
          可获取性版本：{{
            result.coverage.availability_versions.join('、') || '尚无已核实游戏事实'
          }}
        </p>
      </details>
    </section>
    <div class="dex-list-heading">
      <h3>{{ result.filters.dimension === 'species' ? '物种' : '形态' }}目标</h3>
      <span class="muted">筛选后 {{ count(result.meta.total) }} 个</span>
    </div>
    <p v-if="!result.data.length" class="surface" role="status">
      {{
        result.filters.display === 'legal' && result.summary.eligible_targets === 0
          ? '当前范围尚无已核实的合法目标，缺失清单待资料核验。'
          : '此筛选下暂无目标，可更换条件查看。'
      }}
    </p>
    <div class="dex-grid">
      <article
        v-for="row in result.data"
        :key="row.id"
        class="surface dex-card"
        :class="{ 'dex-obtained': row.obtained }"
        :aria-label="row.name"
      >
        <div class="dex-card-top">
          <PokemonPortrait :name="row.name" />
          <div>
            <small class="muted"
              >#{{ String(row.national_dex).padStart(4, '0')
              }}{{ row.form_id ? ` · 形态 #${row.form_id}` : '' }}</small
            >
            <h3>{{ row.name }}</h3>
            <p class="muted dex-slug">{{ row.slug }}</p>
          </div>
        </div>
        <div class="dex-badges">
          <span :class="row.obtained ? 'badge-obtained' : ''">{{
            row.obtained
              ? result.filters.basis === 'historical'
                ? '历史已获得'
                : '当前持有'
              : '未记录获得'
          }}</span
          ><span v-if="row.name_status === 'conflict' || row.cross_check_status === 'conflict'"
            >名称来源冲突 · 已回退</span
          ><span v-else-if="!row.names['zh-hans']">中文名缺失 · 已回退</span
          ><span v-else-if="row.name_status !== 'verified' || row.cross_check_status !== 'verified'"
            >中文名待核实</span
          ><span v-if="row.inclusion !== 'included'">{{
            row.inclusion === 'excluded' ? '主形态口径排除' : '形态口径待核实'
          }}</span>
        </div>
        <p>
          历史 {{ row.historical_count }} · 当前持有 {{ row.held_count
          }}<span v-if="row.unknown_holding_count">
            · 持有未知 {{ row.unknown_holding_count }}</span
          >
        </p>
        <p class="muted">
          {{ availabilityLabels[row.availability]
          }}{{ row.eligible ? ' · 纳入分母' : ' · 不纳入分母' }}
        </p>
        <div class="dex-records" v-if="row.record_ids.length">
          <a v-for="recordId in row.record_ids" :key="recordId" :href="`#/collection/${recordId}`"
            >查看个体 #{{ recordId }}</a
          >
        </div>
        <details class="dex-sources">
          <summary>名称与来源</summary>
          <dl>
            <template v-for="(name, language) in row.names" :key="language"
              ><dt>{{ language }}</dt>
              <dd>{{ name }}</dd></template
            >
          </dl>
          <p v-if="sourceLink(row.source_url)">
            <a :href="sourceLink(row.source_url)" target="_blank" rel="noopener noreferrer"
              >PokéAPI 来源</a
            >
          </p>
          <p v-if="sourceLink(row.cross_check_url)">
            <a :href="sourceLink(row.cross_check_url)" target="_blank" rel="noopener noreferrer"
              >52Poké 待核对</a
            >
          </p>
          <p v-if="row.revision" class="muted">版本 {{ row.revision }}</p>
        </details>
      </article>
    </div>
    <div class="dex-pagination">
      <ElButton :disabled="result.meta.page === 1" @click="pageTo(result.meta.page - 1)"
        >上一页</ElButton
      ><span>第 {{ result.meta.page }} 页</span
      ><ElButton
        :disabled="result.meta.page * result.meta.page_size >= result.meta.total"
        @click="pageTo(result.meta.page + 1)"
        >下一页</ElButton
      >
    </div>
  </template>
</template>
<style scoped>
.dex-filters {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 22px;
}
.dex-filters label {
  font-size: 12px;
  color: #62718a;
  display: grid;
  gap: 6px;
}
.dex-filters select,
.dex-filters input {
  width: 100%;
  min-width: 0;
  min-height: 40px;
  padding: 8px;
  border: 1px solid #dce1ec;
  border-radius: 8px;
  background: white;
  color: #263443;
  font: inherit;
  font-size: 13px;
}
.dex-search {
  grid-column: span 2;
}
.dex-actions {
  align-self: end;
  display: flex;
  gap: 8px;
}
.dex-actions .el-button + .el-button {
  margin-left: 0;
}
.dex-progress {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin: 24px 0;
}
.dex-progress strong {
  display: block;
  font-size: 28px;
  margin: 9px 0;
  color: #435da7;
}
.dex-progress p {
  font-size: 12px;
  color: #738095;
}
.dex-coverage {
  border-left: 4px solid #c7a86b;
  font-size: 13px;
  margin-bottom: 26px;
}
.dex-coverage details {
  margin-top: 10px;
}
.dex-coverage p,
.dex-sources p {
  overflow-wrap: anywhere;
}
.dex-list-heading {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.dex-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}
.dex-card {
  padding: 22px;
  border-top: 3px solid #e6eaf3;
  min-width: 0;
}
.dex-obtained {
  border-top-color: #aa8c50;
  background: #fffefb;
}
.dex-card-top {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 14px;
}
.dex-card h3 {
  font-size: 17px;
  margin: 5px 0;
}
.dex-slug {
  font-size: 11px;
  overflow-wrap: anywhere;
}
.dex-card p {
  font-size: 12px;
}
.dex-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.dex-badges span {
  font-size: 10px;
  background: #f0f2f7;
  color: #65738d;
  border-radius: 5px;
  padding: 3px 7px;
}
.dex-badges .badge-obtained {
  background: #f6ebcf;
  color: #866623;
}
.dex-records {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 10px;
  font-size: 12px;
}
.dex-sources {
  font-size: 11px;
  margin-top: 14px;
  color: #64718a;
}
.dex-sources dl {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 5px 12px;
}
.dex-sources dd {
  margin: 0;
  overflow-wrap: anywhere;
}
.dex-pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 18px;
  margin: 30px 0;
}
summary {
  cursor: pointer;
}
@media (max-width: 1200px) {
  .dex-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 850px) {
  .dex-filters {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .dex-progress {
    grid-template-columns: 1fr;
  }
  .dex-grid {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 400px) {
  .dex-filters {
    grid-template-columns: 1fr;
  }
  .dex-search {
    grid-column: auto;
  }
  .dex-list-heading {
    flex-wrap: wrap;
  }
  .dex-card {
    padding: 18px;
  }
}
</style>
