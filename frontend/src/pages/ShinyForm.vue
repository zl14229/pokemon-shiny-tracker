<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, shallowRef } from 'vue'
import { ElButton, ElForm, ElFormItem, ElInput, ElOption, ElSelect, ElSkeleton } from 'element-plus'
import type { ShinyRecord } from '../api/models'
import { tracker } from '../api/tracker'
import { ApiError } from '../api/client'
import { shinyDraft, validateShiny } from '../domain/shiny-form'
import type { Errors } from '../domain/forms'
import { holdingLabels, sourceLabels } from '../domain/format'
import { go } from '../domain/navigation'
import { useCatalog, errorText, uncertainWrite } from '../composables/resources'
import CatalogNotice from '../components/CatalogNotice.vue'
import ReferenceSelect from '../components/ReferenceSelect.vue'
import HuntPicker from '../components/HuntPicker.vue'
import Feedback from '../components/Feedback.vue'
const props = defineProps<{ id?: number; query?: string }>()
const { catalog, catalogLoading, catalogError, loadCatalog } = useCatalog()
const draft = reactive(shinyDraft()),
  original = shallowRef<ShinyRecord>(),
  errors = ref<Errors>({})
const loading = ref(Boolean(props.id)),
  loadError = ref(''),
  saveError = ref(''),
  saving = ref(false),
  uncertain = ref(false)
