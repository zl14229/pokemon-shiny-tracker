<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import { ElButton, ElEmpty, ElPagination, ElSkeleton, ElTable, ElTableColumn } from 'element-plus'
import { tracker, huntInput } from '../api/tracker'
import { ApiError } from '../api/client'
import type { Hunt, HuntStatus, ShinyRecord } from '../api/models'
import {
  duration,
  referenceName,
  sourceLabels,
  holdingLabels,
  statusLabels,
} from '../domain/format'
import { useCatalog, errorText } from '../composables/resources'
import Feedback from '../components/Feedback.vue'
import HuntStatusTag from '../components/HuntStatusTag.vue'
import CatalogNotice from '../components/CatalogNotice.vue'
const props = defineProps<{ id: number }>()
const { catalog, catalogError, loadCatalog } = useCatalog()
const row = shallowRef<Hunt>(),
  loading = ref(true),
  error = ref(''),
  changing = ref(false),
  actionError = ref('')
const results = shallowRef<ShinyRecord[]>([]),
  resultsTotal = ref(0),
  resultsPage = ref(1),
  resultsLoading = ref(false),
  resultsError = ref('')
const controller = new AbortController()
let alive = true,
  resultsGeneration = 0
async function load() {
  loading.value = true
  error.value = ''
  actionError.value = ''
  try {
    const h = await tracker.hunt(props.id, { signal: controller.signal })
    if (alive) row.value = h
  } catch (e) {
    if (alive) error.value = errorText(e)
  } finally {
    if (alive) loading.value = false
  }
}
async function loadResults(page = 1) {
  const current = ++resultsGeneration
  resultsLoading.value = true
  resultsError.value = ''
  try {
    const result = await tracker.results(props.id, page, { signal: controller.signal })
    if (alive && current === resultsGeneration) {
      results.value = result.data
      resultsTotal.value = result.meta.total
      resultsPage.value = result.meta.page
    }
  } catch (e) {
    if (alive && current === resultsGeneration) resultsError.value = errorText(e)
  } finally {
    if (alive && current === resultsGeneration) resultsLoading.value = false
  }
}
function refresh() {
  void load()
  void loadResults()
}
async function change(status: HuntStatus) {
  if (!row.value || changing.value) return
  changing.value = true
  actionError.value = ''
  try {
    const h = await tracker.updateHunt(
      row.value.id,
      { ...huntInput(row.value), status },
      row.value.updated_at,
    )
    if (alive) row.value = h
  } catch (e) {
    if (alive)
      actionError.value =
        e instanceof ApiError && e.code === 'conflict' && status === 'completed'
          ? '完成任务需要至少一个有效关联成果，请先登记成果后刷新。'
          : errorText(e)
  } finally {
    if (alive) changing.value = false
  }
}
onMounted(() => {
  void loadCatalog()
  void load()
  void loadResults()
})
onBeforeUnmount(() => {
  alive = false
  ++resultsGeneration
  controller.abort()
})
</script>
<template>
  <a class="back-link" href="#/hunts">← 返回任务列表</a
  ><Feedback :message="error" retry @retry="load" />
  <ElSkeleton v-if="loading" :rows="7" animated />
  <template v-else-if="row && !error">
    <header class="page-heading">
      <div>
        <p class="eyebrow">任务 #{{ row.id }}</p>
        <h2>{{ referenceName(catalog.species, row.species_id) }}</h2>
        <p class="muted">
          {{ referenceName(catalog.games, row.game_id) }} · {{ row.method ?? '方法未记录' }}
        </p>
      </div>
      <div class="heading-actions">
        <HuntStatusTag :value="row.status" /><ElButton
          tag="a"
          :href="`#/hunts/${row.id}/edit`"
          :disabled="changing"
          >编辑任务</ElButton
        >
      </div>
    </header>
    <CatalogNotice />
    <Feedback :message="catalogError" retry @retry="loadCatalog" />
    <section class="surface">
      <h3>任务信息</h3>
      <dl class="detail-grid">
        <div>
          <dt>目标物种</dt>
          <dd>{{ referenceName(catalog.species, row.species_id) }}</dd>
        </div>
        <div>
          <dt>目标形态</dt>
          <dd>{{ referenceName(catalog.forms, row.form_id) }}</dd>
        </div>
        <div>
          <dt>目标游戏</dt>
          <dd>{{ referenceName(catalog.games, row.game_id) }}</dd>
        </div>
        <div>
          <dt>方法</dt>
          <dd>{{ row.method ?? '未记录' }}</dd>
        </div>
        <div>
          <dt>累计次数</dt>
          <dd>
            {{
              row.encounter_count === null ? '未记录' : `${row.encounter_count} ${row.count_unit}`
            }}
          </dd>
        </div>
        <div>
          <dt>累计耗时</dt>
          <dd>{{ duration(row.duration_seconds) }}</dd>
        </div>
        <div>
          <dt>开始日期</dt>
          <dd>{{ row.start_date ?? '未记录' }}</dd>
        </div>
        <div>
          <dt>结束日期</dt>
          <dd>{{ row.end_date ?? '未记录' }}</dd>
        </div>
      </dl>
      <div class="note-block">
        <h4>备注</h4>
        <p>{{ row.notes ?? '未记录' }}</p>
      </div>
      <Feedback :message="actionError" />
      <div class="form-actions">
        <ElButton
          v-for="(text, key) in statusLabels"
          :key="key"
          :disabled="
            changing ||
            row.status === key ||
            (key === 'completed' && (!resultsTotal || resultsLoading || Boolean(resultsError)))
          "
          @click="change(key)"
          >{{
            key === 'completed' ? '标为完成' : key === 'abandoned' ? '放弃任务' : '重新进行'
          }}</ElButton
        ><ElButton :disabled="changing" @click="refresh">刷新详情</ElButton>
      </div>
      <p v-if="!resultsTotal" class="field-hint">
        至少登记一个关联成果后才能完成。放弃任务不会删除已有成果。
      </p>
    </section>
    <section class="surface">
      <div class="section-heading">
        <h3>关联成果</h3>
        <ElButton tag="a" :href="`#/collection/new?hunt_id=${id}`" type="primary"
          >登记关联成果</ElButton
        >
        <span class="muted">{{ resultsTotal }} 个有效个体</span>
      </div>
      <Feedback :message="resultsError" retry @retry="loadResults(resultsPage)" /><ElSkeleton
        v-if="resultsLoading"
        :rows="3"
        animated
      /><template v-else-if="!resultsError"
        ><ElEmpty
          v-if="!results.length"
          description="尚无关联成果，进行中的任务也可以拥有多个成果。"
          :image-size="70" />
        <div v-else class="table-scroll">
          <ElTable :data="results" row-key="id"
            ><ElTableColumn label="个体" min-width="170"
              ><template #default="{ row: result }"
                ><a class="record-link" :href="`#/collection/${result.id}`">{{
                  referenceName(catalog.species, result.current_species_id)
                }}</a>
                <div class="cell-secondary">
                  #{{ result.id }} · {{ result.nickname ?? '昵称未记录' }}
                </div></template
              ></ElTableColumn
            ><ElTableColumn label="获得游戏" min-width="150"
              ><template #default="{ row: result }">{{
                referenceName(catalog.games, result.original_game_id)
              }}</template></ElTableColumn
            ><ElTableColumn label="来源" min-width="95"
              ><template #default="{ row: result }">{{
                sourceLabels[(result as ShinyRecord).source]
              }}</template></ElTableColumn
            ><ElTableColumn label="持有状态" min-width="100"
              ><template #default="{ row: result }">{{
                holdingLabels[(result as ShinyRecord).holding_status]
              }}</template></ElTableColumn
            ></ElTable
          >
        </div>
        <div v-if="resultsTotal > 20" class="pagination">
          <ElPagination
            :total="resultsTotal"
            :page-size="20"
            :current-page="resultsPage"
            layout="prev, pager, next"
            @current-change="loadResults"
          /></div
      ></template>
    </section>
  </template>
</template>
