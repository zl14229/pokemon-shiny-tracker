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
