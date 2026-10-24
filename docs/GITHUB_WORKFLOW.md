# GitHub 与 AI 执行工作流

本文件落实 ADR-013/014/015/016；冻结产品规格和 Task 定义以 V1.1 原件为准。

## 分支与交付

1. 读取 AGENTS.md、四份冻结规格、本文件和 `docs/tasks/status.json`；先验证依赖。
2. 获取远程最新 `dev`，在干净工作区从它建立 `feature/Txxx-name`。T000 使用 `feature/T000-repository-baseline`。
3. 将该 Task 标为 `in_progress`，按模板记录目标、依赖、实现、检查、自审、风险、文档和提交。业务/架构变更另走 ADR。
4. 推送任务分支，创建指向 `dev` 的 PR，写入 PR URL；等待真实 CI，修复失败并自审，进入 `in_review`。
5. 依赖满足、验收完成、必需检查通过且讨论解决后，以 Merge Commit 合并；保留源分支。允许已授权的 AI 通过工具合并，不依赖第二个账号伪造批准。
6. 合并结果及 dev CI 核实后，记录 merge SHA，再将 Task 标为 `verified`。源分支通过快进同步该合并提交，再补交证据，经同一任务的收尾 PR 合入 dev；收尾提交自身的合并 SHA 由 GitHub PR 提供，避免在提交内引用自身。
7. 执行范围以当前用户授权的 Task 为准，不能因为依赖满足而自动开始后续任务。当前授权按 T010 → T011 连续执行，每项独立分支/PR/验收收尾；M2 集成验收后停止，不进入 M3，不修改 prod。

状态机：`planned → in_progress → in_review → verified`；阻塞用 `blocked`，产品决定用 `needs_decision`；失败修复返回 `in_progress`。`verified` 必须有实现、自审、检查、提交、PR URL 和真实合并 SHA。任务定义在冻结表中唯一维护，状态注册表镜像其编号、里程碑、依赖和最低验收，由 CI 比对。

## GitHub 配置目标

- 默认分支 dev；只允许 Merge Commit；关闭合并后自动删除分支。
- dev/prod 规则 active；要求 PR、解决讨论、禁止删除/强推；bypass 为空。
- 单账号开发 required approvals 为 0；AI 自审是交付证据，不是 GitHub 人工批准。
- 将 `baseline-checks` 与 `repository-hygiene` 设为必需检查，要求分支跟上 base；不能绕过失败检查。T001 已为 dev 追加必需的 `go-checks`；prod 保持原规则，后续发布另行验收。
- Actions 默认 token 只读；只允许 GitHub 自有及 Marketplace 已验证创建者的 Actions；所有外部贡献者的 fork PR workflow 须批准。
- 使用 `pull_request`，不以 `pull_request_target` 执行外部分支代码；凭证不传给检查脚本。
- CI 使用最小权限、超时、并发取消；Action 固定到完整提交 SHA。升级须核对上游。

Actions 本身不会设置或证明 GitHub 保护规则。实际观察、能力限制和证据必须写入验收报告；若未配置必需状态检查，不得声称服务端会自动阻止红灯合并。

## 当前状态与冻结文档的时间差

- 新仓库已存在，dev 初始提交为 dc68988b7c76bf23ae3b68b588d15858f50b742e，无旧业务历史。
- prod 已由用户提前建立，仍为初始提交；保留它且不作发布。实际发布仍遵循用户授权和 M3 验收。
- `task/TASK-000-repository-baseline` 是早期 CI 预备分支，保留历史；新工作按冻结命名使用 feature/T000-repository-baseline。
- README_BASELINE 中“未入库”是原包交付时的描述，当前进展以任务证据和 GitHub PR 为准。

## AI 自主范围

普通实现、修复、自审、日志、提交、推送及范围内 PR 可继续执行。冻结业务/技术栈变更、真实数据毁损、新权限/费用、强制改写历史、生产发布和不可恢复阻塞须暂停。没有后台执行机制时，不宣称结束对话后持续编码。

真实数据存仓库外；Git 忽略规则只防误入库，不能替代秘密检测和人工审查。原始 Excel 和用户媒体不上传到 PR。T000 不访问数据源、不采集资料、不初始化数据库、不安装 Go/Vue 业务依赖。

## T001 工程检查补充

Go CI 验证格式、module 一致性、锁定依赖、vet、普通/竞态测试、构建和真实本地进程启动/退出。`dev` 的 active Ruleset 已通过 GitHub API 写入并回读三项必需检查，严格最新基线与无 bypass 保持不变。实际回读证据见 `docs/evidence/github-rulesets-t001.json`；T000 的历史证据保留。

## T003 前端检查补充

T003 为 dev 追加第四项必需检查 `frontend-checks`，包含 Node 锁定依赖、严格类型、组件测试、构建、开发/预览 HTTP 启动验证和 Chromium 浏览器验收。已通过 GitHub API 写入并回读，保留原三项、strict=true 和无 bypass；prod 未改动。当前回读见 `docs/evidence/github-rulesets-t003.json`。
