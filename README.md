# Pokémon Shiny Tracker

基于 Baseline V1.1 的个人闪光收藏与 Hunt 管理项目。T000 仓库初始化已通过验收，T001 提供 Go 后端工程骨架，尚无业务功能。任务的实际验收与合并状态见任务证据。

## 冻结需求

- [Product Spec V1.1](docs/PRODUCT_SPEC_V1.1.md)
- [Task Baseline V1.1](docs/TASK_BASELINE_V1.1.md)：T000–T028，共 29 项任务
- [ADR Register V1.1](docs/ADR_REGISTER_V1.1.md)：含 ADR-015 / ADR-016
- [Data Source Policy V1.1](docs/DATA_SOURCE_POLICY_V1.1.md)
- [AI 工作规范](AGENTS.md)
- [原始基线包说明](README_BASELINE.md)

上述六份原件保持不变，SHA-256 记录在 [基线清单](docs/baseline-manifest.json)。原件中“仓库待创建”等文字反映冻结时状态，实际执行状态见下方记录。

## 开发与验收

- [GitHub 工作流及 AI 执行规范](docs/GITHUB_WORKFLOW.md)
- [Task 状态与证据](docs/tasks/status.json)、[任务记录模板](docs/tasks/TEMPLATE.md)
- [T000 执行记录](docs/tasks/T000.md)
- [T001 执行记录](docs/tasks/T001.md)
- [Go 后端开发与本地启动](docs/BACKEND_DEVELOPMENT.md)
- [前端开发与本地启动](docs/FRONTEND_DEVELOPMENT.md)
- [开发日志](docs/DEVELOPMENT_LOG.md)
- [初始化验收报告](docs/T000_ACCEPTANCE.md)

新任务从最新远程 `dev` 建立 `feature/Txxx-name`，经 PR、CI 和自审后以 Merge Commit 合入 `dev`，保留源分支。正式发布至 `prod` 须另获用户授权。

## 本地检查

安装 Python 3.11 或更高版本及 Git，然后在仓库根目录运行：

```text
python scripts/validate_baseline.py
git diff --check
```

Go 后端使用 Linux Go 1.27.3、Gin、GORM。从仓库根目录运行：

```bash
bash -ic 'make deps'
bash -ic 'make run'
```

当前服务仅监听 `127.0.0.1:8080`，`GET /api/v1/health` 返回进程健康状态；未知路径返回统一 JSON 404。详见 [API 契约](docs/API.md)。可用 `SHINY_HTTP_PORT` 更改端口，`.env.example` 仅为配置说明模板。完整构建、测试和启动验证：

```bash
bash -ic 'go test ./... && go build ./...'
bash -ic 'make check && make smoke'
```

前端位于 `frontend/`，使用 Linux Node 24.22.1、npm 11.19.1：

```bash
bash -ic 'cd frontend && npm ci && npm run dev'
```

打开 `http://127.0.0.1:5173`。类型/组件/构建/启动检查：

```bash
bash -ic 'cd frontend && npm run typecheck && npm test && npm run build && npm run smoke'
```

Actions 包含 Go 检查、前端类型/组件/构建/启动与 Chromium 浏览器检查，以及治理基线、任务一致性、PR 来源和仓库卫生检查。API Client 和前后端联调在 T004 建立，业务功能留给后续任务。

真实 SQLite 数据、Excel 原始资料、密钥、附件和备份放在仓库外。脱敏且获准入库的测试夹具需在对应任务中明确审查。
