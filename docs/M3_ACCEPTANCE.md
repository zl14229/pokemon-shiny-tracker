# M3 功能验收报告

- 日期：2026-10-25，Asia/Shanghai；执行环境：Debian WSL / Linux amd64。
- 授权及完成范围：T012 → T013 → T014 → T015，各任务按依赖从当时最新远程 dev 独立创建 feature 分支。T000–T015 均 verified；T016–T028 均 planned，未开始M4。
- 验收结论：本地MVP主要用户路径可完整运行；本报告记录授权范围内的验收，不代表全产品后续里程碑已实现，也不代表整份个人收藏已迁移。
- 最终实现验收基准：T015实际 dev merge `fc60cbff50d9db93695cae99e184b2ac944b80a7`。全量本地回归、实现PR CI、合并后dev CI均通过后，再在该dev上复验完整MVP与真实样本只读检查，随后才归档本报告。
- 本报告由T015验收收尾PR入库；收尾自身merge/dev CI以对应GitHub PR/Actions及最终执行回复为准，避免提交自引用。源分支保留；没有直接提交dev、强推、修改prod、正式Release或公网部署。

## 每项Task的提交、PR、CI与验收收尾

### T012 — Excel 导入、预览、映射、重复检测

- 状态 verified；分支 `feature/T012-excel-import`；[详细证据](tasks/T012.md)。
- 实现：受限只读XLSX读取、显式字段/值映射、逐行预览/错误/疑似重复/公式与未映射警告；确认选中行后事务创建无Hunt个体、历史与批次/行溯源，幂等重试防重；中文数据管理、批次核对入口。
- 初次实现commit `e577ff47c7633a75eaf568f5e0885bfb249fc8ea`；最终PR HEAD `000365ec9427f901407f20d8c2573815d4b78c39`。
- [实现PR #25](https://github.com/zl14229/pokemon-shiny-tracker/pull/25)；实际merge `2daf1e24d4a38853b85d9255fcba48a28fa2e5e3`。
- [最终PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40170737923)、[实现合并后dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40171021238)：四项必需检查全部success。
- [验收收尾PR #26](https://github.com/zl14229/pokemon-shiny-tracker/pull/26)，merge `84390c058e43fd3771c9b9e6760a87767eb98173`；[收尾PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40171231932)、[收尾dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40171564529)全部success。

### T013 — 附件/链接

- 状态 verified；分支 `feature/T013-attachments`；[详细证据](tasks/T013.md)。
- 实现：闪光个体附件/HTTP-HTTPS链接元数据与详情组件；仓库外私有不可变文件、随机键独占写入、名称/路径/权限/格式/大小/尺寸/哈希校验，失败清理仅新文件；查看/下载、链接不抓取、软删除关联与明确保留策略。
- 初次实现commit `f2cd0ea77da3b86b7d917c29fd842b6d2c8c110d`；最终PR HEAD `86d01810a7743bef059df4438d220136dfc8f923`。
- [实现PR #27](https://github.com/zl14229/pokemon-shiny-tracker/pull/27)；实际merge `59ee9a5746b848081469f77121db2ac320e73da5`。
- [最终PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40172780407)、[实现合并后dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40172996313)：四项必需检查全部success。
- [验收收尾PR #28](https://github.com/zl14229/pokemon-shiny-tracker/pull/28)，merge `cc029dfd7d057e9b3a2847dad0451d9f90d47307`；[收尾PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40173289632)、[收尾dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40173441318)全部success。

### T014 — 导出及本地基础备份

- 状态 verified；分支 `feature/T014-export-backup`；[详细证据](tasks/T014.md)。
- 实现：只读SQLite在线快照、全应用表/列JSON导出、含引用附件与manifest的ZIP生成并完整verify后独占发布；CLI备份/导出/校验/只向新目录原子恢复，UI本地下载与数据导航；保守确认上传失败清理引用，防止不明提交误删。
- 初次实现commit `6b07416d377743d8e8d920d77399533ee64b6ffc`；最终PR HEAD `8146b9d849d5b1667861acedaac803083d6b63f4`。
- [实现PR #29](https://github.com/zl14229/pokemon-shiny-tracker/pull/29)；实际merge `1db235fadaf86d734dfe836376ec594ed38a89e7`。
- [最终PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40175263921)、[实现合并后dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40175587296)：四项必需检查全部success。
- [验收收尾PR #30](https://github.com/zl14229/pokemon-shiny-tracker/pull/30)，merge `a86e109082fb4bd27f9dd11929cd34319c013215`；[收尾PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40175962778)、[收尾dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40176200017)全部success。

### T015 — MVP 端到端验收

- 状态 verified；分支 `feature/T015-mvp-acceptance`；[详细证据](tasks/T015.md)。
- 实现：单条真实浏览器完整MVP故事串联Hunt/关联成果/收藏编辑/独立Excel确认导入/附件链接/JSON与ZIP下载，CLI校验恢复到新目录后启动自建恢复Go/Vite并核对UI/API/身份/历史/批次/字节；真实样本按实际离线种子复核并修正历史口径；默认下载名忽略保护。
- 初次实现commit `011115b901baf3b165747ba7c9574b380b33f814`；最终PR HEAD `c0786485ad7956137ec42fd9a7ea927dbc31af69`。
- [实现PR #31](https://github.com/zl14229/pokemon-shiny-tracker/pull/31)；实际merge `fc60cbff50d9db93695cae99e184b2ac944b80a7`。
- [最终PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40178356419)、[实现合并后dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40200048779)：四项必需检查全部success。
- 本报告、真实状态与规则证据由本任务独立收尾PR归档；收尾自身结果由GitHub提供，最终回复给出实际merge/CI。

## MVP主要路径与数据恢复结果

| 验收项 | 实际执行结果 |
|---|---|
| 创建和管理Hunt | 浏览器创建0次/eggs任务，修改为12次/90秒，登记关联成果后完成；未知结束日期保持null |
| 出闪登记与收藏编辑 | 明确选择自刷/持有/部分获得日期，登记个体并关联Hunt，编辑当前位置为HOME及备注；原始物种/游戏未知仍为null，保持同一个体ID |
| Excel历史导入 | 匿名运行时XLSX逐列映射，确认前记录数不变，错误行不能选；选中有效行并确认后新增独立无Hunt个体和批次/原值溯源，疑似重复不会自动覆盖 |
| 附件与外部链接 | 上传匿名PNG/文本并核对内容，关联HTTP/HTTPS链接、移除关联；路径/类型/大小/权限/损坏/无覆盖/软删除保留/结果不明处理有自动测试，链接不抓取 |
| JSON导出 | 下载实际数据文件，包含11张应用表及全部列、ID/NULL/日期精度/历史/批次/软删除，文件原件另由ZIP保存 |
| ZIP备份及校验 | 在线SQLite快照、引用的不可变文件与manifest，生成后完整verify；缺失/损坏/路径/清单不符均整份失败 |
| 恢复后真实使用 | 浏览器下载ZIP→本地CLI verify→全新隔离目录restore→启动该目录独立Go和Vite→浏览器/真实API核对Hunt、个体、导入记录/批次、轻量历史和附件字节 |
| 可重复与无覆盖 | 同一归档恢复到两个不同新目录通过；已有输出文件/恢复目录保留不变；并发创建/上传/软删除时备份仍完整；取消不发布部分备份 |

[完整MVP浏览器故事](../frontend/tests/e2e/mvp.spec.ts)、[下载与恢复流程](../frontend/tests/e2e/backup.spec.ts)、[真实SQLite恢复/并发/损坏/无覆盖测试](../internal/backup/backup_test.go)、[CLI](../cmd/shiny-data/main.go)提供可重复执行的证据。恢复只停止本测试子进程，只清理自建临时目录，未覆盖或读取个人数据库。恢复后桌面/390px截图视觉检查完成，无页面横向溢出，完整故事捕获的页面异常为0。

## 真实Excel样本与迁移边界

- 原始工作簿始终只读：15表、6756个存储行（包含模板/空白结构）。10表在本组表头映射下形成1712个非空候选行；另外5表未由此映射覆盖。
- 首轮T012使用2物种隔离引用夹具，真实结果7行格式/引用可用、1705错误、2973提示；T015自审发现覆盖口径与启动不一致，已改为程序相同的离线 `reference.EnsureSeed`（3物种）实际复验。
- 最终实际启动种子结果：**8行格式/引用可用、1704行错误、2972提示**。两轮原文件SHA256前后均一致、个人写入0。可用行不代表确认8只已获得个体，模板行仍需人工判断；本次没有确认或批量写入真实收藏。
- 额外只读核对现有资料范围内8行编号/姓名缓存，8/8对应，支持全国编号字段的语义；不据此补造未知物种/形态/游戏/日期/途径或Hunt。
- 原始Excel、真实行值、文件名/路径/指纹、个人媒体/SQLite/备份均未上传GitHub。仅汇总统计归档；CI只使用运行时匿名样本，不冒称CI做过真实数据验收。

当前资料仅部分物种/形态/游戏，不覆盖整份收藏。未知资料会明确报错或警告，不静默忽略；未映射非空列在确认后保留于本地批次溯源。原文件未改、没有声称整份迁移完成，后续丰富资料与其他表的明确映射仍需按授权范围推进。细节见 [IMPORT.md](IMPORT.md)。

## 实际测试与环境

从项目仓库使用 `bash -ic` 加载Linux工具链：Go1.27.3、Node24.22.1、npm11.19.1。

- `go test ./...`、`go build ./...`、`make check`（格式/module一致性/verify/vet/race/build）、`make smoke`（真实进程、持久化、CRUD/历史/重启）：成功。缓存命中沿用有效结果，进程smoke实际重新执行。
- `npm ci`、`npm run typecheck`、`npm test`（63项）、`npm run build`、`npm run smoke`、`npm run test:integration`（8项真实Go/Vite/API Client）：成功。
- `npm run test:e2e`：26项真实Chromium成功，含完整恢复后服务/界面；CI颜色环境下也完整复验成功，首次CI失败未合并、修复后才继续；在实际T015合并dev上再次实际执行完整MVP故事和真实样本只读复验。
- 冻结哈希、29Task/依赖/状态、16ADR一致性、7项治理回归、差异/忽略/基础凭证模式检查：成功。增加默认浏览器备份/导出文件名的忽略探针，降低误放源码目录后提交的风险。
- 本机原缺Chromium共享库/字体，仅从Debian公共包解包到仓库外临时目录，以命令限定的字体/动态库环境运行；没有重新安装系统Go/Node或修改Shell配置，也未将机器临时路径固化到项目。CI使用标准浏览器依赖安装。
- SQLite/xsys仅从既有锁定间接依赖改为直接使用，版本不变；前端无新运行时依赖。CI同时构建后端与恢复CLI，四项必需检查、Action完整SHA和只读权限不变。

## 数据正确性与已修复问题

SQLite结构由v2逐步增量至v3导入批次/v4附件，既有个人身份/关联/历史保留，失败迁移/事务回滚和v1升级有回归。导入仅用户明确确认后原子新增，不更新既有个人记录；未知不补造，无Hunt历史不造Hunt。附件随机键独占写入/哈希验证，提交不明时保守保留，备份不会静默遗漏引用文件。恢复全面检查后Linux不替换原子发布，文件/目录同步；没有Web覆盖恢复或自动切换个人数据目录。

开发中发现的ZIP目录条目兼容、迁移注册遗漏、smoke计数断言误改、隐藏checkbox测试定位、空清单比较/初版目录发布方式、真实样本夹具口径、恢复Vite就绪检查的ANSI颜色标记匹配均已修复并重跑，没有跳过失败检查。无已知未解决的M3范围内缺陷；以下边界继续明确保留。

## 当前可用与明确不支持

可用：本地离线基础资料查询、Hunt与收藏CRUD/状态/关联和轻量历史、列表搜索筛选分页、中文桌面/基本响应式界面、XLSX映射预览/错误重复提示/确认导入与批次、图片/文件/链接、本地JSON导出/含文件ZIP备份/CLI新目录恢复。启动与恢复操作见 [README](../README.md)、[前端开发](FRONTEND_DEVELOPMENT.md)、[备份方法](BACKUP.md)。

不支持或未验收：全国/形态/游戏完整覆盖、全量真实收藏自动迁移、旧XLS/XLSM/宏执行、JSON恢复/已有目录覆盖、SVG/HTML/可执行或视频文件上传、外部网页内容归档、附件物理清理、多用户/账户/云同步/公网部署、Windows原生恢复/Windows挂载目录、完整图鉴/Random Hunt/Bingo/统计/会话计时/完整转移与增量资料更新等后续任务。界面当前以浅色为主，深色主题未验收。

## 剩余风险与未验证项

1. 基础资料部分覆盖、中文未交叉核验、5个工作表未由当前映射覆盖；8行可用不等于获得事实，真实整份收藏迁移没有执行。公式只用缓存值，需人工核对；疑似重复不是身份判定。
2. 当前仅schema4/本归档格式；ZIP含软删除文件，不含原Excel/外部内容/孤立文件。JSON是数据导出，不是恢复格式。恢复仅Linux且需文件系统支持不替换原子发布，Windows与挂载目录未验收。
3. 备份未加密，SHA用于完整性而非来源认证；只恢复可信私有备份。文件上限/数据库512MiB/总量2GiB/2分钟期限均明确，最大规模压力、真实断电和正式部署未测试；本地备份不等于异地灾备。
4. 软删除与崩溃孤立文件仍占磁盘，无自动物理清理；请勿手动破坏引用文件。目录同步失败后新目标可能已经存在，需要先核对而非覆盖重试。
5. 继承M2客户端前置版本比对非原子锁，跨窗口同时编辑仍有竞态；跨请求分页无快照。基础模式扫描不等于完整安全/媒体审计。本机浏览器依赖为临时环境，普通应用工具链未重装。
6. GitHub runner/部分Action运行时有上游迁移提示，当前必需检查实际成功；未在本轮扩展平台或升级Action。没有权限/费用/真实数据破坏阻塞，没有正式发布授权。

## 治理、证据与停止边界

[实际Ruleset回读](evidence/github-rulesets-m3.json)：dev active、四项必需检查、strict=true、讨论须解决、Merge Commit、禁止强推/删除、bypass为空。每项先通过实现/测试/自审与CI合并，再核实dev CI标记verified，经收尾PR归档，保留源分支。没有伪造人工批准或未经授权GitHub评论；code-review无配套code-review-*技能，未启动子代理。

prod实际回读仍为 `dc68988b7c76bf23ae3b68b588d15858f50b742e`。本次没有合并prod、创建正式Release/Tag或部署公网；M3完成后停止，T016–T028保持planned，等待后续M4授权。

## 新增与修改文件

以下为M3相对起点dev的实际文件清单（含本收尾报告/证据）；逐任务实现摘要、检查与风险见对应Task记录。

```text
.github/workflows/ci.yml
.gitignore
Makefile
README.md
cmd/shiny-data/main.go
cmd/shiny-data/main_test.go
docs/API.md
docs/ATTACHMENTS.md
docs/BACKUP.md
docs/DATABASE.md
docs/DEVELOPMENT_LOG.md
docs/FRONTEND_DEVELOPMENT.md
docs/GITHUB_WORKFLOW.md
docs/IMPORT.md
docs/M3_ACCEPTANCE.md
docs/evidence/github-rulesets-m3.json
docs/tasks/T012.md
docs/tasks/T013.md
docs/tasks/T014.md
docs/tasks/T015.md
docs/tasks/status.json
frontend/src/App.vue
frontend/src/api/attachments.test.ts
frontend/src/api/attachments.ts
frontend/src/api/client.test.ts
frontend/src/api/client.ts
frontend/src/api/imports.test.ts
frontend/src/api/imports.ts
frontend/src/components/AttachmentsPanel.vue
frontend/src/components/DataNav.vue
frontend/src/pages/BackupPage.vue
frontend/src/pages/ImportPage.vue
frontend/src/pages/ShinyDetail.vue
frontend/tests/e2e/attachments.spec.ts
frontend/tests/e2e/backup.spec.ts
frontend/tests/e2e/import.spec.ts
frontend/tests/e2e/mvp.spec.ts
go.mod
internal/api/attachments.go
internal/api/attachments_test.go
internal/api/data.go
internal/api/data_test.go
internal/api/imports.go
internal/api/imports_test.go
internal/api/local.go
internal/api/router.go
internal/backup/archive.go
internal/backup/backup_test.go
internal/backup/export.go
internal/backup/publish_linux.go
internal/backup/publish_other.go
internal/backup/snapshot.go
internal/database/database.go
internal/database/migrate.go
internal/files/store.go
internal/files/store_test.go
internal/model/attachment.go
internal/model/import.go
internal/service/attachments.go
internal/service/attachments_test.go
internal/service/imports.go
internal/service/imports_test.go
internal/testutil/workbook.go
internal/workbook/read.go
internal/workbook/read_test.go
scripts/anonymous_workbook.py
scripts/smoke_records.py
scripts/validate_baseline.py
```
