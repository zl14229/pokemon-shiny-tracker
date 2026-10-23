# Pokémon Shiny Tracker

基于 Baseline V1.1 的个人闪光收藏与 Hunt 管理项目。当前阶段为 T000 仓库初始化，尚无业务代码。

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
- [初始化验收报告](docs/T000_ACCEPTANCE.md)

新任务从最新远程 `dev` 建立 `feature/Txxx-name`，经 PR、CI 和自审后以 Merge Commit 合入 `dev`，保留源分支。正式发布至 `prod` 须另获用户授权。

## 本地检查

安装 Python 3.11 或更高版本及 Git，然后在仓库根目录运行：

```text
python scripts/validate_baseline.py
git diff --check
```

当前 Actions 只验证治理基线、任务一致性、PR 来源和仓库卫生。Go / Vue 工程、依赖及业务 CI 分别在对应任务中建立。

真实 SQLite 数据、Excel 原始资料、密钥、附件和备份放在仓库外。脱敏且获准入库的测试夹具需在对应任务中明确审查。
