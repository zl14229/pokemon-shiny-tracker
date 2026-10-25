# JSON导出、ZIP备份与隔离恢复（T014）

数据管理→导出与备份（`#/data/backup`）提供两种本地下载，不向外部服务上传。文件含个人信息且未加密，请保存在私有位置，不提交Git。只有完整生成和校验后才交给浏览器保存；失败不会提供部分备份或修改源记录。

## 两种格式

- **JSON导出**：SQLite一致性快照中11张应用表的所有列，含Hunt/个体/历史、来源映射、导入批次/原值与文件元数据、软删除状态，保留ID/空值/日期精度。仅数据和元数据，不含附件文件；不提供JSON恢复。本格式不是Excel，不执行单元格内容。
- **ZIP备份**：`database.sqlite3`、`files/<随机键>` 和 `manifest.json`（格式版本、schema、时间、每项大小/SHA256）。包含数据库引用的全部本地文件（含软删除关联）；不包含原始Excel、无元数据孤立文件、外部网页/视频内容及运行配置。链接元数据在数据库中保留，不抓取外部内容。

当前只接受本程序schema 4与本备份格式；未知、旧/未来schema、结构缺失、完整性或外键错误均拒绝，不自动迁移或修复用户库。SQLite上限512MiB，备份解压总量2GiB、最多100000个文件、单文件沿用10MiB；边界限制不代表已进行最大规模性能验收。HTTP生成/下载有2分钟期限，大型操作可用本地CLI；超时应核对结果并选择新的输出路径，不覆盖旧备份。

## 一致性与文件路径

使用已锁定的go-sqlite3在线Backup API读取SQLite快照，正确处理WAL；不直接复制正在运行的主数据库文件，也不对源库调用迁移/建表/资料同步。随后所有导出/归档读取该快照。文件在元数据提交前独占写入/fsync，正常删除保留不可变文件，因此并发新增/软删除时快照中的文件仍可读取。逐个核对引用、名称/类型/大小/SHA256；缺失/损坏即整份取消，不静默省略。生成ZIP后再完整verify，成功才独占发布输出。

上传提交失败后的清理也先重新查询引用；数据库不可用或已存在引用时保留文件，避免不明提交后误删。崩溃孤立文件不自动清理，亦不当作已关联附件备份。不要手动删改私有数据库或`.files`目录。

新备份以私有临时文件0600生成，再通过不替换的链接操作发布到新输出路径；已有文件、目录和符号链接一律不覆盖。恢复在仓库外自建0700 stage中校验ZIP条目、重复/超限/路径、manifest、文件哈希与数据库/附件引用一致性，校验全部通过后才发布。Linux使用 `RENAME_NOREPLACE` 原子发布到**不存在的新目录**，即便竞争期间目标出现也拒绝；无不安全回退。恢复成功的目录含 `database.sqlite3` 和（如有本地文件）`database.sqlite3.files/`。

当前安全恢复仅实现Debian/Linux；需要文件系统支持不替换原子重命名。WSL Linux临时文件系统已验证，Windows原生和Windows挂载目录未验收。程序不会恢复到现有目录，不切换正在运行的数据库，也没有Web覆盖恢复接口。

## 可重复执行的本地方法

从项目根目录使用Linux工具链构建；以下路径均为占位，必须换成自己的仓库外绝对路径。备份/导出输出文件必须不存在，恢复目标目录必须不存在。CLI不默认读取个人数据库，缺少明确参数会失败。

```bash
bash -ic 'make build'
bash -ic 'bin/shiny-data backup --database /absolute/private/tracker.sqlite3 --output /absolute/private/new-backup.zip'
bash -ic 'bin/shiny-data export --database /absolute/private/tracker.sqlite3 --output /absolute/private/new-data.json'
bash -ic 'bin/shiny-data verify --archive /absolute/private/new-backup.zip'
bash -ic 'bin/shiny-data restore --archive /absolute/private/new-backup.zip --destination /absolute/private/new-restore-check'
```

`verify`在自己新建的临时位置校验，不改归档；`restore`也重新校验且仅发布新目录。再次演练请指定另一个新目录，不删除/覆盖原库。恢复后可再次对恢复库export，核对记录/历史/导入批次与文件；需要浏览器检查时另开隔离进程并使用恢复库路径与独立端口，保持当前个人应用关闭或明确区分。本次所有恢复演练均为匿名测试库/自建临时目录，未对个人数据库运行这些操作。

## API与测试

- `GET /api/v1/data/export`：已完成的一致性JSON文件下载。
- `GET /api/v1/data/backup`：已完成且verify通过的ZIP下载。
- 没有restore/覆盖Web接口；错误沿用统一JSON，前端不渲染原始路径或错误正文。仅loopback/same-origin，拒绝外部Origin/跨站请求，无CORS。

自动测试真实SQLite在线快照、重复隔离恢复、ID/空值/历史/导入溯源、软删除文件、JSON全字段、源库只读、既有输出/目标不变、损坏/缺失/遍历拒绝、并发创建/上传/软删除及取消；HTTP与真实浏览器下载后执行CLI verify/restore/export并核对附件字节。不存在“只生成ZIP就算恢复通过”。文件哈希用于完整性，不是加密或来源身份认证；只恢复可信的个人备份。