let alive = true
const controller = new AbortController()
const speciesChanged = computed(
  () => original.value && original.value.current_species_id !== draft.current_species_id,
)
async function load() {
  if (!props.id) {
    const value = new URLSearchParams(props.query).get('hunt_id')
    if (value !== null) {
      const n = Number(value)
      if (!/^\d+$/.test(value) || !Number.isSafeInteger(n) || n <= 0)
        loadError.value = '关联任务编号无效，请从任务详情重新进入。'
      else draft.hunt_id = n
    }
    return
  }
  loading.value = true
  loadError.value = ''
  try {
    const s = await tracker.shiny(props.id, { signal: controller.signal })
    if (alive) {
      original.value = s
      Object.assign(draft, shinyDraft(s))
    }
  } catch (e) {
    if (alive) loadError.value = errorText(e)
  } finally {
    if (alive) loading.value = false
  }
}
function currentSpecies(value: number | null) {
  if (value !== draft.current_species_id) draft.current_form_id = null
  draft.current_species_id = value
}
function originalSpecies(value: number | null) {
  if (value !== draft.original_species_id) draft.original_form_id = null
  draft.original_species_id = value
}
async function save() {
  if (saving.value || uncertain.value) return
  const result = validateShiny(draft, original.value)
  errors.value = result.errors
  saveError.value = ''
  if (!result.input) {
    saveError.value = '请检查表单中的输入。'
    return
  }
  saving.value = true
  try {
    const s = original.value
      ? await tracker.updateShiny(original.value.id, result.input, original.value.updated_at)
      : await tracker.createShiny(result.input)
    if (alive) go(`/collection/${s.id}`)
  } catch (e) {
    if (alive) {
      uncertain.value = uncertainWrite(e)
      saveError.value = uncertain.value
        ? '保存结果暂未确认。请先检查收藏列表，避免重复登记同一个体。'
        : e instanceof ApiError &&
            e.code === 'conflict' &&
            original.value?.hunt_id != null &&
            original.value?.hunt_id !== draft.hunt_id
          ? '无法移走已完成任务的最后有效成果。请先将关联 Hunt 重新进行或放弃，再修改关联。'
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
  <a class="back-link" :href="id ? `#/collection/${id}` : '#/collection'"
    >← {{ id ? '返回个体详情' : '返回收藏列表' }}</a
  >
  <header class="page-heading">
    <div>
      <p class="eyebrow">闪光收藏</p>
      <h2>{{ id ? '编辑闪光个体' : '登记闪光个体' }}</h2>
      <p class="muted">一只个体，一份永久记录。未知的信息可以留空。</p>
    </div>
  </header>
  <CatalogNotice /><Feedback :message="catalogError" retry @retry="loadCatalog" /><Feedback
    :message="loadError"
    retry
    @retry="load"
  /><ElSkeleton v-if="loading" :rows="8" animated />
  <ElForm v-else-if="!loadError" label-position="top" class="surface editor" @submit.prevent="save">
    <h3>当前个体</h3>
    <div class="form-grid">
      <ElFormItem label="当前物种（必填）" :error="errors.current_species_id"
        ><ReferenceSelect
          :model-value="draft.current_species_id"
          :items="catalog.species"
          label="当前物种"
          required
          :disabled="saving || catalogLoading"
          @update:model-value="currentSpecies"
      /></ElFormItem>
      <ElFormItem label="当前形态"
        ><ReferenceSelect
          v-model="draft.current_form_id"
          :items="catalog.forms.filter((f) => f.species_id === draft.current_species_id)"
          label="当前形态"
          :disabled="saving || !draft.current_species_id || catalogLoading"
      /></ElFormItem>
      <ElFormItem label="昵称" :error="errors.nickname"
        ><ElInput v-model="draft.nickname" aria-label="昵称" :disabled="saving"
      /></ElFormItem>
      <ElFormItem label="持有状态"
        ><ElSelect v-model="draft.holding_status" aria-label="持有状态" :disabled="saving"
          ><ElOption
            v-for="(text, key) in holdingLabels"
            :key="key"
            :value="key"
            :label="text" /></ElSelect
      ></ElFormItem>
      <ElFormItem label="当前保存游戏"
        ><ReferenceSelect
          v-model="draft.current_game_id"
          :items="catalog.games"
          label="当前保存游戏"
          :disabled="saving || catalogLoading"
      /></ElFormItem>
      <ElFormItem label="当前保存位置" :error="errors.current_location"
        ><ElInput
          v-model="draft.current_location"
          aria-label="当前保存位置"
          :disabled="saving"
          placeholder="例如：HOME 某个盒子，未知可留空"
      /></ElFormItem>
    </div>
    <h3>原始获得信息</h3>
    <p class="field-hint">与当前个体分开记录。不会根据当前物种或关联任务推断原始资料。</p>
    <div class="form-grid">
      <ElFormItem label="原始获得物种" :error="errors.original_species_id"
        ><ReferenceSelect
          :model-value="draft.original_species_id"
          :items="catalog.species"
          label="原始获得物种"
          :disabled="saving || catalogLoading"
          @update:model-value="originalSpecies"
      /></ElFormItem>
      <ElFormItem label="原始获得形态"
        ><ReferenceSelect
          v-model="draft.original_form_id"
          :items="catalog.forms.filter((f) => f.species_id === draft.original_species_id)"
          label="原始获得形态"
          :disabled="saving || !draft.original_species_id || catalogLoading"
      /></ElFormItem>
      <ElFormItem label="原始获得游戏"
        ><ReferenceSelect
          v-model="draft.original_game_id"
          :items="catalog.games"
          label="原始获得游戏"
          :disabled="saving || catalogLoading"
      /></ElFormItem>
      <ElFormItem label="获得日期" :error="errors.obtained_date"
        ><ElInput
          v-model="draft.obtained_date"
          aria-label="获得日期"
          :disabled="saving"
          placeholder="YYYY / YYYY-MM / YYYY-MM-DD"
      /></ElFormItem>
      <ElFormItem label="来源"
        ><ElSelect v-model="draft.source" aria-label="来源" :disabled="saving"
          ><ElOption
            v-for="(text, key) in sourceLabels"
            :key="key"
            :value="key"
            :label="text" /></ElSelect
      ></ElFormItem>
      <ElFormItem label="关联 Hunt（可选）" :error="errors.hunt_id"
        ><HuntPicker v-model="draft.hunt_id" :catalog="catalog" :disabled="saving"
      /></ElFormItem>
    </div>
    <h3>个体信息</h3>
    <div class="form-grid">
      <ElFormItem
        v-for="field in [
          { key: 'ball', label: '球种' },
          { key: 'gender', label: '性别' },
          { key: 'nature', label: '性格' },
          { key: 'ability', label: '特性' },
        ] as const"
        :key="field.key"
        :label="field.label"
        :error="errors[field.key]"
        ><ElInput
          v-model="draft[field.key]"
          :aria-label="field.label"
          :disabled="saving"
          placeholder="按原始记录填写，未知可留空"
      /></ElFormItem>
      <ElFormItem label="等级" :error="errors.level"
        ><ElInput
          v-model="draft.level"
          aria-label="等级"
          inputmode="numeric"
          :disabled="saving"
          placeholder="1–100，未知可留空"
      /></ElFormItem>
    </div>
    <template v-if="id"
      ><h3>本次变更</h3>
      <p class="field-hint">进化或转移只更新这一个体，不会生成重复个体。事件日期不填时保持未知。</p>
      <div class="form-grid">
        <ElFormItem v-if="speciesChanged" label="物种变化原因（必填）" :error="errors.change_type"
          ><ElSelect
            v-model="draft.change_type"
            aria-label="物种变化原因"
            :disabled="saving"
            placeholder="请选择原因"
            ><ElOption value="evolution" label="进化（用户记录）" /><ElOption
              value="correction"
              label="资料纠错" /></ElSelect
        ></ElFormItem>
        <ElFormItem label="本次事件日期" :error="errors.event_date"
          ><ElInput
            v-model="draft.event_date"
            aria-label="本次事件日期"
            :disabled="saving"
            placeholder="YYYY / YYYY-MM / YYYY-MM-DD"
        /></ElFormItem>
        <ElFormItem label="变更说明" :error="errors.history_note"
          ><ElInput
            v-model="draft.history_note"
            aria-label="变更说明"
            :disabled="saving"
            placeholder="可选，保留在变更记录中"
        /></ElFormItem></div
    ></template>
    <ElFormItem label="备注" :error="errors.notes"
      ><ElInput
        v-model="draft.notes"
        aria-label="个体备注"
        type="textarea"
        :rows="4"
        :disabled="saving"
        placeholder="其他个体信息与这次相遇的故事……"
    /></ElFormItem>
    <Feedback :message="saveError" />
    <div class="form-actions">
      <ElButton
        native-type="submit"
        type="primary"
        :loading="saving"
        :disabled="uncertain || catalogLoading || Boolean(catalogError)"
        >保存个体</ElButton
      ><ElButton
        v-if="!saving"
        tag="a"
        :href="uncertain ? '#/collection' : id ? `#/collection/${id}` : '#/collection'"
        >{{ uncertain ? '检查收藏列表' : '取消' }}</ElButton
      >
    </div>
  </ElForm>
</template>
