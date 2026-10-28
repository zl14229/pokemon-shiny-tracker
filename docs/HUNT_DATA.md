# 游戏刷闪资料（T026）

本地资料始终是**部分覆盖**。初始批次 `m5-sv-wild-2026-10-28-v1` 仅记录朱、紫各 12 个目标（共 24 个游戏与形态组合），含帕底亚乌波。其他游戏、形态和途径未知；未建立全游戏概率库。中文名沿用图鉴来源状态，并未因此成为官方核验译名。

## 核对范围和来源

2026-10-28 查阅、2026-10-29 00:00（Asia/Shanghai）完成结构化事实复核。阅读以下有限页面，手工选取少量可独立表达的结构化事实及链接；没有抓取循环、网页正文、图片、地图或表格镜像。批量自动读取与公开内容再分发许可未获确认，本实现不启用这些行为。引用页面不是授权背书，也不是游戏内实测。

- [Serebii 南第一区](https://www.serebii.net/pokearth/paldea/southprovinceareaone.shtml)：出现目标。
- [Bulbapedia 南第一区](https://bulbapedia.bulbagarden.net/wiki/South_Province_(Area_One))：逐项交叉核对朱、紫普通野外出现及帕底亚形态。
- [Serebii 朱／紫闪光机制](https://www.serebii.net/scarletviolet/shinypokemon.shtml) 与 [Bulbapedia 第九世代机制](https://bulbapedia.bulbagarden.net/wiki/Shiny_Pok%C3%A9mon#Generation_IX)：普通野外可闪和无加成基础 1/4096。没有导入其他加成表。
- [52Poké 机器读取守则](https://wiki.52poke.com/wiki/神奇宝贝百科:机器读取守则)：已再次查阅，要求可识别 UA、遵守 robots、顺序及间隔等；本轮不自动读取其资料内容，不宣称中文已交叉核验。
- 稳定物种、形态和游戏版本映射来自已缓存的 PokéAPI v2 资料，原始元数据仍保留。

所选目标：Pichu、Hoppip、Bonsly、Happiny、Fletchling、Bounsweet、Skwovet、Lechonk、Tarountula、Pawmi、Shroodle、Wooper（Paldean）。不把目标在地区出现推论为任何其他版本可刷。

版本范围：朱／紫本篇普通野外，查阅日 2026-10-28；源页未固定软件构建号，**不声称验证任意补丁、活动或 DLC 路径**。保留逐条来源、scope、核对时间、来源版本、批次和校验和。两个版本逐条保存，Planner 必须先去重再抽取。

所有攻略为 partial：完整时间、天气和最佳路线尚不齐全。遭遇率全部 null；百科的出现权重及群体率不能充当目标的最终遭遇概率。闪光率仅适用于无护符、闪耀力、大量出现或活动加成的普通野外单次判定，不适用于团体战、孵蛋或其他版本。未导入负面锁闪／仅传送事实；这些状态由结构和合成测试覆盖，不伪造真实断言。

## 本地读取与受控入库

`GET /api/v1/reference/hunt-data?game_id=ID&limit=100&offset=0` 返回映射后的内部 ID、原始外部 ID、可获取性、direct_hunt、攻略完整性、方法、地点、条件、独立遭遇率／闪光率及所有证据。旧 `/reference/availability` 仍供图鉴使用。只有 verified + confirmed_available + direct_hunt=yes 能成为 Planner 候选。未有详细方法仍可候选；概率缺失不代表锁闪。

schema 7 增加 hunt_data_batches、hunt_facts、pokemon_hunt_methods。启动仅加载随程序已审核的本地快照；重复启动幂等。冲突时保留现有资料并警告，个人 CRUD 仍可使用。源表不可覆盖个人记录；Hunt、ShinyRecord、历史、名称与稳定 ID 均不改写。

维护工具显式指定数据库（仓库外，先做备份），不读取网络：

```bash
bin/shiny-data hunt-preview --database /absolute/tracker.sqlite3
bin/shiny-data hunt-apply --database /absolute/tracker.sqlite3 --reviewed-sha PREVIEW_CHECKSUM
bin/shiny-data hunt-rollback --database /absolute/tracker.sqlite3 --version BATCH_VERSION --reviewed-sha PREVIEW_CHECKSUM
```

自备已核对 JSON 可加 `--snapshot /absolute/staged.json`。预览走相同校验和事务路径但回滚；应用要求审核的校验和。新批次仅添加尚无资料的组合；已有组合冲突则全部取消，不静默覆盖。回滚只撤销该批次新增的资料并保留不可变审核快照；已撤销批次不因重启自动重装。修改或同步已有事实、来源适配器及完整增量恢复流程属于 T024。

JSON 结构校验只能验证字段与映射，不能代替资料真实性审核。维护者必须核对版本、形态、来源、条件和访问方式后才应用，不能仅凭校验通过将事实标为可信。

备份／导出包含全部三张新增表。旧 schema 备份请用对应旧版恢复至新目录，再用新版升级；不覆盖真实数据库。
