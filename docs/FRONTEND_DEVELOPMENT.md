# 前端开发（M0）

前端在 `frontend/`，使用 Vue 3、TypeScript、Vite、Element Plus；默认简体中文、本地 loopback 开发服务器。T003 为页面骨架，T004 接入 API Client 和本地服务状态；记录、收藏和数据库业务属于后续任务。

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

## T004 API Client 与前后端联调

启动后端后运行前端，页面的“本地服务”卡片会自动检查健康，提供加载、成功、失败与重试；组件移除时取消未完成请求。客户端固定使用 `/api/v1`，处理 HTTP 状态、已知服务端错误码、非 JSON/异常 DTO、网络失败、5 秒超时和调用方取消。错误提示不直接显示未知服务端文本或 HTML。

[Fetch](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch) 的 HTTP 失败需显式检查状态；[AbortController](https://developer.mozilla.org/en-US/docs/Web/API/AbortController) 用于取消 fetch 和响应体读取。对应行为均有测试，超时覆盖响应体读取过程。当前只支持健康查询及通用 JSON GET 基础，不实现业务接口。

Vite 开发与构建预览均按[官方 proxy 配置](https://vite.dev/config/server-options#server-proxy)代理 `/api/v1` 到 `127.0.0.1:8080`，不增加 CORS 或外网监听。后端换端口时分别给两个进程设置环境变量：

```bash
bash -ic 'SHINY_HTTP_PORT=8081 make run'
bash -ic 'cd frontend && SHINY_API_PORT=8081 npm run dev'
```

`SHINY_API_PORT` 只从前端启动进程环境读取，须为 1–65535 十进制端口；模板见根 `.env.example`，不自动加载该模板。没有可配置的远程服务地址或密钥。Vite proxy 属于本地开发/预览能力，后续正式部署的同源路由另行验收。

```bash
bash -ic 'make build && cd frontend && npm run test:integration'
bash -ic 'make build && cd frontend && npm run test:e2e'
```

- 客户端/组件测试覆盖正常响应、JSON 404/405/500、非 JSON 错误、错误 DTO、网络、超时（含响应体阶段）、取消、加载/重试/组件清理。
- `test:integration` 使用临时端口启动真实 Go 程序和 Vite，验证 API Client 健康、404、405、构建预览代理、实际后端停机产生 502，以及重启后成功；不访问数据库。
- Chromium 测试同时启动真实 Go 和 Vite，先验证无 mock 的成功调用，再分别注入 JSON/非 JSON 错误、错误成功体、网络失败和超时，确认用户提示与重试恢复。故障仅在测试拦截中注入，不向实际后端增加故障端点。测试使用端口 45080/45173，拒绝复用未知服务。

CI 的 frontend-checks 构建后端并运行所有联调/浏览器测试。本机缺少 Chromium 共享库的限制仍保留，浏览器验收以安装完整依赖后的 CI 实际结果为证据。
