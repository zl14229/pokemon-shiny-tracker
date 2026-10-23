# Pokémon Shiny Tracker — ADR Register V1.1

状态定义：`Accepted` = 产品/技术方向已经讨论确认；`Accepted (default)` = 需求冻结采用的可变实现默认值；`Verify at implementation` = 有意延迟到实际资料/仓库检查，不是未确定产品方向。

| ADR | 决策 | 状态 | 理由/后果 |
|---|---|---|---|
| ADR-001 | Hunt 与 ShinyRecord 双实体，0..N 成果，ShinyRecord 可没有 Hunt | Accepted | 同时保留成功、失败与历史 Excel 数据 |
| ADR-002 | 全国图鉴物种 + 形态双层统计 | Accepted | 重复个体不虚增完成度，形态有独立收藏价值 |
| ADR-003 | MVP 全系列基础记录，非主系列专属规则后续完善 | Accepted | 控制初期复杂度且不封闭其他游戏 |
| ADR-004 | 历史获得 + 当前持有双口径 | Accepted（原则确认） | 兼顾成就与 Living Dex；历史不自动推断 |
| ADR-005 | 可获取性决定完成度分母，未知不计入、可记录 | Accepted（原则确认） | 不将锁闪/未知错误视为缺失目标 |
| ADR-006 | Hunt Planner 双随机池：全系列未获得 / 指定游戏未获得，默认全系列 | Accepted | 补图鉴和版本专刷兼顾 |
| ADR-007 | Random Hunt 和 Bingo 正式产品功能 | Accepted | 将规划与执行、记录和收藏形成闭环 |
| ADR-008 | 本地单机优先，Vue/Go/SQLite，局域网及互联网后续 | Accepted | MVP 维护和部署成本低 |
| ADR-009 | 随机从去重目标均匀采样，再选符合条件的游戏 | Accepted (default) | 避免跨游戏重复提高抽中概率 |
| ADR-010 | Bingo 默认 3×3/5×5、无免费格、用挑战后自刷成果完成 | Accepted (default) | 可自动验证；防止旧收藏直接完成卡片 |
| ADR-011 | 轻量历史事件保留进化、形态、持有变化；不引入复杂事件溯源 | Accepted (default) | 支持历史/当前图鉴并保持架构简单 |
| ADR-012 | Excel 导入先预览、标记疑似重复，不强制重建 Hunt | Accepted | 保护旧数据、兼容不完整资料 |
| ADR-013 | 完成每项 Task 必须留测试、自审、日志和可追踪状态 | Accepted | 支持 AI 跨上下文继续推进 |
| ADR-014 | 非重大实现 AI 自决；架构变更、毁损数据、外部权限需停下来 | Accepted | 减少用户两端传话并保留关键控制权 |
| ADR-015 | GitHub；新建干净仓库；dev 集成、prod 稳定发布；Task 分支→PR→dev，合并后保留历史分支；受控发布与 CI | Accepted | 用户已删除旧远程，不沿用旧实现提交历史；GitHub 使用 Pull Request 而非 MR |
| ADR-016 | PokéAPI 作为基础结构化来源；52Poké 中文核对；Serebii 游戏/锁闪/概率核对；Bulbapedia、Pokémon Database 和官方交叉验证；本地缓存、来源溯源和合规采集 | Accepted (strategy) / Verify at implementation (coverage and permission) | 支持全系列图鉴与 Hunt Planner，同时避免将尚未核实的来源视为可无限抓取数据集 |

## 执行期需要验证但不重新争议的事项
- 用户已删除旧 GitHub 远程仓库，计划重建；旧本地项目须备份但不作为新项目的已实现 Task；T000 在新 GitHub 远程建立基线。
- Excel 原始字段、值范围、丢失信息与去重冲突。
- PokéAPI v2 具体资源覆盖、语言与 form ID 映射；52Poké 的 API/读取规则和再使用许可；Serebii 的自动获取许可；分游戏合法可刷性、方法/概率来源、覆盖率及核对日期。
- 图片和视频的格式、体积、存储路径及备份策略。

## ADR-016：宝可梦基础数据来源与治理说明
**已决定**：源站角色分工、本地缓存/来源追踪/分级核验的架构方向，及未知状态不能进入默认合法候选池。
**没有宣称已核实**：52Poké 大规模接口可用性、Serebii 自动抓取许可、各游戏完整遭遇信息、条件闪光概率及图片转用授权。
**变更成本**：新增来源优先通过来源适配层扩展；如需要换主来源或大幅改变映射主键，再补充 ADR 和迁移测试。

参考：
- PokéAPI v2 文档：https://pokeapi.co/docs/v2
- 52Poké 机器读取守则：https://wiki.52poke.com/wiki/神奇宝贝百科:机器读取守则
- Serebii：https://www.serebii.net/

## 变更控制
改动 Accepted 的用户业务口径、一级产品范围、核心实体关系或技术栈：记录新 ADR，调整 Task、说明迁移与回归成本后才实施。普通实现修复和内部重构由 AI 自行处理并写开发日志。
