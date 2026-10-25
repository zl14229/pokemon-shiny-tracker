<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { ElButton, ElInput, ElPopconfirm, ElEmpty } from 'element-plus'
import {
  listAttachments,
  uploadAttachment,
  linkAttachment,
  deleteAttachment,
  safeURL,
  type Attachment,
} from '../api/attachments'
import { errorText, uncertainWrite } from '../composables/resources'
const props = defineProps<{ id: number }>()
const rows = ref<Attachment[]>([])
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const uncertain = ref(false)
const name = ref('')
const url = ref('')
const input = ref<HTMLInputElement>()
const broken = ref<number[]>([])
const controller = new AbortController()
let alive = true
async function load() {
  loading.value = true
  error.value = ''
  try {
    const list = await listAttachments(props.id, { signal: controller.signal })
    if (alive) {
      rows.value = list
      uncertain.value = false
    }
  } catch (e) {
    if (alive) error.value = errorText(e)
  } finally {
    if (alive) loading.value = false
  }
}
function failed(e: unknown) {
  error.value = errorText(e)
  if (uncertainWrite(e)) {
    uncertain.value = true
    notice.value = '写入结果不明，请先刷新并核对名称、时间与文件大小，未核对前不要重复提交。'
  }
}
async function upload(event: Event) {
  const control = event.target as HTMLInputElement
  const file = control.files?.[0]
  control.value = ''
  if (!file || busy.value || uncertain.value) return
  error.value = ''
  notice.value = ''
  if (
    file.size < 1 ||
    file.size > 10 * 1024 * 1024 ||
    !/^.+\.(png|jpe?g|gif|pdf|txt)$/i.test(file.name)
  ) {
    error.value = '请选择1字节至10MiB的PNG/JPEG/GIF/PDF/UTF8 TXT文件。'
    return
  }
  busy.value = true
  try {
    await uploadAttachment(props.id, file, { signal: controller.signal })
    notice.value = '附件已关联。'
    await load()
  } catch (e) {
    failed(e)
  } finally {
    busy.value = false
  }
}
async function addLink() {
  if (busy.value || loading.value || uncertain.value) return
  if (!name.value.trim() || !safeURL(url.value)) {
    error.value = '请填写名称与无认证信息的HTTP/HTTPS链接。'
    return
  }
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    await linkAttachment(props.id, name.value, url.value)
    name.value = ''
    url.value = ''
    notice.value = '链接已关联。'
    await load()
  } catch (e) {
    failed(e)
  } finally {
    busy.value = false
  }
}
async function remove(id: number) {
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    await deleteAttachment(id)
    notice.value = '关联已移除；文件暂时保留供一致性备份，不立即释放磁盘。'
    await load()
  } catch (e) {
    failed(e)
  } finally {
    busy.value = false
  }
}
onMounted(load)
onBeforeUnmount(() => {
  alive = false
  controller.abort()
})
</script>
<template>
  <section class="surface" aria-label="附件与链接" :aria-busy="loading || busy">
    <div class="section-heading">
      <h3>附件与链接</h3>
      <ElButton :disabled="busy || loading" @click="load">刷新附件</ElButton>
    </div>
    <p class="muted">
      支持PNG/JPEG/GIF图片、PDF和UTF8
      TXT，每个文件最多10MiB、每只最多100项；视频可关联外部链接。文件保存在本地私有目录。
    </p>
    <p v-if="error" role="alert" class="error-text">{{ error }}</p>
    <p v-if="notice" role="status">{{ notice }}</p>
    <p v-if="loading">正在读取附件…</p>
    <ElEmpty v-else-if="!rows.length && !error" description="暂无附件或链接" :image-size="60" />
    <ul class="attachment-list">
      <li v-for="a in rows" :key="a.id">
        <div>
          <strong>{{ a.name }}</strong>
          <p class="muted">
            {{ a.kind === 'link' ? '外部链接' : `${a.size_bytes} 字节` }} · 系统关联时间
            {{ new Date(a.created_at).toLocaleString('zh-CN') }}
          </p>
          <template v-if="a.kind === 'file'"
            ><img
              v-if="a.media_type.startsWith('image/') && !broken.includes(a.id)"
              :src="`/api/v1/attachments/${a.id}/content`"
              :alt="a.name"
              loading="lazy"
              @error="broken.push(a.id)"
            />
            <p v-if="broken.includes(a.id)">预览不可用，请核对附件完整性或下载原件。</p>
            <a
              :href="`/api/v1/attachments/${a.id}/content`"
              target="_blank"
              rel="noopener noreferrer"
              >查看或下载文件</a
            ></template
          ><a v-else :href="a.url!" target="_blank" rel="noopener noreferrer">打开外部链接</a>
        </div>
        <ElPopconfirm
          title="移除关联？文件将保留供备份，不立即释放磁盘。"
          confirm-button-text="确认移除"
          cancel-button-text="保留"
          @confirm="remove(a.id)"
          ><template #reference
            ><ElButton
              :aria-label="`移除附件${a.id}`"
              :disabled="busy || loading || uncertain"
              type="danger"
              plain
              >移除关联</ElButton
            ></template
          ></ElPopconfirm
        >
      </li>
    </ul>
    <div class="heading-actions">
      <ElButton
        :disabled="busy || loading || uncertain || rows.length >= 100"
        @click="input?.click()"
        >上传本地附件</ElButton
      ><input
        ref="input"
        type="file"
        aria-label="上传附件文件"
        hidden
        accept=".png,.jpg,.jpeg,.gif,.pdf,.txt"
        :disabled="busy || loading || uncertain || rows.length >= 100"
        @change="upload"
      />
    </div>
    <form class="form-grid" @submit.prevent="addLink">
      <label
        >链接名称<ElInput
          v-model="name"
          aria-label="附件链接名称"
          :maxlength="200"
          :disabled="busy || uncertain" /></label
      ><label
        >HTTP/HTTPS地址<ElInput
          v-model="url"
          aria-label="附件链接地址"
          :maxlength="2048"
          :disabled="busy || uncertain" /></label
      ><ElButton native-type="submit" :disabled="busy || loading || uncertain || rows.length >= 100"
        >关联外部链接</ElButton
      >
    </form>
    <p class="muted">
      删除仅移除关联，文件暂保留以保证并发备份的一致性；本版不自动物理清理。链接内容不会由本服务下载或备份。
    </p>
  </section>
</template>
<style scoped>
.attachment-list {
  list-style: none;
  padding: 0;
}
.attachment-list li {
  display: flex;
  align-items: start;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 0;
  border-bottom: 1px solid #edf0f4;
}
.attachment-list li > div {
  min-width: 0;
  overflow-wrap: anywhere;
}
.attachment-list img {
  display: block;
  max-width: 180px;
  max-height: 140px;
  object-fit: contain;
  margin-bottom: 8px;
}
.form-grid {
  margin-top: 20px;
}
@media (max-width: 600px) {
  .attachment-list li {
    flex-direction: column;
  }
}
</style>
