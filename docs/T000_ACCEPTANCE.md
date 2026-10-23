# T000 初始化验收报告

状态：实施中，尚未宣称 T000 verified。最终证据将在 GitHub PR 和任务状态中补齐。

- 仓库：https://github.com/zl14229/pokemon-shiny-tracker
- 默认分支：dev；初始化起点：dc68988b7c76bf23ae3b68b588d15858f50b742e
- 工作分支：feature/T000-repository-baseline；保留旧预备分支
- dev 验收提交、PR、合并 SHA、CI：待实际完成后补齐

## 已核实配置

- 公开新仓库；默认 dev；只允许 Merge Commit；auto-merge 可用；delete_branch_on_merge=false。
- protect-dev（25725833）、protect-prod（25725884）：active，准确匹配 dev/prod，禁止删除和强推，要求 PR、讨论解决，approvals=0，allowed_merge_methods=[merge]，bypass=[]。
- 两套规则均已通过登录后的 GitHub 设置页补齐 required_status_checks，并以 GitHub REST 读取结果复核：baseline-checks、repository-hygiene，integration_id=15368（GitHub Actions），strict_required_status_checks_policy=true，无创建豁免。
- Actions 设置页已独立复核：允许仓库自身、GitHub 自有及 Marketplace 已验证创建者；默认令牌 contents/packages 只读；全部外部贡献者的 fork PR workflow 需批准；Actions 创建/批准 PR 选项关闭。当前工作流中的 Actions 均已固定完整 SHA；仓库全局强制固定 SHA 开关未开启。
- prod 提前存在但未发布，仍为初始提交。

## 验证范围

六份冻结原件 SHA-256；29 个 Task 编号/定义/依赖/最低验收与无环性；16 ADR 编号；状态证据；PR 分支规则；忽略规则与追踪文件检查；空白字符；实际 GitHub CI。检查命令和结果在完成后补齐。

## 自我审查

- 产品冻结内容不变；无业务代码、数据库、资料抓取或真实用户数据。
- 任务状态不以生成文档为完成；只有实际 PR/检查/merge 证据齐全才能 verified。
- CI 当前只覆盖仓库治理，Go/Vue 构建测试延后对应工程任务。
- Task 分支保留；不强推、不删除分支、不发布 prod。

## 未完成事项

PR #1（https://github.com/zl14229/pokemon-shiny-tracker/pull/1）已建立，初始提交 0ffc2eae07f605a4c14fa2411c301bffc88c1075 的 push 与 PR CI 均成功。PR CI：https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/39975588374。

初步自审发现“无合并证据不得 verified”的反例测试需要独立清空证据，才能在 T000 后续完成后继续有效；已修复。仍待最终提交 CI、合并与 dev CI，以及真实合并证据归档。完成前 T000 不标 verified。
