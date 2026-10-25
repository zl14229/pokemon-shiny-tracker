<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { ElButton, ElAlert } from 'element-plus'
import DataNav from '../components/DataNav.vue'
import { api } from '../api/client'
import { errorText } from '../composables/resources'
const busy = ref('')
const error = ref('')
const notice = ref('')
const controller = new AbortController()
async function download(kind: 'export' | 'backup') {
  if (busy.value) return
  busy.value = kind
  error.value = ''
  notice.value = ''
  try {
    const blob = await api.download(
      `/data/${kind}`,
      kind === 'backup' ? 'application/zip' : 'application/json',
      { signal: controller.signal },
    )
    const href = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = href
    link.download = kind === 'backup' ? 'shiny-backup.zip' : 'shiny-data.json'
    document.body.append(link)
    link.click()
    link.remove()
    setTimeout(() => URL.revokeObjectURL(href), 10000)
    notice.value = '文件已生成并交给浏览器下载，请保存到私有位置。ZIP可用本地verify命令再次校验。'
  } catch (e) {
    error.value = errorText(e)
  } finally {
    busy.value = ''
  }
}
onBeforeUnmount(() => controller.abort())
</script>
<template>
  <header class="page-heading">
    <div>
      <p class="eyebrow">DATA MANAGEMENT</p>
      <h2>导出与备份</h2>
      <p class="muted">保留你的记录、历史与附件，恢复时始终使用新的隔离目录。</p>
    </div>
  </header>
  <DataNav active="backup" /><ElAlert
    title="下载包含个人信息且未加密，请保存到私有位置。不会上传到外部服务，也不会覆盖当前数据库。"
    type="info"
    :closable="false"
  />
  <p v-if="error" role="alert" class="error-text">{{ error }}</p>
  <p v-if="notice" role="status">{{ notice }}</p>
  <section class="surface">
    <h3>数据导出 · JSON</h3>
    <p>
      导出全部应用表与字段，包括
      Hunt、个体、轻量历史、导入批次/原值、附件元数据与软删除状态。保留未知值与稳定ID，不补造记录。
    </p>
    <p class="muted">JSON 不含附件原件，不能代替完整恢复备份；本版不从JSON恢复。</p>
    <ElButton :loading="busy === 'export'" :disabled="!!busy" @click="download('export')"
      >下载 JSON 数据</ElButton
    >
  </section>
  <section class="surface">
    <h3>完整备份 · ZIP</h3>
    <p>
      一致性SQLite快照、数据库引用的本地文件与SHA256清单；包含软删除附件，校验缺失或损坏时整个备份失败。外部网页/视频内容、原始Excel与孤立文件不包含。
    </p>
    <p class="muted">
      数据库上限512MiB，归档解压总量上限2GiB。大型文件请使用本地命令；只有生成并校验完成才提供下载。
    </p>
    <ElButton
      type="primary"
      :loading="busy === 'backup'"
      :disabled="!!busy"
      @click="download('backup')"
      >下载 ZIP 备份</ElButton
    >
  </section>
  <section class="surface">
    <h3>验证与隔离恢复</h3>
    <p>
      先用本地 shiny-data verify 检查ZIP，再用 restore
      指定一个不存在的新目录；已有目标一律拒绝。恢复会再次核对清单、数据库完整性、外键与附件，然后原子生成新目录。
    </p>
    <p>
      此操作不切换正在运行的应用、不覆盖当前记录。仅 Debian/Linux
      的安全原子恢复已支持；Windows原生尚未验收。
    </p>
    <p class="muted">
      恢复命令需明确填写自己的绝对路径，完整步骤见仓库
      docs/BACKUP.md。恢复后先在隔离位置检查记录与附件，再决定后续使用；本界面没有覆盖恢复入口。
    </p>
  </section>
</template>
