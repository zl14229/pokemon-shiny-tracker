<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref, shallowRef, watch } from 'vue'
import {
  ElButton,
  ElEmpty,
  ElInput,
  ElOption,
  ElPagination,
  ElSelect,
  ElSkeleton,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus'
import type { ShinyRecord } from '../api/models'
import { tracker } from '../api/tracker'
import { useCatalog, errorText } from '../composables/resources'
import { holdingLabels, sourceLabels, referenceName } from '../domain/format'
import { go } from '../domain/navigation'
import Feedback from '../components/Feedback.vue'
import CatalogNotice from '../components/CatalogNotice.vue'
import ReferenceSelect from '../components/ReferenceSelect.vue'
const props = defineProps<{ query: string }>()
const { catalog, catalogError, loadCatalog } = useCatalog()
const filters = reactive({
  q: '',
  holding_status: '',
  source: '',
  has_hunt: '',
  original_game_id: null as number | null,
  current_game_id: null as number | null,
  current_species_id: null as number | null,
})
const rows = shallowRef<ShinyRecord[]>([]),
  total = ref(0),
  currentPage = ref(1),
  pageSize = ref(20),
  loading = ref(false),
  error = ref('')
let generation = 0,
  controller: AbortController | undefined
async function load() {
  const params = new URLSearchParams(props.query)
  for (const key of ['q', 'holding_status', 'source', 'has_hunt'] as const)
    filters[key] = params.get(key) ?? ''
  for (const key of ['original_game_id', 'current_game_id', 'current_species_id'] as const)
    filters[key] = params.has(key) ? Number(params.get(key)) : null
  const current = ++generation
  controller?.abort()
  controller = new AbortController()
  loading.value = true
  error.value = ''
  try {
    const result = await tracker.shinies(props.query || 'page_size=20', {
      signal: controller.signal,
    })
    if (current === generation) {
      rows.value = result.data
      total.value = result.meta.total
      currentPage.value = result.meta.page
      pageSize.value = result.meta.page_size
    }
  } catch (e) {
    if (current === generation) {
      error.value = errorText(e)
      rows.value = []
      total.value = 0
    }
  } finally {
    if (current === generation) loading.value = false
  }
}
function filter(page = 1) {
  const query = new URLSearchParams(props.query)
  query.set('page', String(page))
  query.set('page_size', String(pageSize.value))
  if (!query.has('sort')) query.set('sort', 'created_at')
  if (!query.has('order')) query.set('order', 'desc')
  for (const [key, value] of Object.entries(filters)) {
    if (value != null && String(value).trim()) query.set(key, String(value).trim())
    else query.delete(key)
  }
  const path = `/collection?${query}`
  if (location.hash.slice(1) === path) void load()
  else go(path)
}
function reset() {
  const path = '/collection?page=1&page_size=20'
  if (location.hash.slice(1) === path) void load()
  else go(path)
}
watch(() => props.query, load, { immediate: true })
onMounted(loadCatalog)
onBeforeUnmount(() => {
  ++generation
  controller?.abort()
})
</script>
<template>
  <header class="page-heading">
    <div>
      <p class="eyebrow">每一次相遇的归档</p>
      <h2>闪光收藏</h2>
      <p class="muted">保存拥有过的个体，也记住它们如今的去向。</p>
    </div>
    <ElButton tag="a" href="#/collection/new" type="primary">＋ 登记个体</ElButton>
  </header>
  <CatalogNotice /><Feedback :message="catalogError" retry @retry="loadCatalog" />
  <section class="surface list-surface" aria-label="闪光收藏列表" :aria-busy="loading">
    <form class="filter-bar collection-filters" @submit.prevent="filter()">
      <ElInput
        v-model="filters.q"
        aria-label="搜索收藏"
        placeholder="搜索物种、昵称、游戏或备注"
        clearable
      />
      <ElSelect
        v-model="filters.holding_status"
        aria-label="持有状态筛选"
        placeholder="全部持有状态"
        clearable
        ><ElOption v-for="(text, key) in holdingLabels" :key="key" :value="key" :label="text"
      /></ElSelect>
      <ElSelect v-model="filters.source" aria-label="来源筛选" placeholder="全部来源" clearable
        ><ElOption v-for="(text, key) in sourceLabels" :key="key" :value="key" :label="text"
      /></ElSelect>
      <div class="filter-field">
        <label>当前物种</label
        ><ReferenceSelect
          v-model="filters.current_species_id"
          :items="catalog.species"
          label="当前物种筛选"
          placeholder="全部物种"
        />
      </div>
      <div class="filter-field">
        <label>原始获得游戏</label
        ><ReferenceSelect
          v-model="filters.original_game_id"
          :items="catalog.games"
          label="原始游戏筛选"
          placeholder="全部原始游戏"
        />
      </div>
      <div class="filter-field">
        <label>当前保存游戏</label
        ><ReferenceSelect
          v-model="filters.current_game_id"
          :items="catalog.games"
          label="当前游戏筛选"
          placeholder="全部保存游戏"
        />
      </div>
      <div class="filter-field">
        <label>任务关联</label
        ><ElSelect
          v-model="filters.has_hunt"
          aria-label="Hunt关联筛选"
          placeholder="全部关联状态"
          clearable
          ><ElOption value="true" label="已关联Hunt" /><ElOption value="false" label="未关联Hunt"
        /></ElSelect>
      </div>
      <div class="filter-buttons">
        <ElButton native-type="submit" :disabled="loading">查询</ElButton
        ><ElButton @click="reset">重置</ElButton>
      </div>
    </form>
    <Feedback :message="error" retry @retry="load" /><ElSkeleton
      v-if="loading"
      :rows="5"
      animated
      class="loading-block"
    />
    <template v-else-if="!error">
      <ElEmpty v-if="!rows.length" description="暂无符合条件的闪光个体"
        ><ElButton tag="a" href="#/collection/new" type="primary">登记已有的闪光</ElButton></ElEmpty
      >
      <div v-else class="table-scroll">
        <ElTable :data="rows" row-key="id" :fit="false">
          <ElTableColumn label="当前个体" min-width="195"
            ><template #default="{ row }"
              ><a class="record-link" :href="`#/collection/${row.id}`">{{
                referenceName(catalog.species, row.current_species_id)
              }}</a>
              <div class="cell-secondary">
                {{ row.nickname ?? '昵称未记录' }} · #{{ row.id }}
              </div></template
            ></ElTableColumn
          >
          <ElTableColumn label="原始获得" min-width="170"
            ><template #default="{ row }"
              >{{ referenceName(catalog.games, row.original_game_id) }}
              <div class="cell-secondary">{{ row.obtained_date ?? '日期未记录' }}</div></template
            ></ElTableColumn
          >
          <ElTableColumn label="当前保存" min-width="170"
            ><template #default="{ row }"
              >{{ referenceName(catalog.games, row.current_game_id) }}
              <div class="cell-secondary">{{ row.current_location ?? '位置未记录' }}</div></template
            ></ElTableColumn
          >
          <ElTableColumn label="来源 / 持有" min-width="135"
            ><template #default="{ row }"
              ><ElTag :type="row.holding_status === 'held' ? 'success' : 'info'">{{
                holdingLabels[(row as ShinyRecord).holding_status]
              }}</ElTag>
              <div class="cell-secondary">
                {{ sourceLabels[(row as ShinyRecord).source] }}
              </div></template
            ></ElTableColumn
          >
          <ElTableColumn label="球种" min-width="120"
            ><template #default="{ row }">{{ row.ball ?? '未记录' }}</template></ElTableColumn
          >
          <ElTableColumn label="操作" width="85"
            ><template #default="{ row }"
              ><a :href="`#/collection/${row.id}/edit`" :aria-label="`编辑个体 #${row.id}`"
                >编辑</a
              ></template
            ></ElTableColumn
          >
        </ElTable>
      </div>
      <div class="pagination">
        <span class="muted">共 {{ total }} 条</span
        ><ElPagination
          :current-page="currentPage"
          :page-size="pageSize"
          :total="total"
          :pager-count="5"
          layout="prev, pager, next"
          @current-change="filter"
        />
      </div>
    </template>
  </section>
</template>
