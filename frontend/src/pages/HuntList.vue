<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
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
} from 'element-plus'
import { tracker } from '../api/tracker'
import type { Hunt } from '../api/models'
import { useCatalog, errorText } from '../composables/resources'
import { duration, referenceName, statusLabels } from '../domain/format'
import { go } from '../domain/navigation'
import Feedback from '../components/Feedback.vue'
import CatalogNotice from '../components/CatalogNotice.vue'
import HuntStatusTag from '../components/HuntStatusTag.vue'
const props = defineProps<{ query: string }>()
const { catalog, catalogError, loadCatalog } = useCatalog()
const rows = shallowRef<Hunt[]>([]),
  total = ref(0),
  currentPage = ref(1),
  pageSize = ref(20)
const search = ref(''),
  status = ref(''),
  loading = ref(false),
  error = ref('')
let generation = 0,
  controller: AbortController | undefined
async function load() {
  const params = new URLSearchParams(props.query)
  search.value = params.get('q') ?? ''
  status.value = params.get('status') ?? ''
  currentPage.value = Number(params.get('page') ?? 1)
  pageSize.value = Number(params.get('page_size') ?? 20)
  const current = ++generation
  controller?.abort()
  controller = new AbortController()
  loading.value = true
  error.value = ''
  try {
    const result = await tracker.hunts(props.query || 'page_size=20', { signal: controller.signal })
    if (current !== generation) return
    rows.value = result.data
    total.value = result.meta.total
    currentPage.value = result.meta.page
    pageSize.value = result.meta.page_size
  } catch (e) {
    if (current === generation) {
      rows.value = []
      total.value = 0
      error.value = errorText(e)
    }
  } finally {
    if (current === generation) loading.value = false
  }
}
function filter(page = 1) {
  const query = new URLSearchParams({
    page: String(page),
    page_size: String(pageSize.value),
    sort: 'created_at',
    order: 'desc',
  })
  if (search.value.trim()) query.set('q', search.value.trim())
  if (status.value) query.set('status', status.value)
  const target = `/hunts?${query}`
  if (location.hash.slice(1) === target) void load()
  else go(target)
}
function reset() {
  search.value = ''
  status.value = ''
  pageSize.value = 20
  filter()
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
      <p class="eyebrow">旅程记录</p>
      <h2>刷闪任务</h2>
      <p class="muted">记录尝试，也留住每一次相遇。</p>
    </div>
    <ElButton tag="a" href="#/hunts/new" type="primary">＋ 新建任务</ElButton>
  </header>
  <CatalogNotice /><Feedback :message="catalogError" retry @retry="loadCatalog" />
  <section class="surface list-surface" aria-label="刷闪任务列表" :aria-busy="loading">
    <form class="filter-bar" @submit.prevent="filter()">
      <ElInput
        v-model="search"
        aria-label="搜索任务"
        placeholder="搜索物种、游戏、方法或备注"
        clearable
      />
      <ElSelect v-model="status" aria-label="任务状态筛选" placeholder="全部状态" clearable
        ><ElOption v-for="(text, key) in statusLabels" :key="key" :value="key" :label="text"
      /></ElSelect>
      <ElButton native-type="submit" :disabled="loading">查询</ElButton
      ><ElButton @click="reset">重置</ElButton>
    </form>
    <Feedback :message="error" retry @retry="load" />
    <ElSkeleton v-if="loading" :rows="5" animated class="loading-block" />
    <template v-else-if="!error">
      <ElEmpty
        v-if="!rows.length"
        :description="total ? '这一页暂无任务，请返回上一页。' : '暂无符合条件的刷闪任务'"
        ><ElButton tag="a" href="#/hunts/new" type="primary">创建第一条任务</ElButton></ElEmpty
      >
      <div v-else class="table-scroll">
        <ElTable :data="rows" row-key="id" style="width: 100%" :min-width="800">
          <ElTableColumn label="目标" min-width="190"
            ><template #default="{ row }"
              ><a class="record-link" :href="`#/hunts/${row.id}`">{{
                referenceName(catalog.species, row.species_id)
              }}</a>
              <div class="cell-secondary">
                {{ row.form_id ? referenceName(catalog.forms, row.form_id) : '形态未记录' }} · #{{
                  row.id
                }}
              </div></template
            ></ElTableColumn
          >
          <ElTableColumn label="游戏 / 方法" min-width="175"
            ><template #default="{ row }"
              >{{ referenceName(catalog.games, row.game_id) }}
              <div class="cell-secondary">{{ row.method ?? '方法未记录' }}</div></template
            ></ElTableColumn
          >
          <ElTableColumn label="累计次数" min-width="140"
            ><template #default="{ row }">{{
              row.encounter_count === null ? '未记录' : `${row.encounter_count} ${row.count_unit}`
            }}</template></ElTableColumn
          >
          <ElTableColumn label="累计耗时" min-width="125"
            ><template #default="{ row }">{{
              duration(row.duration_seconds)
            }}</template></ElTableColumn
          >
          <ElTableColumn label="状态" width="105"
            ><template #default="{ row }"><HuntStatusTag :value="row.status" /></template
          ></ElTableColumn>
          <ElTableColumn label="操作" width="90"
            ><template #default="{ row }"
              ><a :href="`#/hunts/${row.id}/edit`" :aria-label="`编辑任务 #${row.id}`"
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
