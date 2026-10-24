# 开发日志

日期采用 Asia/Shanghai。历史任务的详细执行与证据以 `docs/tasks/Txxx.md` 为准。

## 2026-10-24 — T002

- 从最新 dev `af58aebc57e81e2561ba444d383f5c760f19593a` 创建 `feature/T002-router-health`，确认 T001 verified，工作区干净。
- 增加 `/api/v1/health`、统一 JSON 404/405/500、内部错误和 panic 恢复；不连接数据库，不引入业务端点。
- 更新原有真实 HTTP 生命周期测试与进程验证脚本。首次 smoke 检查遇到端口重绑的 `Address already in use`；分析为 TCP TIME_WAIT 对探测 socket 的影响，将探测按 Go listener 的行为启用 SO_REUSEADDR，仍会拒绝真实占用中的监听端口。
- 测试、PR、合并和 CI 的实际最终结果在任务记录中归档。

- T002 实现 PR https://github.com/zl14229/pokemon-shiny-tracker/pull/5 已合并（51ba5d81dcf719bb09abc743b3d349b5646889e6），PR CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40055913517 与 dev CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40056003815 成功；本任务标记 verified 并走验收收尾 PR。

## 2026-10-24 — T003

- T002 验收收尾 PR #6 和 dev CI 成功后，从最新 dev 1c3e472 创建独立 T003 分支。
- 精确锁定 Vue/Vite/Element Plus 与 TS/测试工具；验证 nvm Linux Node/npm。npm install/ci、严格类型、组件交互、生产构建和真实 Vite/预览 HTTP smoke 成功，npm 审计 0 已知漏洞。
- 本机 Chromium 下载后实际启动报缺少 libglib-2.0.so.0；不假称通过。CI 安装完整依赖后执行同一 Chromium 测试，作为必需 frontend-checks 的一部分。
- dev Ruleset 追加 frontend-checks 并回读，保留原检查、严格模式与无 bypass，不改 prod；不提前实现 T004。

- T003 实现 PR https://github.com/zl14229/pokemon-shiny-tracker/pull/7 已合并（ea46d294a60b02badb48cd916cc7d1075fdddc46），PR CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40056989398 与 dev CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40057135058 成功；本任务标记 verified 并走验收收尾 PR。

## 2026-10-24 — T004

- T003 收尾 PR #8 与最终 dev CI 四项成功后，从最新 dev d7eab809 创建独立 T004 分支，确认 T002/T003 verified。
- 实现同源 API Client、健康 DTO 校验、安全错误提示、超时/取消、状态卡片与重试、loopback 开发/构建预览代理；无业务或数据库。
- 修正 Vitest 串行 API 兼容性和 Vite 实际 502 网关语义断言；未跳过失败检查。复验类型、21 项客户端/组件测试、构建/HTTP smoke 和 5 项真实 Go/Vite 联调全部成功。
- Chromium 验证真实成功链路与五类故障提示/重试；故障仅测试拦截，CI 为必需检查。最终 M0 验收在实现合并与 dev CI 成功后执行并归档。

- T004 实现 PR https://github.com/zl14229/pokemon-shiny-tracker/pull/9 已合并（70347166e587abbdb4a0dab2c750b1a6be9c55cf），PR CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40057984013 与 dev CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40058212554 成功；本任务标记 verified 并走验收收尾 PR。

## 2026-10-24 — M0 最终集成验收

- T002 → T003 → T004 依次独立从最新远程 dev 创建分支；实现 PR #5/#7/#9 各经必需 CI 与自审后 Merge Commit，源分支保留；T002/T003 通过收尾 PR #6/#8 归档，T004 收尾同时归档本报告。
- 在 T004 实际合并 dev 7034716 上再次执行本地 M0 全套 Go 与前端/真实集成检查，全部成功；最终实现 PR 的 21 项组件/客户端、5 项真实集成及 7 项 Chromium 验收成功，合并后 dev CI 四项 success。
- 四项 GitHub 必需检查、strict=true、无 bypass、prod 不变均回读确认；完整证据见 M0_ACCEPTANCE.md 和各 Task 记录。
- 本轮 T002–T004 全部 verified；不进入 T005，不发布 prod，无未解决的 M0 功能阻塞。WSL 缺浏览器共享库、正式部署路由与秘密扫描边界仍如实记录。

## 2026-10-24 — T005

