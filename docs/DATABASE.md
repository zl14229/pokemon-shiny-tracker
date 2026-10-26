# SQLite 与模型（M1）

## 存储与启动

SQLite 使用 GORM 官方 driver v1.6.0、go-sqlite3 v1.14.52（module 元数据核实后精确锁定）。需要 CGO=1 和 C 编译器，本机及 Ubuntu CI 使用 GCC。相关行为参考 [GORM 连接文档](https://gorm.io/docs/connecting_to_the_database.html)、[迁移文档](https://gorm.io/docs/migration.html)及 [go-sqlite3 参数说明](https://github.com/mattn/go-sqlite3)。

默认位置为 `$XDG_DATA_HOME/pokemon-shiny-tracker/tracker.sqlite3`；未设置 XDG_DATA_HOME 时使用 `$HOME/.local/share/pokemon-shiny-tracker/tracker.sqlite3`。可通过进程环境 `SHINY_DB_PATH` 指定仓库外的干净绝对路径。配置读取不创建文件；实际启动时初始化数据库，新目录权限 0700、新文件 0600。拒绝相对路径、文件符号链接和任何 Git checkout 内的数据库路径；不会把个人数据写入源码仓库。

```bash
bash -ic 'SHINY_DB_PATH=/absolute/private-directory/tracker.sqlite3 make run'
```

WAL、foreign_keys=on、busy_timeout=5000ms、同步 NORMAL、立即写事务和单连接用于本地单机；DSN 参数对每次连接生效，避免只对一条连接执行 PRAGMA。程序正常退出关闭数据库。健康接口仍只表示 HTTP 能响应，不承诺资料/业务健康。

## 版本化迁移

`PRAGMA user_version` 标识结构版本。版本 1 建立基础六张实体表，事务内建表、约束、索引和版本一起提交；失败回滚。仅空库首次使用 GORM AutoMigrate。之后启动不按当前 struct 隐式重建用户表，后续变更需增加明确的顺序迁移。

已有文件先以只读连接检查版本和表；未知非空库、未来版本和缺失基本表的已知版本拒绝打开，不能靠删除表或覆盖数据库“修复”。未来迁移须保留旧数据并经过恢复验证。当前提供失败回滚，不提供对真实数据执行破坏性 downgrade/恢复的命令。

## 实体与不变量

| 实体 | 角色与边界 |
|---|---|
| Game | 本地游戏身份、名称、类型、收藏体系；HOME 可用 storage 类型保存，不是普通刷闪游戏 |
| Species | 本地不可变 ID、唯一全国编号，与名称分离 |
| Form | 本地身份与所属物种，ID 不等于全国编号；复合唯一索引支持物种/形态一致性外键 |
| Hunt | 一个过程/目标；可没有成果，可关联多个成果；计数为空或非负，存在计数时必须有非空单位；耗时以可空非负秒数保存 |
| ShinyRecord | 一个永久个体；可无 Hunt，原始获得与当前位置分开；未知原始物种/形态/游戏不从当前值反推 |
| ShinyHistory | 关联个体的轻量历史；注册、进化、形态、位置、持有变化/普通更新/删除均可追溯，前后快照为有效 JSON |

日期采用可空 ISO 年/月/日字符串，精度由 YYYY、YYYY-MM、YYYY-MM-DD 明确保留，绝不以记录创建时间伪造获得日期。CreatedAt/UpdatedAt 只是系统审计时间，统一 UTC。API 层会在对应 CRUD Task 校验真实日历日期及业务状态转换。

Hunt 与个体使用 soft delete，历史实体限制物理删除关联身份；业务删除的关联约束由 T007/T008 实现。完成 Hunt 至少一项成果的跨表规则由 T007/T008 的事务业务层验证，不能把基础状态枚举 CHECK 当作已经实现状态转换。

SQLite CHECK 约束状态枚举、计数/耗时和等级范围；外键保护所有引用。形态与其物种使用复合外键，避免仅检查形态存在却错误挂在其他物种。原始形态非空时必须有原始物种。基础名称区分 unknown/source_provided/verified/conflict；外部来源映射与覆盖说明在 T006 建立，不据此推断合法可刷性。

## 测试与数据边界

Go 数据库/关系测试使用 `t.TempDir()`。后端 smoke、前端真实集成与 Playwright 分别创建临时目录，向进程传入 SHINY_DB_PATH；只清理自己创建的目录。测试不打开用户默认数据库。

验证从零初始化、重复迁移/重开、外键/形态错配、数值和状态 CHECK、0..N 成果、无 Hunt 个体、历史引用、失败回滚、未知/未来库不修改。根 Make/CI 继续执行普通与竞态测试、格式/依赖/vet/build、启动和前端回归。

## T006 来源表升级

版本 2 只创建 reference_batches/external_references，迁移不重建已有核心表。v1→v2 保留个人身份和记录有测试；新库由程序离线初始化小范围种子。来源/查询及 partial 覆盖详见 REFERENCE_DATA.md。

## T012 导入溯源升级

版本3仅新增import_batches/import_rows，保留v1/v2身份与数据。预览只在有界短期内存中；确认后批次/选中行/原始单元格与ShinyRecord/注册历史在同一事务写入，批次令牌唯一保证重试幂等，外键限制物理删除。来源原值属个人数据，仅存私有SQLite，按T014随数据备份；不保存上传的Excel原件。详见 [导入说明](IMPORT.md)。

## T013 附件元数据升级

版本4仅新增attachments，所属个体外键限制物理删除、文件键可空唯一（多个外部链接可并存）、文件/链接字段一致性CHECK与软删除索引；上传元数据失败不触碰原文件。文件目录不在SQLite中存绝对路径，按该数据库路径+.files隔离，具体限制/保留策略见 [ATTACHMENTS.md](ATTACHMENTS.md)。

## T014 数据导出与备份

不增加schema或改写源库；采用只读在线SQLite备份快照，保留稳定身份、NULL、软删除、历史/导入溯源及引用文件。仅schema4与当前归档格式，恢复先全面校验再原子发布新目录，绝不覆盖已有个人记录。方法与限制见 [BACKUP.md](BACKUP.md)。

T016 schema5：新增catalog_entries，只关联稳定物种/形态身份，存多语言、形态纳入状态、同源CSV版本与交叉核对状态。迁移不改个人表/原始和当前物种字段；个人备份/JSON导出包含新表，ZIP恢复要求本程序当前schema（旧schema4备份需旧版本先恢复到新目录再升级，不原地覆盖）。
