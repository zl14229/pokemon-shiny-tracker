# 开发日志

日期采用 Asia/Shanghai。历史任务的详细执行与证据以 `docs/tasks/Txxx.md` 为准。

## 2026-10-24 — T002

- 从最新 dev `af58aebc57e81e2561ba444d383f5c760f19593a` 创建 `feature/T002-router-health`，确认 T001 verified，工作区干净。
- 增加 `/api/v1/health`、统一 JSON 404/405/500、内部错误和 panic 恢复；不连接数据库，不引入业务端点。
- 更新原有真实 HTTP 生命周期测试与进程验证脚本。首次 smoke 检查遇到端口重绑的 `Address already in use`；分析为 TCP TIME_WAIT 对探测 socket 的影响，将探测按 Go listener 的行为启用 SO_REUSEADDR，仍会拒绝真实占用中的监听端口。
- 测试、PR、合并和 CI 的实际最终结果在任务记录中归档。

- T002 实现 PR https://github.com/zl14229/pokemon-shiny-tracker/pull/5 已合并（51ba5d81dcf719bb09abc743b3d349b5646889e6），PR CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40055913517 与 dev CI https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40056003815 成功；本任务标记 verified 并走验收收尾 PR。
