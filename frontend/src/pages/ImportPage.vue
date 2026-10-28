<script setup lang="ts">
import { computed, ref, onBeforeUnmount, onMounted } from 'vue'
import {
  ElAlert,
  ElButton,
  ElCheckbox,
  ElInput,
  ElInputNumber,
  ElOption,
  ElSelect,
  ElTable,
  ElTableColumn,
} from 'element-plus'
import {
  importBatches,
  batchRecords,
  inspectWorkbook,
  previewWorkbook,
  confirmImport,
  discardPreview,
  type ExcelSheet,
  type Mapping,
  type Preview,
} from '../api/imports'
import { useCatalog } from '../composables/resources'
import DataNav from '../components/DataNav.vue'
import CatalogNotice from '../components/CatalogNotice.vue'
import { ApiError } from '../api/client'
const { catalog, catalogError, loadCatalog } = useCatalog()
onMounted(loadCatalog)
const file = ref<File>()
const fileControl = ref<HTMLInputElement>()
const sheets = ref<ExcelSheet[]>([])
const sheet = ref('')
const header = ref(1)
const columns = ref<Record<string, string>>({})
const valueText = ref('{}')
const preview = ref<Preview>()
const selected = ref<number[]>([])
const ack = ref(false)
const busy = ref(false)
const error = ref('')
const result = ref<{
  batch: { id: number }
  rows: { row_number: number; shiny_record_id: number }[]
}>()
const page = ref(1)
const uncertain = ref(false)
const batches = ref<Awaited<ReturnType<typeof importBatches>>>([])
const traced = ref<Awaited<ReturnType<typeof batchRecords>>>([])
async function showBatches() {
  busy.value = true
  error.value = ''
  try {
    batches.value = await importBatches()
  } catch (e) {
    error.value = message(e)
  } finally {
    busy.value = false
  }
}
async function showTrace(id: number) {
  busy.value = true
  try {
    traced.value = await batchRecords(id)
  } catch (e) {
    error.value = message(e)
  } finally {
    busy.value = false
  }
}
const fields: Record<string, string> = {
  '': '仅保留在批次原值',
  current_national_dex: '当前物种全国编号',
  current_species_id: '当前物种内部ID',
  original_national_dex: '原始物种全国编号',
  original_species_id: '原始物种内部ID',
  current_form_id: '当前形态内部ID',
  original_form_id: '原始形态内部ID',
  original_game_id: '原始获得游戏内部ID',
  current_game_id: '当前游戏内部ID',
  current_location: '当前保存位置',
  obtained_date: '获得日期（ISO）',
  source: '来源（需映射枚举）',
  holding_status: '持有状态（需映射枚举）',
  nickname: '昵称',
  gender: '性别',
  nature: '性格',
  ability: '特性',
  ball: '球种',
  level: '等级',
  notes: '备注（多列拼接原值）',
}
const chosen = computed(() => sheets.value.find((s) => s.name === sheet.value))
const headers = computed(
  () => chosen.value?.rows.find((r) => r.number === header.value)?.cells ?? {},
)
const rows = computed(() => preview.value?.rows.slice((page.value - 1) * 50, page.value * 50) ?? [])
function message(e: unknown) {
  return e instanceof ApiError ? e.message : '格式无效或操作失败，请检查映射后重试。'
}
async function clearPreview() {
  if (preview.value) {
    try {
      await discardPreview(preview.value.token)
    } catch {
      /* expires automatically */
    }
  }
  preview.value = undefined
  selected.value = []
  ack.value = false
  uncertain.value = false
  page.value = 1
}
async function changeFile(e: Event) {
  if (busy.value) return
  busy.value = true
  await clearPreview()
  sheets.value = []
  result.value = undefined
  columns.value = {}
  file.value = (e.target as HTMLInputElement).files?.[0]
  if (!file.value) {
    busy.value = false
    return
  }
  busy.value = true
  error.value = ''
  try {
    sheets.value = await inspectWorkbook(file.value)
    sheet.value = sheets.value[0]?.name ?? ''
  } catch (e) {
    error.value = message(e)
  } finally {
    busy.value = false
  }
}
async function generate() {
  if (!file.value || busy.value) return
  busy.value = true
  await clearPreview()
  busy.value = true
  error.value = ''
  result.value = undefined
  try {
    const values: unknown = JSON.parse(valueText.value)
    if (typeof values !== 'object' || values === null || Array.isArray(values))
      throw new Error('mapping')
    const mapping: Mapping = {
      sheet: sheet.value,
      header_row: header.value,
      columns: Object.fromEntries(Object.entries(columns.value).filter(([, v]) => v)),
      values: values as Mapping['values'],
    }
    preview.value = await previewWorkbook(file.value, mapping)
  } catch (e) {
    error.value = message(e)
  } finally {
    busy.value = false
  }
}
function choose(number: number, value: unknown) {
  selected.value = value ? [...selected.value, number] : selected.value.filter((n) => n !== number)
}
function selectVisible() {
  selected.value = [
    ...new Set([
      ...selected.value,
      ...rows.value.filter((r) => !r.errors.length).map((r) => r.number),
    ]),
  ].slice(0, 500)
}
async function confirm() {
  if (!preview.value) return
  busy.value = true
  error.value = ''
  try {
    result.value = await confirmImport(preview.value.token, selected.value, ack.value)
    preview.value = undefined
    selected.value = []
    uncertain.value = false
    await showBatches()
  } catch (e) {
    error.value = message(e)
    if (!(e instanceof ApiError) || e.kind !== 'http' || (e.status ?? 0) >= 500) {
      uncertain.value = true
      error.value += ' 确认结果不明；可用同一令牌和相同行安全重试，勿重新预览导入。'
    }
  } finally {
    busy.value = false
  }
}
onBeforeUnmount(() => {
  if (!uncertain.value) void clearPreview()
})
</script>
<template>
  <header class="page-heading">
    <div>
      <p class="eyebrow">DATA MANAGEMENT</p>
      <h2>Excel 历史记录导入</h2>
      <p class="muted">先映射与预览，再选择明确的个体行；每批最多500条，不创建 Hunt。</p>
    </div>
  </header>
  <DataNav active="import" />
  <ElAlert
    title="文件只读，不执行公式或外部链接。未知日期、来源、形态及原始信息不会自动补造。预览30分钟失效；确认后仅新增个体，不覆盖已有记录。"
    type="info"
    :closable="false"
  />
  <CatalogNotice />
  <details class="surface">
    <summary>本地可选资料与内部ID（不代表可刷性）</summary>
    <p v-if="catalogError">{{ catalogError }}</p>
    <p>
      物种：<span v-for="s in catalog.species" :key="s.id"
        >{{ s.name_zh || s.name }} · ID {{ s.id }} / 全国 {{ s.national_dex }}；</span
      >
    </p>
    <p>
      形态：<span v-for="f in catalog.forms" :key="f.id"
        >{{ f.name_zh || f.name }} · ID {{ f.id }} / 物种ID {{ f.species_id }}；</span
      >
    </p>
    <p>
      游戏：<span v-for="g in catalog.games" :key="g.id"
        >{{ g.name_zh || g.name }} · ID {{ g.id }}；</span
      >
    </p>
  </details>
  <p v-if="error" role="alert" class="error-text">{{ error }}</p>
  <section v-if="result" class="surface">
    <h3>导入成功</h3>
    <p>
      批次 #{{ result.batch.id }}，已核对 {{ result.rows.length }} 条个体与来源行。<a
        href="#/collection"
        >查看闪光收藏</a
      >
    </p>
  </section>
  <section class="surface" :aria-busy="busy">
    <div>
      <p>选择 XLSX（上限8MiB）</p>
      <ElButton :disabled="busy || uncertain" @click="fileControl?.click()">选择文件</ElButton>
      <input
        ref="fileControl"
        aria-label="选择 Excel 文件"
        type="file"
        accept=".xlsx"
        hidden
        :disabled="busy || uncertain"
        @change="changeFile"
      />
      <span class="muted"> {{ file?.name || '尚未选择文件' }}</span>
    </div>
    <template v-if="sheets.length">
      <div class="form-grid">
        <label
          >工作表<ElSelect v-model="sheet" aria-label="工作表" :disabled="busy || !!preview"
            ><ElOption
              v-for="s in sheets"
              :key="s.name"
              :label="`${s.name}（${s.rows.length}行）`"
              :value="s.name" /></ElSelect></label
        ><label
          >表头行<ElInputNumber
            v-model="header"
            aria-label="表头行"
            :min="1"
            :max="100000"
            :disabled="busy || !!preview"
        /></label>
      </div>
      <p>
        逐列确认语义。编号是全国编号或内部ID，不能混用；工作表名不自动作为获得游戏。姓名与形态不能证明进化前信息。“次数”可映射备注，不重建
        Hunt。未映射的非空值会警告并保留于批次溯源。
      </p>
      <div v-for="(cell, col) in headers" :key="col" class="form-grid">
        <label
          >{{ col }} · {{ cell.value || '无表头'
          }}<ElSelect
            v-model="columns[col]"
            :aria-label="`映射列${col}`"
            :disabled="busy || !!preview"
            ><ElOption
              v-for="(name, key) in fields"
              :key="key"
              :label="name"
              :value="key" /></ElSelect
        ></label>
      </div>
      <label class="mapping-help"
        >可选值映射（JSON；如 source 的原值映射到
        self_hunted/trade/event/other/unknown，holding_status 映射到
        held/traded/released/unknown；游戏/形态映射到经核对的内部ID）<ElInput
          v-model="valueText"
          aria-label="值映射JSON"
          type="textarea"
          :rows="3"
          :disabled="busy || !!preview"
      /></label>
      <ElButton
        v-if="!preview"
        type="primary"
        :loading="busy"
        :disabled="busy || uncertain"
        @click="generate"
        >生成只读预览</ElButton
      ><ElButton v-else :disabled="busy || uncertain" @click="clearPreview"
        >清除预览重新映射</ElButton
      >
    </template>
  </section>
  <section v-if="preview" class="surface">
    <h3>预览与逐行校验</h3>
    <p>
      共 {{ preview.rows.length }} 行，已选
      {{ selected.length }} 行。未选行不会导入。只选实际获得的个体，空白模板与收藏规划不是获得记录。
    </p>
    <div class="table-scroll">
      <ElTable :data="rows" :fit="false" row-key="number"
        ><ElTableColumn label="选择" width="70"
          ><template #default="{ row }"
            ><ElCheckbox
              :aria-label="`选择第${row.number}行`"
              :model-value="selected.includes(row.number)"
              :disabled="busy || uncertain || row.errors.length > 0"
              @change="choose(row.number, $event)" /></template></ElTableColumn
        ><ElTableColumn prop="number" label="原始行" width="80" /><ElTableColumn
          label="映射结果"
          width="360"
          ><template #default="{ row }">
            <p>当前物种ID：{{ row.input.current_species_id }}</p>
            <p>获得日期：{{ row.input.obtained_date || '未知' }}</p>
            <p>来源 / 持有：{{ row.input.source }} / {{ row.input.holding_status }}</p>
            <details>
              <summary>展开全部映射字段</summary>
              <pre>{{ JSON.stringify(row.input, null, 2) }}</pre>
            </details>
          </template></ElTableColumn
        ><ElTableColumn label="错误与警告" width="400"
          ><template #default="{ row }"
            ><p v-for="e in row.errors" :key="e" class="error-text">{{ e }}</p>
            <p v-for="w in row.warnings" :key="w">{{ w }}</p>
            <span v-if="!row.errors.length && !row.warnings.length"
              >可导入（仍须人工核对）</span
            ></template
          ></ElTableColumn
        ><ElTableColumn label="原始单元格" width="330"
          ><template #default="{ row }"
            ><p v-for="(c, k) in row.raw" :key="k">
              {{ k }}: {{ c.value }} {{ c.formula ? '（公式缓存）' : '' }}
            </p></template
          ></ElTableColumn
        ></ElTable
      >
    </div>
    <div class="list-actions">
      <ElButton :disabled="page <= 1 || busy" @click="page--">上一页</ElButton
      ><span>第{{ page }}页</span
      ><ElButton :disabled="page * 50 >= preview.rows.length || busy" @click="page++"
        >下一页</ElButton
      ><ElButton :disabled="busy || uncertain" @click="selectVisible">选择本页有效行</ElButton>
    </div>
    <ElCheckbox v-model="ack" :disabled="busy || uncertain"
      >我已核对选中行及疑似重复、公式缓存和未映射列；相似记录可能是不同个体，仍确认新增。</ElCheckbox
    >
    <p>
      <ElButton
        type="primary"
        :loading="busy"
        :disabled="busy || selected.length === 0 || selected.length > 500 || !ack"
        @click="confirm"
        >{{ uncertain ? '用相同令牌安全重试' : '确认导入选中个体' }}</ElButton
      >
    </p>
  </section>
  <section class="surface">
    <h3>已确认导入批次</h3>
    <p>结果不明或重新打开页面时，先核对批次与收藏。此处仅显示最近100批；完整数据随备份保留。</p>
    <ElButton :disabled="busy" @click="showBatches">读取最近批次</ElButton>
    <div class="table-scroll">
      <ElTable :data="batches" :fit="false"
        ><ElTableColumn prop="id" label="批次" width="80" /><ElTableColumn
          prop="created_at"
          label="系统导入时间"
          width="250"
        /><ElTableColumn prop="sheet" label="原工作表" width="180" /><ElTableColumn
          prop="selected_json"
          label="选中原行号"
          width="220"
        /><ElTableColumn label="结果" width="120"
          ><template #default="{ row }"
            ><ElButton :disabled="busy" @click="showTrace(row.id)">查看个体</ElButton></template
          ></ElTableColumn
        ></ElTable
      >
    </div>
    <p v-for="r in traced" :key="r.row_number">
      原行 {{ r.row_number }} →
      <a :href="`#/collection/${r.shiny_record_id}`">个体 #{{ r.shiny_record_id }}</a>
    </p>
  </section>
</template>

<style scoped>
.mapping-help {
  display: block;
  max-width: 100%;
  overflow-wrap: anywhere;
}
</style>
