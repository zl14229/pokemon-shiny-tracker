# M2 验收报告

- 验收日期：2026-10-25，Asia/Shanghai；执行平台 Debian WSL / Linux amd64。
- 授权及完成范围：仅 T010、T011；T004/T007/T008/T009 等前置任务已 verified，按 T010 实现/CI/合并/dev CI/收尾闭环后，从新 dev 开始 T011。T000–T011 均 verified，T012–T028 保持 planned，未开始 M3，未发布 prod。
- M2 合并后集成复验基准：T011 实现实际 dev merge `910bed7c670fbfd86444707a509732b60b11f0c3`。报告在该实现 PR 与 dev CI 核实、本地合并后复验通过之后归档。
- 沿用 Vue 3 + TypeScript + Vite + Element Plus、现有同源 API Client 与 Go/Gin/GORM/SQLite API，无新增运行时依赖、后端/模型/迁移或冻结技术栈/产品规则变更。

## Task、PR、提交与 CI

### T010 — Hunt 前端列表/表单/详情

- 状态：verified；独立分支 `feature/T010-hunt-ui`，从当时最新远程 dev 创建，源分支保留。
- 实现：中文响应式 Hunt 列表/筛选/分页、创建/完整编辑、详情/关联成果/三态操作；扩展同源Client/严格DTO与安全整数/部分日期/空值验证；读取取消与写入不明防重复、旧版本前置比对；共享资料/反馈/状态组件。
- [实现 PR](https://github.com/zl14229/pokemon-shiny-tracker/pull/21)；初次实现 commit `b50d92e602206e63a978bfe22f1de09b49785b2b`；最终 PR HEAD `7824df97d4dc6c6b1a8e7dbcca61c1861552247d`。
- 实际 merge SHA：`1d74d00e91ef48b614aca9213928fbd5ced97e3e`。
- [最终 PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40107701589)、[实现合并后 dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40107960361)：四项检查全部成功。
- [验收收尾 PR #22](https://github.com/zl14229/pokemon-shiny-tracker/pull/22) 已合并，merge `ec6071dbb861150e428223845049cff9116d3f8b`；[收尾 PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40108268703)、[收尾 dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40108509235) 均成功。
- 详细记录：[任务证据](tasks/T010.md)。

### T011 — ShinyRecord 前端列表/详情/编辑

- 状态：verified；独立分支 `feature/T011-shiny-collection-ui`，从当时最新远程 dev 创建，源分支保留。
- 实现：闪光收藏列表/详情/独立登记/完整编辑，原始与当前信息、球种/个体字段和持有/来源明确；可选Hunt分页搜索/清空与双向联动，用户声明进化/纠错同ID更新和只读轻量历史；组合筛选/保留URL排序分页、加载/错误/空态和安全写入。
- [实现 PR](https://github.com/zl14229/pokemon-shiny-tracker/pull/23)；初次实现 commit `5f7934464f51710c4d47ed358e5c9b944f928533`；最终 PR HEAD `16503c9b2880b38106c6627030290af6fb4e5bc8`。
- 实际 merge SHA：`910bed7c670fbfd86444707a509732b60b11f0c3`。
- [最终 PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40111613096)、[实现合并后 dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40112146952)：四项检查全部成功。
- 本报告由同一源分支的 T011 验收收尾 PR 归档；收尾自身 merge/dev CI 由对应 GitHub PR/Actions 提供，避免提交自引用，最终执行回复给出实际结果。
- 详细记录：[任务证据](tasks/T011.md)。

## 用户要求的端到端验收

| 步骤 | 真实执行与结果 |
|---|---|
| 1. 创建 Hunt | 浏览器使用真实表单选择物种并保存；字段空值/0/单位/日期精度均按真实 API 保存 |
| 2. 修改状态和字段 | 浏览器完整编辑次数/耗时/日期/备注，进行中→放弃→重开，登记有效成果后完成；已有成果保留 |
| 3. 登记闪光个体 | 浏览器填写个体信息并真实保存；来源/持有/未知资料不默认补造 |
| 4. 关联或不关联 Hunt | 从已有 Hunt 明确登记关联成果（只预选关联ID，不推断其他事实）；独立历史个体 HuntID 为 null，不创建伪造任务 |
| 5. 查看及修改收藏 | 原始/当前物种与游戏、位置、球种、等级、来源、持有和备注正确；同ID进化/位置/持有变化不重复建档，轻量历史可读，原始未知仍为未知 |
| 6. 异常输入与错误 | 客户端校验阻止无效写入；HTTP/网络/超时/异常DTO/源数据缺失反馈，服务器原始消息/HTML不渲染；草稿保留、旧版本前置拒绝；最后成果解除关联冲突不破坏原关系；实际已提交而代理报错后不重复创建 |
| 7. 构建及 CI | 类型、51项单元/组件、生产构建/HTTP smoke、8项真实Go/Vite/API Client联调、19项Chromium全部通过，最终实现PR和合并后dev四项CI成功 |

额外验证：收藏21条真实临时记录，组合筛选/原始与当前角色/排序/总数/上一页与下一页无重复；加载/空状态/不存在/失败与重试；桌面1280px和窄屏390px无页面横向溢出，表格在内容区滚动。截图视觉自审与页面异常检查完成。修复测试发现的下拉定位、分页参数保留及表格布局观察通知，最终完整复验通过。

对应场景为 [Hunt 浏览器验收](../frontend/tests/e2e/hunts.spec.ts)、[收藏与完整M2验收](../frontend/tests/e2e/collection.spec.ts)、[真实API联调](../frontend/tests/integration/api.test.ts)。测试使用真实后端、Vite同源代理、Element Plus控件与自建临时SQLite，没有访问默认个人库、真实记录或附件。

## 合并后的检查与环境

从项目仓库使用交互 Bash 加载 Linux Go PATH/nvm；Go 1.27.3、Node 24.22.1、npm 11.19.1。

- `go test ./...`、`go build ./...`、`make check`（格式/module/tidy/verify/vet/race/build）与 `make smoke`（含M1真实进程持久化回归）：成功，缓存命中沿用有效测试结果，进程 smoke 实际重新执行。
- `npm ci`、`npm run typecheck`、`npm test`（51项）、`npm run build`、`npm run smoke`、`npm run test:integration`（8项）：成功。
- `npm run test:e2e`：19项真实 Chromium 成功，包括两项完整跨模块/独立记录流程；CI使用相同测试，安装标准系统依赖后执行。
- 本机原先缺 Chromium 共享库/字体；本轮从 Debian 公共包只解包至仓库外临时目录，以命令限定的 `LD_LIBRARY_PATH` / `FONTCONFIG_FILE` 运行，未安装系统软件或重新安装工具链，也未把机器临时路径固化到项目配置。截图、字体、浏览器/依赖缓存及数据库不提交。
- 冻结哈希、29 Task/依赖、16 ADR、7项治理回归、暂存/合并差异、忽略规则与基础凭证模式检查通过；无冻结原件漂移。

## 自审、治理与文档

逐任务在提交和合并前检查实际差异、类型/DTO/完整PUT、空值与精度、原始/当前角色、身份/历史/关联、异步/错误/写入防重、筛选/分页和响应式。没有已知未解决的范围内缺陷，PR 均标记 code-reviewed；未伪造第二账号批准，未留下未授权GitHub评论。code-review 无配套 code-review-* 子技能，未启动子代理。

实际回读 [M2 Ruleset 证据](evidence/github-rulesets-m2.json)：dev active、四项必需检查、strict=true、讨论须解决、禁止强推/删除、bypass为空、Merge Commit；所有合并等必需检查成功才执行，没有直接提交dev或绕过规则。两个任务源分支保留；prod实际读取仍为 `dc68988b7c76bf23ae3b68b588d15858f50b742e`，本轮未修改prod。

已更新 README、FRONTEND_DEVELOPMENT.md、GITHUB_WORKFLOW.md、两份任务记录、status.json 与 DEVELOPMENT_LOG；没有修改冻结产品规格/Task定义/ADR。用户正常使用入口和命令见 [前端开发文档](FRONTEND_DEVELOPMENT.md)，可从首页进入刷闪任务或闪光收藏。

## 保留限制与未验证事项

- **并发**：updated_at客户端前置比对可发现已提交的新版本，但不是服务端原子锁；比对与PUT之间仍有竞态，应避免多窗口同时编辑同一记录。分页跨请求遇新增/编辑/删除可移动边界，未实现跨请求快照/游标。
- **资料**：基础资料仅部分覆盖，源中文未做52Poké交叉核验，未知不补造；物种/游戏存在不证明合法可刷。进化是用户声明，不代表已核验完整进化图或游戏条件。
- **数值**：浏览器只编辑安全整数，超出安全范围的响应明确拒绝写回，不静默舍入。事件日期未知与系统审计时间分开，部分日期不推断精确日期。
- **环境**：本机直接无环境前缀运行浏览器仍需系统依赖；本轮隔离环境及CI均实际验收。Windows原生运行、正式同源部署、生产发布不属于本次验收。基础凭证模式检查不等于完整安全审计。
- **范围**：无范围内阻塞或需用户决定事项；Excel导入/附件/导出备份和其余M3+功能未开发。T012–T028仍planned，完成M2后停止。