- 在最新远程 dev b885394 上确认 M0 verified，工作区干净，独立建立 feature/T005-sqlite-models；编码前记录实体、迁移、数据安全和测试方案。
- 引入 GORM/SQLite 与精确驱动版本，基础六实体、复合形态外键、可空未知、状态/数值约束和版本化事务迁移。默认库在用户目录，拒绝仓库内/未知/未来数据库，不对已有数据反复 AutoMigrate。
- 现有 smoke、真实前端集成和浏览器启动器全部改用各自创建的临时库，避免触碰个人默认库。测试与合并实际结果在 T005 记录归档。

- T005 实现 PR https://github.com/zl14229/pokemon-shiny-tracker/pull/11 已合并（d5d42679079ac44af4f7ffa8e3cdf2af3dd0821d），PR CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40062408488 与 dev CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40062568518 成功；本任务标记 verified 并走验收收尾 PR。

## 2026-10-24 — T006

- T005 收尾 #12/dev CI 成功，从最新 dev 465eaf69 创建 feature/T006-reference-data，编码前记录源合规、缓存/暂存/应用、映射、未知/冲突和测试方案。
- 使用带明确 UA、顺序至少 1 秒间隔和缓存的客户端真实读取 PokéAPI，保留最少结构化事实与源 URL/哈希/时间，得到 3 物种/3 基础形态/4 游戏。实际源语言标签为小写，采用兼容大小写匹配；版本 ID 和名称以实际响应为准，不按旧编号假设。
- 原始 JSON 字节以缓存封装保留，离线重建规范化结果逐字节一致；不提交缓存原文/图片/正文。中文只标 source_provided，形态缺失名称 unknown，源内冲突明确 conflict。
- SQL v2 仅新增来源批次和映射；ID/父形态校验、预览回滚、重复导入、证据元数据刷新、冲突差异/整批拒绝、源失败保留个人记录及 v1 升级通过真实 SQLite 测试。CLI 临时库 preview/apply/reapply 结果为 10 计划未提交、10 新增、0 新增/10 不变。
- 启动仅在空参考表离线导入种子，现有表不覆盖；查询本地 SQLite，明确 partial 和未验证可刷性。

- T006 实现 PR https://github.com/zl14229/pokemon-shiny-tracker/pull/13 已合并（f51dea775f5b8bd693324e23acc955c869bbda09），PR CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40065170340 与 dev CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40065402167 成功；本任务标记 verified 并走验收收尾 PR。

- T007 开始：最新 dev dc36c0e530174a9fabb776273970e66eeffe3b2f，依赖 verified、工作区干净；方案：新增 service 层，在 SQLite 事务中验证物种/形态/游戏关系与三状态；完成须有有效成果，放弃或重新打开不删除成果，有历史关联则拒绝删除 Hunt。严格 JSON DTO、完整 PUT、未知值保留 null；计数/单位/耗时和部分精度日期校验，API 统一 400/404/409/415/500。真实临时数据库测试覆盖状态、0..N/意外成果、错误参数、事务回滚；不实现计时会话、查询筛选或业务 UI。

- T007 实现及本地验收完成：Hunt CRUD/状态/累计值、严格 JSON、真实 SQLite 与 HTTP 边界测试；Go 普通/race/vet/build/启动全部成功。逐文件自审无待修复项；高级列表查询与个体 API 分别留待 T009/T008。

- T007 实现 PR https://github.com/zl14229/pokemon-shiny-tracker/pull/15 已合并（7555c331277de45742970b41c9b2e53594bec50c），PR CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40082676847 与 dev CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40082901088 成功；本任务标记 verified 并走验收收尾 PR。

- T008 开始：最新 dev 939406b5b625ae938fef26e5a7cf512f26231857，依赖 verified、工作区干净；方案：严格 DTO 与 service 事务登记/更新/软删除个体，原始获得与当前物种/形态/游戏分离且不推断未知。进化/形态/位置/持有变化使用原 ID 并存不可变前后快照；物种变化由调用者声明进化或纠错，不假称验证进化合法性。历史日期可空/部分精度，删除保留审计，保护已完成 Hunt 的最后有效成果。真实临时库测试覆盖独立/多成果/意外闪光、迁移/进化不重复、关联与数据校验、历史事务失败回滚及 HTTP 错误；不做 M2 界面或 T023 完整历史系统。

- T008 实现/本地验收：个体 CRUD、独立/关联登记、原始/当前分离、原 ID 进化/位置/持有历史、完成 Hunt 最后成果保护；Go 普通/race/vet/build/启动均成功。历史失败注入验证三类写操作原子回滚，自审无待修复项。

- T008 实现 PR https://github.com/zl14229/pokemon-shiny-tracker/pull/17 已合并（c9ae1ced0614aa9b256399016c0952eb1144d7ad），PR CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40083753536 与 dev CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40083990124 成功；本任务标记 verified 并走验收收尾 PR。

