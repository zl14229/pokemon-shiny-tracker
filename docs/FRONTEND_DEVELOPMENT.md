# 前端开发（M0）

前端在 `frontend/`，使用 Vue 3、TypeScript、Vite、Element Plus；默认简体中文、本地 loopback 开发服务器。T003 为可启动/可构建的页面骨架，T004 接入 API Client；记录、收藏和数据库业务属于后续任务。

## 工具链与依赖

- Linux Node **24.22.1**、npm **11.19.1**，来自本机 nvm，`.node-version` 与 CI 共用 Node 版本，`packageManager` 记录 npm 版本。
- 经 npm registry 元数据核实并精确锁定 Vue **3.5.44**、Vite **8.3.5**、plugin-vue **6.0.10**、Element Plus **2.14.7**。当前 Node 满足 [Vue 快速开始](https://vuejs.org/guide/quick-start.html)、[Vite 要求](https://vite.dev/guide/)和测试工具引擎约束。
- TypeScript **5.9.3**、vue-tsc **3.3.13**：vue-tsc 的 peer 范围为 TypeScript >=5；选用已有稳定编译器版本，通过本工程严格类型与生产构建检查验证兼容性。
- Vitest **5.0.5**、Vue Test Utils **2.5.1**、jsdom **30.1.3** 测试真实 Vue/Element Plus 组件交互；Playwright **1.64.1** 用 Chromium 验证页面加载、交互和基本响应式布局。
- `package-lock.json` 和精确版本保证 `npm ci` 可重复执行；`.npmrc` 仅含 save-exact/engine-strict，无认证设置。升级需同步版本文件、锁文件并重新验证。

## 本地命令

从仓库根目录执行，使用 nvm 初始化的 Linux Node/npm：

```bash
cd /home/zhanglin/projects/pokemon-shiny-tracker
bash -ic 'command -v node; node --version; command -v npm; npm --version'
bash -ic 'cd frontend && npm ci'
bash -ic 'cd frontend && npm run dev'
```

浏览器打开 `http://127.0.0.1:5173`。端口被占用时直接失败，不静默换端口；开发服务器和 preview 均固定 loopback，不增加局域网访问权限。Ctrl+C 停止进程。

```bash
bash -ic 'cd frontend && npm run typecheck && npm test && npm run build && npm run smoke'
bash -ic 'cd frontend && npm run preview'
```

`build` 先执行 vue-tsc，再输出 `frontend/dist/`；preview 默认 `127.0.0.1:4173`，用于验证构建结果，不是生产部署方案。开发工具、node_modules、dist、测试报告和浏览器缓存不提交。

`npm run smoke` 在临时 loopback 端口启动真实 Vite 开发服务器，检查 HTML 与 Vue 模块编译，再启动构建预览并读取构建后的 JavaScript；结束时关闭两个服务器。

## 浏览器验收

```bash
bash -ic 'cd frontend && npx playwright install chromium'
bash -ic 'cd frontend && npm run test:e2e'
```

测试自行启动 loopback Vite 服务器并清理进程，不复用未知的既有服务。覆盖桌面 1280px 和窄屏 390px、页面 JS 错误、Element Plus 按钮展开/收起，以及横向溢出。测试服务端口 45173 必须空闲。

Chromium 还需要系统共享库；Debian WSL 若缺少浏览器依赖，不影响类型检查、组件测试、构建和 HTTP 启动验证。CI 在临时 Ubuntu runner 中运行 `npx playwright install --with-deps chromium` 后执行真实浏览器检查，本机缺少依赖时应如实记录，不假称浏览器测试通过。

## CI 与边界

`frontend-checks` 使用固定 SHA 的 setup-node、`.node-version`、`npm ci`，执行严格类型、组件测试、生产构建和 Chromium 测试。dev Ruleset 已实际增加该必需检查并回读，原三项检查、strict=true 和无 bypass 保留；prod 未改动，见 `docs/evidence/github-rulesets-t003.json`。

本阶段不引入业务导航、收藏表单、假统计数据或网络资料采集。前端包及所有 `VITE_*` 配置都对浏览器公开，禁止放置密钥；当前没有任何认证凭据或个人数据。
