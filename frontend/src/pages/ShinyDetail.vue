<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import { ElButton, ElEmpty, ElPagination, ElSkeleton, ElTag } from 'element-plus'
import type { ShinyRecord, History } from '../api/models'
import { tracker } from '../api/tracker'
import { useCatalog, errorText } from '../composables/resources'
import { referenceName, holdingLabels, sourceLabels } from '../domain/format'
import { historyLabels } from '../domain/shiny-form'
import AttachmentsPanel from '../components/AttachmentsPanel.vue'
import CatalogNotice from '../components/CatalogNotice.vue'
import Feedback from '../components/Feedback.vue'
const props = defineProps<{ id: number }>()
const { catalog, catalogError, loadCatalog } = useCatalog()
const row = shallowRef<ShinyRecord>(),
  loading = ref(true),
  error = ref(''),
  history = shallowRef<History[]>([]),
  historyError = ref(''),
  historyLoading = ref(false),
  historyTotal = ref(0),
  historyPage = ref(1)
let alive = true,
  historyGeneration = 0
const controller = new AbortController()
async function load() {
  loading.value = true
  error.value = ''
  try {
    const s = await tracker.shiny(props.id, { signal: controller.signal })
    if (alive) row.value = s
  } catch (e) {
    if (alive) error.value = errorText(e)
  } finally {
    if (alive) loading.value = false
  }
}
async function loadHistory(page = 1) {
  const current = ++historyGeneration
  historyLoading.value = true
  historyError.value = ''
  try {
    const result = await tracker.history(props.id, page, { signal: controller.signal })
    if (alive && current === historyGeneration) {
      history.value = result.data
      historyTotal.value = result.meta.total
      historyPage.value = page
    }
  } catch (e) {
    if (alive && current === historyGeneration) historyError.value = errorText(e)
  } finally {
    if (alive && current === historyGeneration) historyLoading.value = false
  }
}
const recordedTime = (value: string) => new Date(value).toLocaleString('zh-CN')
onMounted(() => {
  void loadCatalog()
  void load()
  void loadHistory()
})
onBeforeUnmount(() => {
  alive = false
  ++historyGeneration
  controller.abort()
})
</script>
<template>
  <a class="back-link" href="#/collection">← 返回收藏列表</a
  ><Feedback :message="error" retry @retry="load" /><ElSkeleton v-if="loading" :rows="7" animated />
  <template v-else-if="row && !error">
    <header class="page-heading">
      <div>
        <p class="eyebrow">个体 #{{ row.id }}</p>
        <h2>{{ row.nickname ?? referenceName(catalog.species, row.current_species_id) }}</h2>
        <p class="muted">
          {{ referenceName(catalog.species, row.current_species_id) }} ·
          {{ sourceLabels[row.source] }}
        </p>
      </div>
      <div class="heading-actions">
        <ElTag :type="row.holding_status === 'held' ? 'success' : 'info'">{{
          holdingLabels[row.holding_status]
        }}</ElTag
        ><ElButton tag="a" :href="`#/collection/${row.id}/edit`">编辑个体</ElButton>
      </div>
    </header>
    <CatalogNotice /><Feedback :message="catalogError" retry @retry="loadCatalog" />
    <div class="detail-panels">
      <section class="surface">
        <h3>原始获得信息</h3>
        <dl class="detail-grid">
          <div>
            <dt>原始物种</dt>
            <dd>{{ referenceName(catalog.species, row.original_species_id) }}</dd>
          </div>
          <div>
            <dt>原始形态</dt>
            <dd>{{ referenceName(catalog.forms, row.original_form_id) }}</dd>
          </div>
          <div>
            <dt>原始获得游戏</dt>
            <dd>{{ referenceName(catalog.games, row.original_game_id) }}</dd>
          </div>
          <div>
            <dt>获得日期</dt>
            <dd>{{ row.obtained_date ?? '未记录' }}</dd>
          </div>
          <div>
            <dt>来源</dt>
            <dd>{{ sourceLabels[row.source] }}</dd>
          </div>
          <div>
            <dt>关联任务</dt>
            <dd>
              <a v-if="row.hunt_id !== null" :href="`#/hunts/${row.hunt_id}`"
                >查看 Hunt #{{ row.hunt_id }}</a
              ><span v-else>未关联 Hunt</span>
            </dd>
          </div>
        </dl>
      </section>
      <section class="surface">
        <h3>当前个体与保存位置</h3>
        <dl class="detail-grid">
          <div>
            <dt>当前物种</dt>
            <dd>{{ referenceName(catalog.species, row.current_species_id) }}</dd>
          </div>
          <div>
            <dt>当前形态</dt>
            <dd>{{ referenceName(catalog.forms, row.current_form_id) }}</dd>
          </div>
          <div>
            <dt>当前保存游戏</dt>
            <dd>{{ referenceName(catalog.games, row.current_game_id) }}</dd>
          </div>
          <div>
            <dt>当前保存位置</dt>
            <dd>{{ row.current_location ?? '未记录' }}</dd>
          </div>
          <div>
            <dt>持有状态</dt>
            <dd>{{ holdingLabels[row.holding_status] }}</dd>
          </div>
          <div>
            <dt>昵称</dt>
            <dd>{{ row.nickname ?? '未记录' }}</dd>
          </div>
        </dl>
      </section>
    </div>
    <section class="surface">
      <h3>个体信息</h3>
      <dl class="detail-grid">
        <div
          v-for="field in [
            { key: 'ball', label: '球种' },
            { key: 'gender', label: '性别' },
            { key: 'nature', label: '性格' },
            { key: 'ability', label: '特性' },
            { key: 'level', label: '等级' },
          ] as const"
          :key="field.key"
        >
          <dt>{{ field.label }}</dt>
          <dd>{{ row[field.key] ?? '未记录' }}</dd>
        </div>
      </dl>
      <div class="note-block">
        <h4>备注</h4>
        <p>{{ row.notes ?? '未记录' }}</p>
      </div>
    </section>
    <AttachmentsPanel :id="row.id" />
    <section class="surface">
      <div class="section-heading">
        <h3>变更记录</h3>
        <span class="muted">同一个体的轻量历史</span>
      </div>
      <Feedback :message="historyError" retry @retry="loadHistory(historyPage)" /><ElSkeleton
        v-if="historyLoading"
        :rows="3"
        animated
      />
      <template v-else-if="!historyError"
        ><ElEmpty v-if="!history.length" description="暂无变更记录" :image-size="70" />
        <ol v-else class="history-list">
          <li v-for="event in history" :key="event.id">
            <strong>{{ historyLabels[event.kind] ?? '记录变更' }}</strong>
            <p>{{ event.event_date ? `事件日期：${event.event_date}` : '事件日期未记录' }}</p>
            <p v-if="event.note" class="history-note">{{ event.note }}</p>
            <small>系统记录时间：{{ recordedTime(event.created_at) }}</small>
          </li>
        </ol>
        <ElPagination
          v-if="historyTotal > 20"
          :current-page="historyPage"
          :page-size="20"
          :total="historyTotal"
          :pager-count="5"
          layout="prev, pager, next"
          @current-change="loadHistory"
      /></template>
    </section>
  </template>
</template>
