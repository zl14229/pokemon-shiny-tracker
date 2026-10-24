<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref, shallowRef } from 'vue'
import { ElButton, ElForm, ElFormItem, ElInput, ElOption, ElSelect, ElSkeleton } from 'element-plus'
import { tracker } from '../api/tracker'
import type { Hunt } from '../api/models'
import { huntDraft, validateHunt, type Errors } from '../domain/forms'
import { statusLabels } from '../domain/format'
import { go } from '../domain/navigation'
import { useCatalog, errorText, uncertainWrite } from '../composables/resources'
import Feedback from '../components/Feedback.vue'
import ReferenceSelect from '../components/ReferenceSelect.vue'
import CatalogNotice from '../components/CatalogNotice.vue'
const props = defineProps<{ id?: number }>()
const { catalog, catalogLoading, catalogError, loadCatalog } = useCatalog()
const draft = reactive(huntDraft()),
  original = shallowRef<Hunt>(),
  errors = ref<Errors>({})
const loading = ref(Boolean(props.id)),
  loadError = ref(''),
  saveError = ref(''),
  saving = ref(false),
  uncertain = ref(false)
const controller = new AbortController()
let alive = true
async function load() {
  if (!props.id) return
  loading.value = true
  loadError.value = ''
  try {
    const h = await tracker.hunt(props.id, { signal: controller.signal })
    if (alive) {
      original.value = h
      Object.assign(draft, huntDraft(h))
    }
  } catch (e) {
    if (alive) loadError.value = errorText(e)
  } finally {
    if (alive) loading.value = false
  }
}
function species(value: number | null) {
  if (value !== draft.species_id) draft.form_id = null
  draft.species_id = value
}
async function save() {
  if (saving.value || uncertain.value) return
  const result = validateHunt(draft)
  errors.value = result.errors
  saveError.value = ''
  if (!result.input) {
    saveError.value = '请检查表单中的输入。'
    return
  }
  saving.value = true
  try {
    const saved = original.value
      ? await tracker.updateHunt(original.value.id, result.input, original.value.updated_at)
      : await tracker.createHunt(result.input)
    if (alive) go(`/hunts/${saved.id}`)
  } catch (e) {
    if (alive) {
      uncertain.value = uncertainWrite(e)
      saveError.value = uncertain.value
        ? '保存结果暂未确认。请先查看任务列表确认是否已保存，避免重复创建。'
        : errorText(e)
    }
  } finally {
    if (alive) saving.value = false
  }
}
onMounted(() => {
  void loadCatalog()
  void load()
})
onBeforeUnmount(() => {
  alive = false
  controller.abort()
})
</script>
<template>
  <a class="back-link" :href="id ? `#/hunts/${id}` : '#/hunts'"
    >← {{ id ? '返回任务详情' : '返回任务列表' }}</a
  >
  <header class="page-heading">
    <div>
      <p class="eyebrow">旅程记录</p>
      <h2>{{ id ? '编辑刷闪任务' : '新建刷闪任务' }}</h2>
      <p class="muted">只记录已知信息，留空的字段会保存为未记录。</p>
    </div>
  </header>
  <CatalogNotice /><Feedback :message="catalogError" retry @retry="loadCatalog" /><Feedback
    :message="loadError"
    retry
    @retry="load"
  />
  <ElSkeleton v-if="loading" :rows="8" animated />
  <ElForm v-else-if="!loadError" class="surface editor" label-position="top" @submit.prevent="save">
    <h3>目标与方法</h3>
    <div class="form-grid">
      <ElFormItem label="目标物种（必填）" :error="errors.species_id"
        ><ReferenceSelect
          :model-value="draft.species_id"
          :items="catalog.species"
          label="目标物种"
          required
          :disabled="catalogLoading || saving"
          @update:model-value="species"
      /></ElFormItem>
      <ElFormItem label="目标形态"
        ><ReferenceSelect
          v-model="draft.form_id"
          :items="catalog.forms.filter((f) => f.species_id === draft.species_id)"
          label="目标形态"
          :disabled="!draft.species_id || catalogLoading || saving"
      /></ElFormItem>
      <ElFormItem label="目标游戏"
        ><ReferenceSelect
          v-model="draft.game_id"
          :items="catalog.games.filter((g) => g.kind !== 'storage')"
          label="目标游戏"
          :disabled="catalogLoading || saving"
      /></ElFormItem>
      <ElFormItem label="方法" :error="errors.method"
        ><ElInput
          v-model="draft.method"
          aria-label="方法"
          :disabled="saving"
          placeholder="例如：孵蛋、野外遭遇"
      /></ElFormItem>
    </div>
    <h3>进度记录</h3>
    <p class="field-hint">未知次数与耗时请留空；填写 0 表示已知为零。日期可只填写年或年月。</p>
    <div class="form-grid">
      <ElFormItem label="累计次数" :error="errors.encounter_count"
        ><ElInput
          v-model="draft.encounter_count"
          aria-label="累计次数"
          inputmode="numeric"
          :disabled="saving"
          placeholder="未记录"
      /></ElFormItem>
      <ElFormItem label="计数单位" :error="errors.count_unit"
        ><ElInput
          v-model="draft.count_unit"
          aria-label="计数单位"
          :disabled="saving"
          placeholder="例如：eggs、encounters"
      /></ElFormItem>
      <ElFormItem label="累计耗时（秒）" :error="errors.duration_seconds"
        ><ElInput
          v-model="draft.duration_seconds"
          aria-label="累计耗时（秒）"
          inputmode="numeric"
          :disabled="saving"
          placeholder="未记录"
      /></ElFormItem>
      <ElFormItem label="状态"
        ><ElSelect v-model="draft.status" aria-label="任务状态" :disabled="saving"
          ><ElOption
            v-for="(text, key) in statusLabels"
            :key="key"
            :value="key"
            :label="text"
            :disabled="!id && key === 'completed'" /></ElSelect
      ></ElFormItem>
      <ElFormItem label="开始日期" :error="errors.start_date"
        ><ElInput
          v-model="draft.start_date"
          aria-label="开始日期"
          :disabled="saving"
          placeholder="YYYY / YYYY-MM / YYYY-MM-DD"
      /></ElFormItem>
      <ElFormItem label="结束日期" :error="errors.end_date"
        ><ElInput
          v-model="draft.end_date"
          aria-label="结束日期"
          :disabled="saving"
          placeholder="YYYY / YYYY-MM / YYYY-MM-DD"
      /></ElFormItem>
    </div>
    <p class="field-hint">完成任务需要至少一个有效关联成果；放弃任务会保留已有成果。</p>
    <ElFormItem label="备注" :error="errors.notes"
      ><ElInput
        v-model="draft.notes"
        aria-label="任务备注"
        :disabled="saving"
        type="textarea"
        :rows="4"
        placeholder="记录方法、地点或这段旅程的细节……"
    /></ElFormItem>
    <Feedback :message="saveError" />
    <div class="form-actions">
      <ElButton
        native-type="submit"
        type="primary"
        :loading="saving"
        :disabled="uncertain || catalogLoading || Boolean(catalogError)"
        >保存任务</ElButton
      ><ElButton
        v-if="!saving"
        tag="a"
        :href="uncertain ? '#/hunts' : id ? `#/hunts/${id}` : '#/hunts'"
        >{{ uncertain ? '检查任务列表' : '取消' }}</ElButton
      >
    </div>
  </ElForm>
</template>
