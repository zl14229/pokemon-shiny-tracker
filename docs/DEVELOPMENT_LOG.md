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
