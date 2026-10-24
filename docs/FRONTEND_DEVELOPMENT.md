# 前端开发

前端在 `frontend/`，使用 Vue 3、TypeScript、Vite、Element Plus；默认简体中文、本地 loopback 开发服务器。T003 为页面骨架，T004 接入 API Client 和本地服务状态；T010 增加 Hunt 管理界面，T011 增加闪光收藏界面。数据库与业务 API 已在 M1 实现。

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

T010 提供首页与 Hunt 导航，不引入图鉴、Planner、统计或网络资料采集；T011 已接入收藏写入，后续里程碑按单独授权执行。前端包及所有 `VITE_*` 配置都对浏览器公开，禁止放置密钥；当前没有任何认证凭据或个人数据。

## T004 API Client 与前后端联调

启动后端后运行前端，页面的“本地服务”卡片会自动检查健康，提供加载、成功、失败与重试；组件移除时取消未完成请求。客户端固定使用 `/api/v1`，处理 HTTP 状态、已知服务端错误码、非 JSON/异常 DTO、网络失败、5 秒超时和调用方取消。错误提示不直接显示未知服务端文本或 HTML。

[Fetch](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch) 的 HTTP 失败需显式检查状态；[AbortController](https://developer.mozilla.org/en-US/docs/Web/API/AbortController) 用于取消 fetch 和响应体读取。对应行为均有测试，超时覆盖响应体读取过程。T004 交付时仅健康与 JSON GET；T010 的类型化业务查询和写入扩展见下方。

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
- `test:integration` 使用临时端口启动真实 Go 程序和 Vite，验证 API Client 健康、404、405、构建预览代理、实际后端停机产生 502，以及重启后成功；所有测试仅使用自建临时 SQLite，不访问默认个人库。
- Chromium 测试同时启动真实 Go 和 Vite，先验证无 mock 的成功调用，再分别注入 JSON/非 JSON 错误、错误成功体、网络失败和超时，确认用户提示与重试恢复。故障仅在测试拦截中注入，不向实际后端增加故障端点。测试使用端口 45080/45173，拒绝复用未知服务。

CI 的 frontend-checks 构建后端并运行所有联调/浏览器测试。本机缺依赖时不可虚报浏览器成功；T010 本轮用仓库外临时解包的共享库与字体环境执行本地 Chromium，未安装系统包、未修改工具链或固化临时路径。CI 仍在标准 runner 安装依赖后执行相同场景。

## T010 Hunt 管理界面

hash 路由不增加部署重写要求：`#/home`、`#/hunts`、`#/hunts/new`、`#/hunts/:id`、`#/hunts/:id/edit`。使用现有 Vue/TypeScript/Element Plus，无额外运行时依赖或状态框架。列表以 q/status 组合筛选、20条分页，保留 URL 查询参数；详情呈现游戏/目标/累计/日期、三态操作及分页关联成果。允许意外闪光，完成规则由后端权威校验；放弃保留成果。不存在/失败/加载/空页均有反馈与可用的返回路径。

`api/client.ts` 负责同源 GET/POST/PUT、超时/取消/安全错误；`api/models.ts` 与 `tracker.ts` 做运行时 DTO 校验和类型化调用；domain 层纯函数管理表单/日期/格式；共享反馈、资料选择和状态组件复用。只从本地参考接口分批读取，超过单页500条会继续加载，不静默截断；源中文待核验/覆盖有限的提示保留，不推断可刷性。

表单不默认填日期/游戏/次数/耗时；空值保存 null，0保留为已知零。次数须提供单位，耗时为非负秒。支持原有年/月/日精度与真实日历/范围验证。浏览器仅编辑安全整数（最大9007199254740991）；超出范围的响应明确报错，不能悄悄舍入后写回。编辑保留全部既有可编辑字段，状态操作也构造完整 PUT，不以零值/空对象覆盖未展示字段。

编辑前读取并比对 updated_at，发现已提交的新版本会拒绝本次写入并保留草稿；这是客户端前置检查，不能替代服务端原子并发控制，仍应避免同时在多个窗口编辑同一条记录。未发出 PUT 前的读取失败可安全重试；POST/PUT 发出后遇网络/超时/异常成功体/5xx，结果不明时先检查列表，不自动重试创建。所有服务器原始错误文字/HTML保持脱敏。

测试包含纯字段/DTO/请求测试、真实 Element Plus 表单与错误/草稿保持、真实 Go/Vite 写入/完整更新/关联成果/409、以及 Chromium 创建→编辑→放弃→重开→完成、桌面/390px、验证阻止写入、失败重试、已实际提交但代理报错时不重复创建、旧版本编辑保护。浏览器所需两个服务端口必须空闲，不复用外部服务；数据库及测试注入只在自建临时目录。

## T011 闪光收藏与 M2 联动

路由 `#/collection`、`#/collection/new`、`#/collection/:id`、`#/collection/:id/edit`，沿用全局简体中文、桌面/窄屏布局、资料/反馈/安全写入组件。列表区分原始获得游戏与当前保存游戏，提供搜索、当前物种、来源、持有、原始/当前游戏、Hunt关联筛选与分页；分页保留既有URL条件及排序，重置清除所有条件。软删除记录不在正常列表，已交换/放生仍可通过全部状态查看，不伪造持有统计或图鉴完成率。

登记仅要求当前物种；来源/持有默认未记录，其余未知字段为空，不从当前物种、关联Hunt或系统日期补原始资料。可直接登记已有个体而不建Hunt；现有Hunt选择器支持搜索与分页、明确清空关联。Hunt详情的“登记关联成果”只预选用户明确选择的Hunt，不预填物种/来源/日期/游戏，以支持意外闪光。详情的成果与个体关联任务互相链接。

完整编辑保留原始/当前物种、形态、游戏、位置、日期、昵称、来源、持有、性别、性格、特性、球种、等级、备注及关联。等级1–100，日期保留年/月/日精度，字符长度与后端一致；未填写的原始信息保持未知。当前物种变化由用户选择“进化（用户记录）”或“资料纠错”，不假称已核验进化条件；进化/保存位置/持有变化使用同一ID，详情只读展示现有轻量历史，事件日期未知与系统记录时间分开。完整历史管理留T023，不增加新的历史写入端点。

客户端保留原有版本前置读取与写入结果不明防重复规则。解除/移走完成Hunt最后有效成果时显示明确冲突提示，原关联和草稿保留，需先调整任务状态；不绕过后端约束。源不可用不触发个人记录同步，所有页面查询仅本地API。M2不新增运行时依赖、后端模型/迁移、数据源采集、导入/附件/备份或M4–M6模块。

新增自动验收覆盖：独立/关联登记、原始/当前字段与完整编辑、同ID进化/转移/持有与历史、真实组合过滤/21条跨页无重复、错误输入阻止写入、关联冲突、接口失败恢复、实际提交但代理报错时不重复登记；完整浏览器流程由创建Hunt走到收藏编辑。所有实例继续使用自建临时库，拒绝复用未知服务；字体/系统库仅用于本机临时测试环境，CI独立安装标准依赖。
