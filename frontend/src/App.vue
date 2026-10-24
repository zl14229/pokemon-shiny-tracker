<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElButton, ElCard, ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import ServiceStatus from './components/ServiceStatus.vue'
import HuntList from './pages/HuntList.vue'
import HuntForm from './pages/HuntForm.vue'
import HuntDetail from './pages/HuntDetail.vue'
import ShinyList from './pages/ShinyList.vue'
import ShinyForm from './pages/ShinyForm.vue'
import ShinyDetail from './pages/ShinyDetail.vue'
import { useRoute } from './domain/navigation'
const route = useRoute(),
  showGuide = ref(false)
const path = computed(() => route.value.split('?')[0] ?? '/home')
const query = computed(() => route.value.split('?').slice(1).join('?'))
const detail = computed(() => /^\/hunts\/([1-9]\d*)(\/edit)?$/.exec(path.value))
const id = computed(() =>
  detail.value && Number.isSafeInteger(Number(detail.value[1])) ? Number(detail.value[1]) : null,
)
const collectionDetail = computed(() => /^\/collection\/([1-9]\d*)(\/edit)?$/.exec(path.value))
const collectionId = computed(() =>
  collectionDetail.value && Number.isSafeInteger(Number(collectionDetail.value[1]))
    ? Number(collectionDetail.value[1])
    : null,
)
</script>
<template>
  <ElConfigProvider :locale="zhCn">
    <div class="app-shell">
      <aside class="sidebar">
        <a class="brand" href="#/home"
          ><span class="brand-mark" aria-hidden="true">✦</span>
          <h1>Pokémon Shiny Tracker</h1></a
        >
        <p class="sidebar-caption">你的闪光收藏手帐</p>
        <nav aria-label="主导航">
          <a href="#/home" :aria-current="path === '/home' ? 'page' : undefined"
            ><span aria-hidden="true">⌂</span>首页</a
          ><a href="#/hunts" :aria-current="path.startsWith('/hunts') ? 'page' : undefined"
            ><span aria-hidden="true">◎</span>刷闪任务</a
          >
          <a href="#/collection" :aria-current="path.startsWith('/collection') ? 'page' : undefined"
            ><span aria-hidden="true">✧</span>闪光收藏</a
          >
        </nav>
        <div class="sidebar-foot"><span class="local-dot" />本地运行 · 个人收藏</div>
      </aside>
      <main id="main" class="main-content">
        <template v-if="path === '/home'">
          <header class="page-heading">
            <div>
              <p class="eyebrow">属于你的闪光旅程</p>
              <h2>每一次相遇，都值得记录</h2>
              <p class="muted">从一个目标开始，把尝试和回忆留在这里。</p>
            </div>
          </header>
          <ElCard class="welcome-card" shadow="never"
            ><span class="section-kicker">HUNT JOURNAL</span>
            <h3>下一次闪光，正在路上</h3>
            <p>创建刷闪任务，记录次数与耗时，整理每段旅程的进度与成果。</p>
            <ElButton tag="a" href="#/hunts" type="primary">查看刷闪任务 →</ElButton
            ><ElButton
              :aria-expanded="showGuide"
              aria-controls="usage-guide"
              @click="showGuide = !showGuide"
              >{{ showGuide ? '收起说明' : '查看说明' }}</ElButton
            >
            <p v-if="showGuide" id="usage-guide" class="usage-guide">
              新建任务时只填写已知信息；未知日期、次数与游戏可留空。完成任务需要至少一个关联成果，放弃会保留已有成果。
            </p></ElCard
          >
          <ElCard class="collection-welcome" shadow="never"
            ><span class="section-kicker">SHINY COLLECTION</span>
            <h3>留住值得珍藏的相遇</h3>
            <p class="muted">登记已有闪光，无需重建刷闪过程；原始获得与当前保存位置分开记录。</p>
            <ElButton tag="a" href="#/collection">查看闪光收藏 →</ElButton></ElCard
          >
          <ServiceStatus />
        </template>
        <HuntList v-else-if="path === '/hunts'" :query="query" />
        <HuntForm v-else-if="path === '/hunts/new'" key="new-hunt" />
        <HuntForm v-else-if="id && detail?.[2]" :key="`${id}-edit`" :id="id" />
        <HuntDetail v-else-if="id" :key="id" :id="id" />
        <ShinyList v-else-if="path === '/collection'" :query="query" />
        <ShinyForm v-else-if="path === '/collection/new'" key="new-shiny" :query="query" />
        <ShinyForm
          v-else-if="collectionId && collectionDetail?.[2]"
          :key="`${collectionId}-edit`"
          :id="collectionId"
        />
        <ShinyDetail v-else-if="collectionId" :key="`shiny-${collectionId}`" :id="collectionId" />
        <section v-else class="surface">
          <h2>页面不存在</h2>
          <p>请通过导航返回任务列表。</p>
          <a href="#/hunts">返回刷闪任务</a>
        </section>
        <footer class="app-footer">每一份记录，都属于你。 · 本地数据请自行妥善保管</footer>
      </main>
    </div>
  </ElConfigProvider>
</template>
