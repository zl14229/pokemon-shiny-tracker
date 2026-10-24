# M1 集成验收报告

- 日期：2026-10-24（Debian WSL / Linux amd64）。
- 范围：仅 T005–T009；M0 前置任务及这五项均已 verified。T010–T028 保持 planned，未进入 M2，未发布 prod。
- 集成复验基准：T009 实现 PR #19 的实际 dev 合并提交 `e36373d83c7d5c6b66f30e80115c06462025b122`；先核实实现 PR、合并结果和 dev CI，再更新 verified 并走验收收尾 PR。
- 工具链：Linux Go 1.27.3、CGO/GCC、Node 24.22.1、npm 11.19.1；从项目根目录通过交互 Bash 加载 Go PATH/nvm。未重新安装工具链。

## Task / 提交 / CI 证据

### T005 — 数据库基础模型/迁移/关联及轻量历史

- 状态：verified；分支 `feature/T005-sqlite-models` 从当时最新远程 dev 创建，保留源分支。
- 实现：GORM/SQLite 基础六实体、稳定本地身份、0..N 成果/可空 Hunt、形态复合外键、CHECK、轻量历史与 soft delete、仓库外路径、显式版本化事务迁移。
- [实现 PR](https://github.com/zl14229/pokemon-shiny-tracker/pull/11)；初次实现提交 `d5dfaea8656f034ad1ef69580109f34a574d0f8c`，最终 PR HEAD `71e722d4e88c38eaef6370e307b86b6d29c6e46c`。
- 实际实现 merge SHA：`d5d42679079ac44af4f7ffa8e3cdf2af3dd0821d`。
- [最终 PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40062408488)、[实现合并后 dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40062568518)：四项检查全部成功。
- [验收收尾 PR #12](https://github.com/zl14229/pokemon-shiny-tracker/pull/12) 已合并，merge SHA `465eaf69c1c792b924c8e36ea8505371a7708387`；[收尾 PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40062902359)、[收尾 dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40063076744) 均成功。
- 详细测试、自审及风险：[任务记录](tasks/T005.md)。

### T006 — PokéAPI 基础资料导入、本地种子缓存与查询接口

- 状态：verified；分支 `feature/T006-reference-data` 从当时最新远程 dev 创建，保留源分支。
- 实现：PokéAPI 串行节流/原字节 SHA 缓存、规范化暂存/差异/事务新增与幂等导入、SQL v2 来源批次/外部 ID 映射、3 物种/3 基础形态/4 版本离线种子与本地查询，中文 source_provided/unknown/conflict，覆盖 partial。
- [实现 PR](https://github.com/zl14229/pokemon-shiny-tracker/pull/13)；初次实现提交 `9ae490d66d7a415f5643d3e1f044b2e10d3f3d65`，最终 PR HEAD `c8655fb9686dd671be0cba2cbe30cc45fd94f51d`。
- 实际实现 merge SHA：`f51dea775f5b8bd693324e23acc955c869bbda09`。
- [最终 PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40065170340)、[实现合并后 dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40065402167)：四项检查全部成功。
- [验收收尾 PR #14](https://github.com/zl14229/pokemon-shiny-tracker/pull/14) 已合并，merge SHA `dc36c0e530174a9fabb776273970e66eeffe3b2f`；[收尾 PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40082052807)、[收尾 dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40082243752) 均成功。
- 详细测试、自审及风险：[任务记录](tasks/T006.md)。

### T007 — Hunt CRUD 与状态/累计值

- 状态：verified；分支 `feature/T007-hunt-crud` 从当时最新远程 dev 创建，保留源分支。
- 实现：Hunt service/DTO/API CRUD、事务状态保护、关联个体保护、可空累计/单位/耗时和部分精度日期验证；不推断可刷性、不限制意外闪光物种。
- [实现 PR](https://github.com/zl14229/pokemon-shiny-tracker/pull/15)；初次实现提交 `879da2c916a9ef45bccc224a57f19cc9302ec5a1`，最终 PR HEAD `b8f7a848776f2f8bb3cb9a37adb16040db18ada5`。
- 实际实现 merge SHA：`7555c331277de45742970b41c9b2e53594bec50c`。
- [最终 PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40082676847)、[实现合并后 dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40082901088)：四项检查全部成功。
- [验收收尾 PR #16](https://github.com/zl14229/pokemon-shiny-tracker/pull/16) 已合并，merge SHA `939406b5b625ae938fef26e5a7cf512f26231857`；[收尾 PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40083106956)、[收尾 dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40083267629) 均成功。
- 详细测试、自审及风险：[任务记录](tasks/T007.md)。

### T008 — ShinyRecord CRUD/关联/持有与历史变化

- 状态：verified；分支 `feature/T008-shiny-record-crud` 从当时最新远程 dev 创建，保留源分支。
- 实现：独立/关联个体 CRUD、原始与当前字段分离、默认未知来源/持有、原 ID 进化/形态/位置/持有变化与前后快照、只读历史和软删除、完成 Hunt 最后成果保护。
- [实现 PR](https://github.com/zl14229/pokemon-shiny-tracker/pull/17)；初次实现提交 `8225d3e61ab3d732abc47e735743b59b46f1e963`，最终 PR HEAD `4d89dac97f6b4d2026cc741a8fa277fd96d57b41`。
- 实际实现 merge SHA：`c9ae1ced0614aa9b256399016c0952eb1144d7ad`。
- [最终 PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40083753536)、[实现合并后 dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40083990124)：四项检查全部成功。
- [验收收尾 PR #18](https://github.com/zl14229/pokemon-shiny-tracker/pull/18) 已合并，merge SHA `7aa5c7f4fef483d1dd598b8fddd6a6343d30dc8f`；[收尾 PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40084308302)、[收尾 dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40084536027) 均成功。
- 详细测试、自审及风险：[任务记录](tasks/T008.md)。

### T009 — 查询、搜索、筛选、分页

- 状态：verified；分支 `feature/T009-query-filter-pagination` 从当时最新远程 dev 创建，保留源分支。
- 实现：Hunt/个体白名单组合查询、原始/当前分角色引用、字面搜索、日期区间重叠、同单位累计值边界、空值后置/ID次级排序、事务 total/分页；M1 真实进程验收接入 make smoke/必需 go-checks。
- [实现 PR](https://github.com/zl14229/pokemon-shiny-tracker/pull/19)；初次实现提交 `de337bcd3d23d5c08afbeb67ad4eecf678cb0a9d`，最终 PR HEAD `0f27fb33d9ba6413d7872cc7b54fe7bb083e2f65`。
- 实际实现 merge SHA：`e36373d83c7d5c6b66f30e80115c06462025b122`。
- [最终 PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40085147745)、[实现合并后 dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40085461939)：四项检查全部成功。
- 本报告与 verified 证据由同一 T009 源分支的验收收尾 PR 提交；收尾自身 merge SHA/dev CI 由 GitHub PR/Actions 提供，避免提交自引用，最终执行回复给出实际结果。
- 详细测试、自审及风险：[任务记录](tasks/T009.md)。

## 集成验收结果

| 验收项 | 结果与证据 |
|---|---|
| 从零初始化/重开 | 通过；真实二进制在自建空临时库启动，SQL user_version=2、重启后状态/资料/历史保留 |
| 模型/约束/关联/迁移 | 通过；真实 SQLite 外键、形态归属、数值/枚举约束、v1→v2、不识别/未来版本不覆盖、迁移失败事务回滚；PRAGMA foreign_key_check 无违规 |
| 基础资料离线查询 | 通过；随程序打包3物种/3基础形态/4版本，无启动联网；稳定内部/外部ID和来源映射，中文缺失/冲突明确；T006真实采集后使用 offline 模式从缓存重建字节一致 |
| Hunt/个体独立和关联使用 | 通过；历史个体可无 Hunt，一个 Hunt 可有零/多个/意外成果；状态转换不删除成果，完成及最后成果移除保护双向成立 |
| 进化/位置/持有/历史 | 通过；变更保持原个体ID，原始/当前分离，registered/evolution/form_change/location_change/holding_change/updated/deleted保留快照；事件日期不伪造；历史失败注入验证创建/更新/删除原子回滚 |
| CRUD/查询/筛选/分页与异常 | 通过；真实 HTTP 进程与路由/SQL测试覆盖严格JSON、关联/状态/边界、400/404/409/413/415/500、字面搜索/注入输入、组合筛选、角色区分、空值/部分日期、排序同值/跨页/总数 |
| 源不可用不损坏个人记录 | 通过；来源HTTP503/无效响应/缓存失败、资料冲突和导入回滚测试保留原有个体；日常 CRUD/查询只读本地资料，无同步覆写个人记录路径 |
| Go/真实启动 | `go test ./...`、`go build ./...`、`make check`（格式/module verify/tidy/vet/race/build）及 `make smoke` 成功；测试缓存命中保持原结果，真实进程 smoke 在合并提交上重新执行 |
| 前端兼容性回归 | `npm ci`（本次审计0个已知漏洞）、类型、21项组件/客户端、构建/HTTP smoke、5项真实 Go/Vite/API Client 集成全部成功 |
| 浏览器 | 最终实现 PR 与 dev 的 frontend-checks 真实 Chromium 成功；本机 WSL 缺少系统共享库，未虚报本地浏览器成功 |
| 治理/敏感信息 | 冻结哈希、29项任务/依赖、16 ADR、7项治理回归、忽略路径、基础凭证模式、暂存与合并差异检查通过；未提交个人数据库、附件或认证材料 |

真实进程集成入口为 [smoke_records.py](../scripts/smoke_records.py)，已接入 `make smoke`，由必需 go-checks 执行。模型/导入/业务与查询测试分别在 internal/database、internal/reference、internal/service 和 internal/api。未使用真实用户数据库，所有测试数据库/触发器仅在自建临时目录创建并清理。

## 自审、文档和 GitHub 规则

逐任务完成代码差异、输入/关联/空值、事务/迁移、数据源边界、身份/审计与敏感内容自审，修复发现的问题后重新验证；无已知未解决的范围内缺陷，PR 已标记 code-reviewed，未伪造人工批准。没有配套 code-review-* 子技能，未启动子代理或留下未授权 GitHub 评论。

文档包括 DATABASE.md、REFERENCE_DATA.md、API.md、BACKEND_DEVELOPMENT.md、README、五份任务记录、status.json 和 DEVELOPMENT_LOG。冻结规格及 ADR 未改变。后端业务 API 已可使用；前端业务界面仍归 M2。

GitHub dev Ruleset 实际回读：[M1 证据](evidence/github-rulesets-m1.json)。active、四项必需检查、strict 最新基线、PR/讨论解决、禁止强推/删除、bypass为空，单账号 approvals=0；只用 Merge Commit。一次检查仍运行时的合并请求被 GitHub 拒绝，待全部成功后正常合并，未绕过规则。五个源分支已从远程读取确认保留。prod 实际读取仍为 `dc68988b7c76bf23ae3b68b588d15858f50b742e`，本轮无 prod 写入。

## 保留风险与范围外事项

- 参考资料是已校验的小范围种子，非全图鉴/全游戏覆盖；中文仅源提供/未知/冲突，尚未进行52Poké交叉核验。物种/形态存在不证明任何游戏可合法直接刷闪，获取机制/概率未实现。
- 进化为用户报告，不代表已验证完整进化图与游戏条件；轻量快照不构造不存在的中间状态，完整转移管理仍属 T023。
- 部分日期筛选表示可能日期区间相交，不能证明精确发生日期。分页在数据不变时稳定；跨请求编辑会移动页边界，未实现跨请求快照/游标或大数据性能承诺。
- SQLite 构建依赖 CGO/GCC，Windows 原生启动未验收；本机 Chromium 缺系统库，浏览器验证由 CI 承担。基础凭证模式检查不等于完整安全审计。
- 完整资料更新/回滚管理、Excel、附件、备份、统计、HuntSession、可获取性/Planner/Bingo、生产发布均按后续任务授权实施。本轮不启动 T010 或其他 M2+ 工作。
