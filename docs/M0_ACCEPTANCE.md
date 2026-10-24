# M0 最终集成验收

- 验收日期：2026-10-24，Asia/Shanghai。
- 结论：**通过**。T000–T004 verified；T005–T028 planned。本轮授权止于 T004，无业务/数据库实现，无生产发布。
- 实际集成验收基线：`70347166e587abbdb4a0dab2c750b1a6be9c55cf`，T004 实现 PR 的真实 dev Merge Commit。
- 本报告通过 T004 验收收尾 PR 归档；该收尾自身 merge SHA 与最终 dev CI 由 GitHub PR/Actions 提供，避免提交自引用。

## 各任务合并与 CI

| Task | 实现 PR | 实现 merge SHA | 最终实现 PR CI | 合并后 dev CI |
|---|---|---|---|---|
| T002 | [实现 PR](https://github.com/zl14229/pokemon-shiny-tracker/pull/5) | `51ba5d81dcf719bb09abc743b3d349b5646889e6` | [PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40055913517) | [dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40056003815) |
| T003 | [实现 PR](https://github.com/zl14229/pokemon-shiny-tracker/pull/7) | `ea46d294a60b02badb48cd916cc7d1075fdddc46` | [PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40056989398) | [dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40057135058) |
| T004 | [实现 PR](https://github.com/zl14229/pokemon-shiny-tracker/pull/9) | `70347166e587abbdb4a0dab2c750b1a6be9c55cf` | [PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40057984013) | [dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40058212554) |

T002 验收收尾 [PR #6](https://github.com/zl14229/pokemon-shiny-tracker/pull/6)，merge `1c3e472f487eca701cbac9882fd8cdb622f7f0b2`；T003 验收收尾 [PR #8](https://github.com/zl14229/pokemon-shiny-tracker/pull/8)，merge `d7eab8093ccc7fa9a72133c916253f30cf4cb3d0`。两次收尾 PR/合并后的 dev CI 均成功。T004 验收收尾包含本报告，其最终结果以对应 PR 为准。

## 依赖与分支

| Task | 依赖验证与顺序 | 独立源分支 | 创建基线 |
|---|---|---|---|
| T002 | T001 verified | feature/T002-router-health | af58aebc57e81e2561ba444d383f5c760f19593a |
| T003 | T000 verified，按授权等待 T002 收尾完成 | feature/T003-vue-frontend-init | 1c3e472f487eca701cbac9882fd8cdb622f7f0b2 |
| T004 | T002/T003 verified 且收尾完成 | feature/T004-api-client-integration | d7eab8093ccc7fa9a72133c916253f30cf4cb3d0 |

每次创建分支前 fetch、检查干净工作区并从最新 dev 创建；未继承上个任务源分支快照。各任务独立实现、自审、PR 和验收收尾，Merge Commit、保留源分支。未伪造人工批准；按 code-review 技能标记 code-reviewed，无配套子技能，未启动子代理。

## 实际验收

所有本地 Go/Node/npm/构建命令在仓库根目录通过交互 Bash 执行。实际工具为 Linux Go 1.27.3、nvm Node 24.22.1、npm 11.19.1。

| 检查 | 结果与范围 |
|---|---|
| `go test ./...`、`go build ./...` | 成功；健康/路由/错误/生命周期等测试 |
| `make check` | 格式、tidy diff、module verify、vet、race、build 成功 |
| `make smoke` | 真实进程健康 200、JSON 404/405、非法配置/端口占用失败、SIGTERM 退出 0 和端口释放成功 |
| `npm ci` | 锁文件可重复安装，审计 0 已知漏洞 |
| `npm run typecheck`、`npm test` | 成功，21 项客户端/组件测试 |
| `npm run build`、`npm run smoke` | 成功；真实开发服务器、Vue 编译、构建预览和资源可读取 |
| `npm run test:integration` | 5 项成功；真实 Go + Vite + API Client 的健康、404、405、构建预览代理、实际后端停机 502 与重启恢复 |
| CI `npm run test:e2e` | 7 项 Chromium 场景成功：桌面/窄屏、真实成功调用，以及 JSON/非 JSON/异常成功体/网络/超时提示与重试 |
| `python3 scripts/validate_baseline.py` | 冻结原件哈希、29 Task/依赖/状态、16 ADR、忽略/凭证规则通过 |
| `python3 -m unittest discover -s scripts -p 'test_*.py' -v` | 7 项治理正反例通过 |
| `git diff --check`、差异/敏感信息自审 | 通过；无真实 SQLite、附件、密钥、备份或运行产物入库 |

浏览器正常链路为真实 Go/Vite；故障只通过测试拦截注入，不添加生产故障端点。Mock 的客户端/组件测试与真实进程/浏览器验收分别记录，没有将 mock 当作真实集成。

## 已解决问题

- T002 smoke 的端口探测会将 TIME_WAIT 误判为监听占用；按 Go socket 的地址重用行为修复后完整复验。
- T004 对 Vitest 串行 API 和 Vite 网关 502 的旧假设修正后，严格类型、所有客户端/组件、构建/启动和真实集成测试重新通过。
- 本机 Chromium 实际启动因缺少 libglib-2.0.so.0 失败；未假称本机通过。CI 安装依赖后真实 Chromium 测试已通过。

## GitHub 规则与数据边界

dev active Ruleset 要求 baseline-checks、repository-hygiene、go-checks、frontend-checks，strict=true、bypass 为空，禁止强推/删除、要求 PR 与讨论解决，仅 Merge Commit。最终回读与 T003 的配置证据一致，见 docs/evidence/github-rulesets-t003.json；实际服务端规则始终为准。

prod 在整个执行期间保持 `dc68988b7c76bf23ae3b68b588d15858f50b742e`，未发布。冻结六份原件哈希未变，其他 Task 未启动。测试仅用临时 loopback 端口，不创建或覆盖个人数据库。

## 剩余风险与范围

- WSL 本机缺少 Chromium 系统共享库，本机浏览器测试未完成；CI 的真实浏览器验收已成功，运行应用不要求本机安装测试浏览器。
- 健康只表示 HTTP 进程可响应，尚无数据库/个人记录业务；Vite 同源代理用于本地开发和预览，正式部署路由属于后续发布验收。
- 首次依赖/浏览器下载依赖网络；基础凭证模式和 npm 审计不等于完整安全审计。

M0 范围内无遗留功能阻塞。后续 Task 须另获用户授权，本轮停止。
