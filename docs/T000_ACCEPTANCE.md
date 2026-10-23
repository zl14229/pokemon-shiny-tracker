# T000 初始化验收报告

状态：T000 验收通过。业务任务 T001–T028 均保持 planned。本报告的证据归档通过独立收尾 PR 进入 dev；该收尾 PR 自身的最终合并提交以 GitHub 记录为准。

- 仓库：https://github.com/zl14229/pokemon-shiny-tracker
- 默认分支：dev；初始化起点：dc68988b7c76bf23ae3b68b588d15858f50b742e
- 工作分支：feature/T000-repository-baseline；保留旧预备分支
- 实现提交：911684857b9b20f34d961945d1ca122e10a7da6c（初始导入 0ffc2eae07f605a4c14fa2411c301bffc88c1075）
- 已合并 PR：https://github.com/zl14229/pokemon-shiny-tracker/pull/1
- dev 验收合并提交：5781ec3840cca34ab7136f041f06419fd2766eca
- 最终 PR CI：https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/39980288279（success）
- 合并后 dev CI：https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/39980414107（success）

## 已核实配置

- 公开新仓库；默认 dev；只允许 Merge Commit；auto-merge 可用；delete_branch_on_merge=false。
- protect-dev（25725833）、protect-prod（25725884）：active，准确匹配 dev/prod，禁止删除和强推，要求 PR、讨论解决，approvals=0，allowed_merge_methods=[merge]，bypass=[]。
- 两套规则均已通过登录后的 GitHub 设置页补齐 required_status_checks，并以 GitHub REST 读取结果复核：baseline-checks、repository-hygiene，integration_id=15368（GitHub Actions），strict_required_status_checks_policy=true，无创建豁免。
- Actions 设置页已独立复核：允许仓库自身、GitHub 自有及 Marketplace 已验证创建者；默认令牌 contents/packages 只读；全部外部贡献者的 fork PR workflow 需批准；Actions 创建/批准 PR 选项关闭。当前工作流中的 Actions 均已固定完整 SHA；仓库全局强制固定 SHA 开关未开启。
- prod 提前存在但未发布，仍为初始提交。

## 验证范围

| 验证 | 结果 |
|---|---|
| 六份冻结原件逐文件 SHA-256 与 ZIP 对照 | 一致；清单见 baseline-manifest.json |
| 29 Task 编号、里程碑、交付、依赖、最低验收；依赖无环 | 通过，与冻结表逐字段相同 |
| ADR-001–016 完整性；ADR-015/016 内容 | 通过，无产品决策改动 |
| `python scripts/validate_baseline.py` | 本地及 GitHub CI 通过 |
| `python -m unittest discover -s scripts -p 'test_*.py' -v` | 7 项测试通过，含冻结变更、依赖漂移、验收漂移、虚假完成、强制跟踪个人数据、错误 PR 命名等反例 |
| `git diff --cached --check` / CI whitespace | 通过 |
| 两个 required checks 来源及严格最新基线模式 | GitHub REST 实际返回确认（integration_id=15368） |
| 服务端门槛实际生效 | PR 说明编辑触发重跑后，合并被 405 拒绝，提示两项检查运行中；等待 success 后以普通 Merge Commit 成功合并，无 bypass |
| 合并后 dev CI | 两项作业均通过 |
| 数据/密钥/附件误入库检查 | 忽略样例与强制跟踪反例通过；不等同于全功能秘密扫描 |
| 干净历史/范围 | 基于新仓库初始提交；无业务实现、真实记录或旧实现提交 |

## 自我审查

- 产品冻结内容不变；无业务代码、数据库、资料抓取或真实用户数据。
- 任务状态不以生成文档为完成；只有实际 PR/检查/merge 证据齐全才能 verified。
- CI 当前只覆盖仓库治理，Go/Vue 构建测试延后对应工程任务。
- Task 分支保留；不强推、不删除分支、不发布 prod。

## 审查修复与边界

自审修复：Windows 文本 stdin 换行会误报 Git 忽略规则，改用 NUL 分隔；“缺少合并证据不得 verified”的反例主动清空证据，确保 T000 完成后仍有效。修复后测试和 CI 通过。未发现剩余阻断问题。

仓库全局“强制 Action 固定 SHA”开关未启用，当前工作流自身已固定 SHA。AI 自审通过 PR 标签和任务证据留档；approvals=0 适合当前单账号，并不等于人工双人审查。规则仍要求 PR、CI 及讨论解决。

T000 初始化无遗留开发事项。Go/Vue 工程、业务 CI、真实 Excel 字段和数据源覆盖/许可核验均属于后续冻结任务，不在本次实施范围。prod 未发布。保留符合命名的任务分支及早期 task/TASK-000-repository-baseline 历史分支。当前 Windows checkout 可供检查，本次无需 Debian 克隆。