- T009 开始：最新 dev 7aa5c7f4fef483d1dd598b8fddd6a6343d30dc8f，依赖 verified、工作区干净；方案：参数白名单、绑定值与字面搜索，分别筛选目标/原始/当前引用、状态、关联、单位及部分精度日期；AND 组合、明确空值、ID 次级稳定排序、同快照 total/分页，计数比较要求同单位。测试 SQL 注入/通配符/同值排序/跨页/错误边界；增添仅临时库的真实后端进程 M1 CRUD/离线/重启验收并接入既有 CI。完成 Go/前端回归及自审后 PR/合并/dev CI，收尾归档 M1 报告，T009 后停止，不实现 M2。

- T009 实现/本地验收：组合筛选、原始/当前角色、字面搜索、同单位累计过滤、稳定排序与事务分页通过。新增 M1 真实进程验收接入 make smoke/既有 go-checks；Go 普通/race/vet/build/两层启动验收、npm ci/类型/21测试/build/HTTP/5集成成功，自审无待修复项。

- T009 实现 PR https://github.com/zl14229/pokemon-shiny-tracker/pull/19 已合并（e36373d83c7d5c6b66f30e80115c06462025b122），PR CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40085147745 与 dev CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40085461939 成功；本任务标记 verified 并走验收收尾 PR。

- M1 集成验收：在 T009 实现合并 e36373d83c7d5c6b66f30e80115c06462025b122 上本地 Go/构建/race/vet/空库与真实进程 CRUD/查询/历史/重启、前端 npm ci/类型/21测试/构建/HTTP/5集成全部成功。dev CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40085461939 四项成功；五个源分支、prod原SHA及strict/no-bypass规则实际回读，M1报告及规则证据通过T009收尾PR归档。仅T005–T009，T010+保持planned，完成后停止。

- T010 开始：最新 dev 7e2f3ebc9f72437e4624b58b1ea423599f6815c8，依赖 verified、工作区干净；方案：复用 Vue3/TS/Element Plus、既有同源 Client，扩展严格 DTO/写请求与错误分类；简单 hash 页面导航、中文响应式列表/筛选/分页、创建/完整编辑、详情/关联成果、三态操作。空字段保留 null、部分日期/安全整数验证、保留完整 PUT 字段，写入结果不明不自动重试。复用字段/反馈/资料选择器，已知资料部分覆盖不推断可刷性。组件/Client单测、真实Go/Vite联调及Playwright实际流程（独立临时库）验证，类型/构建/CI、自审后 PR/merge/dev CI/收尾；不开发 T011 写入界面或 M3+。

- T010 实现及本地验收：中文Hunt列表/表单/详情/关联成果/三态、同源DTO/写请求、安全整数/空值/部分日期/完整PUT与错误反馈完成；38项组件/Client、7项真实API、12项本机Chromium（桌面/390px）和类型/构建/HTTP/Go回归通过。浏览器依赖仅临时解包，未改系统；截图和代码自审无未解决项，CI/合并待核实。

- T010 实现 PR https://github.com/zl14229/pokemon-shiny-tracker/pull/21 已合并（1d74d00e91ef48b614aca9213928fbd5ced97e3e），PR CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40107701589 与 dev CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40107960361 成功；本任务标记 verified 并走验收收尾 PR。

- T011 开始：最新 dev ec6071dbb861150e428223845049cff9116d3f8b，依赖 verified、工作区干净；方案：复用T010中文导航/表单/反馈与Client：收藏列表以来源/持有/原始与当前游戏等筛选分页；个体可不关联Hunt，关联选择器分页搜索现有任务。完整DTO保持原始/当前物种、形态、游戏及球种/等级/备注等全部字段，未知不补造；进化/纠错由用户明确声明，同ID更新、轻量历史只读。关联成果与Hunt页面互通，最后成果冲突按后端反馈；版本前置比对与写入不明防重复沿用。真实临时库组件/API/完整M2浏览器流程、响应式截图、类型/构建/CI与自审；PR/merge/dev CI后收尾报告，M2后停止，不做M3。

- T011 实现/本地验收：收藏独立/关联登记、原始/当前分离、球种等完整编辑、同ID进化/转移/持有与轻量历史、组合筛选分页及Hunt双向联动完成。51单元/组件、8真实API、19本机Chromium完整M2流程、类型/构建/HTTP/Go/治理均通过，修复自审与测试发现问题并检查桌面/390px截图；PR/合并验收待核实。
