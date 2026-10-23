# T000 初始化验收报告

状态：实施中，尚未宣称 T000 verified。最终证据将在 GitHub PR 和任务状态中补齐。

- 仓库：https://github.com/zl14229/pokemon-shiny-tracker
- 默认分支：dev；初始化起点：dc68988b7c76bf23ae3b68b588d15858f50b742e
- 工作分支：feature/T000-repository-baseline；保留旧预备分支
- dev 验收提交、PR、合并 SHA、CI：待实际完成后补齐

## 已核实配置

- 公开新仓库；默认 dev；只允许 Merge Commit；auto-merge 可用；delete_branch_on_merge=false。
- protect-dev（25725833）、protect-prod（25725884）：active，准确匹配 dev/prod，禁止删除和强推，要求 PR、讨论解决，approvals=0，allowed_merge_methods=[merge]，bypass=[]。
- 初检发现 required_status_checks 尚未配置。完成情况须复核，不能把本地 CI 文件当成服务端阻止失败合并的证明。
- 当前 GitHub 连接可读/写内容及 PR，但不提供管理规则写入工具；Actions 权限读取接口也被连接器拒绝。此前截图显示受限 Actions、只读 token、全部外部 fork 贡献者需批准；本次暂不能独立重新读取该后台配置。
- prod 提前存在但未发布，仍为初始提交。

## 验证范围

六份冻结原件 SHA-256；29 个 Task 编号/定义/依赖/最低验收与无环性；16 ADR 编号；状态证据；PR 分支规则；忽略规则与追踪文件检查；空白字符；实际 GitHub CI。检查命令和结果在完成后补齐。

## 自我审查

- 产品冻结内容不变；无业务代码、数据库、资料抓取或真实用户数据。
- 任务状态不以生成文档为完成；只有实际 PR/检查/merge 证据齐全才能 verified。
- CI 当前只覆盖仓库治理，Go/Vue 构建测试延后对应工程任务。
- Task 分支保留；不强推、不删除分支、不发布 prod。

## 未完成事项

推送及 PR CI；服务端必需检查门槛；合并与合并后 dev CI；真实提交证据及最终状态。上述完成前 T000 不标 verified。
