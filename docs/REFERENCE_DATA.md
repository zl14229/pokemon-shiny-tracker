# 基础资料与来源（T006）

## 当前已核实范围

来源为 [PokéAPI v2](https://pokeapi.co/docs/v2)。本次用程序的 Stage 客户端实际采集并校验，规范化小范围种子在 `internal/reference/seed.json`，包含必要结构化事实、原始 URL、响应 SHA-256 和采集时间，不含正文、图片、flavor text 或个人数据。

- 物种：Pikachu（25）、Raichu（26）、Eevee（133），编号通过源 national Pokédex 条目核验，不按数组位置推算。
- 基础形态：上述默认 Pokemon 实际关联的 3 个 form；核验 Species→Pokemon→Form 及反向链接，不将 form ID 当全国编号。
- 版本：red（1）、blue（2）、scarlet（40）、violet（41）；括号是外部 version ID，不是本地 Game ID。

**覆盖始终标为 partial**，并非全部图鉴/形态/游戏。基本存在不表示可闪光、可直接刷闪、有攻略或已验证概率；相关规则属于后续任务。

中文值标 `source_provided`，未做 52Poké/官方简繁交叉核验，保留源值、不自动转换；例如源 zh-hans 的版本名可能含繁体字。缺失为 null/unknown，同语言冲突为 null/conflict，不用物种名补造形态译名。en 缺失时回退源 slug。语言标签大小写兼容。

## 离线与身份

空参考表启动时从嵌入种子导入，**普通启动和查询不联网**。已有参考数据不自动覆盖或补写，不通过新种子改变个人历史。Hunt/ShinyRecord 不参与导入写操作。

SQL 版本 2 仅增加 reference_batches/external_references，版本 1 的个人身份与数据保留。本地 Species/Form/Game ID 自动生成，与外部 ID 分开；映射包含外键、来源+类型+外部 ID 唯一键和目标类型 CHECK，以及 URL、响应哈希、采集/结构核验时间、状态、批次和规范化事实。批次保留完整规范化快照，形态保留 Pokemon 链接/哈希/采集时间证据。

`validated` 指结构、ID、链接与本地事实校验通过，不能解释为官方译名或合法刷闪性。

## 采集、预览与应用

遵守 [Fair Use Policy](https://pokeapi.co/docs/v2#fairuse)：网络响应本地缓存，单客户端串行 GET、至少 1 秒起始间隔、明确 User-Agent，不跟随来源跳转或自动重试洪泛。单响应最多 2 MiB，缓存保留原始字节并校验 SHA-256。默认缓存为用户缓存目录下 `pokemon-shiny-tracker/pokeapi-v2`，拒绝 Git checkout 内路径与未知缓存文件覆盖。

从仓库根目录使用 Linux Go，暂存输出文件必须不存在：

```bash
bash -ic 'go run ./cmd/reference -mode stage -species 25,26,133 -versions 1,2,40,41 -output /absolute/staging/new-reference.json'
bash -ic 'go run ./cmd/reference -mode stage -offline -output /absolute/staging/new-offline-reference.json'
```

stage 不接收数据库连接，不打开个人库。`-cache` 指定仓库外绝对缓存目录；`-refresh` 只明确刷新已识别的缓存，不应用数据库；`-offline` 严禁网络请求。每个选择最多 50 个不重复正整数 ID，不采集其他未经授权站点。

审查暂存文件后，用仓库外数据库预览与明确应用：

```bash
bash -ic 'SHINY_DB_PATH=/absolute/private-directory/tracker.sqlite3 go run ./cmd/reference -mode preview -input /absolute/staging/new-reference.json'
bash -ic 'SHINY_DB_PATH=/absolute/private-directory/tracker.sqlite3 go run ./cmd/reference -mode apply -input /absolute/staging/new-reference.json'
```

preview 会按应用版本初始化/迁移结构，但参考插入全部在事务内验证后回滚，不提交参考/个人记录。report 的 added 是计划数，只有 applied=true 表示应用。apply 校验后原子提交；重复文件、仅证据时间/哈希刷新均不重建身份或重复批次。

M1 是新增/幂等导入：源事实变化、本地资料改写或无映射身份冲突，返回 before/after 差异并拒绝整批应用，不绕过冲突重建主键或覆盖个人记录。完整增量更新和资料回滚管理仍属 T024，当前不声称完成。

## 查询 API

| GET 路径 | 内容 |
|---|---|
| `/api/v1/reference/species` | 本地物种列表 |
| `/api/v1/reference/forms?species_id=本地ID` | 本地形态列表，支持所属物种过滤 |
| `/api/v1/reference/games` | 本地游戏列表 |
| `/api/v1/reference/{species,forms,games}/:id` | 本地 ID 详情，缺失为 404 |
| `/api/v1/reference/{species,forms,games}/:id/sources` | 来源映射与批次证据；无来源时为空数组 |
| `/api/v1/reference/coverage` | 实际计数、partial 与 shiny_availability=not_verified |

列表响应 `{data:[],meta:{total,limit,offset,coverage}}`，默认 limit=100，范围 1–500，offset 非负，按 ID 升序；非法参数为 JSON 400。全国编号不能直接当本地 API 主键，全程只读 SQLite。

## 验证

真实采集后用离线模式重建，规范化输出逐字节一致。CLI 在自建临时库 preview→apply→重复 apply，结果为未提交、10 新增、0 新增/10 不变。测试覆盖缓存字节/哈希、来源失败/错 ID、节流、名称缺失/冲突、链接、ID 分离、幂等/元数据刷新、可审查的本地冲突、原子回滚、源失败不损坏个人数据、v1 升级及离线查询。CI 不请求真实资料源，使用种子与本机 HTTP fixture，继续 Go/前端/浏览器回归。

## M4 / T016 全国快照

随程序附带 PokéAPI 官方仓库 v2 CSV 固定版本 `2bf639e5572812a67d2b94903fdcb9d095f5aeb2`：1,025物种（全国编号1–1025）、1,579形态；4个已存在游戏种子保持不变。本地内部ID不是全国编号或外部form ID；新增映射由catalog_entries管理，与现有external_references逐条比对。既有资料不改名、不改ID，个人表不写入。首次启动离线事务增补，冲突时回滚并继续使用既有资料，不能宣称此时已全量载入。已有同版本批次幂等跳过；事实变更不自动覆盖，待T024。

采用PokéAPI官方v2同源CSV快照减少逐实体网络请求；这不是1,025次v2 API响应采集。每个文件固定提交URL、SHA256、采集时间保留于批次快照；每条物种/形态含来源文件、稳定外部ID、数据版本和本地校验时间。名称分别来自pokemon_species_names/pokemon_form_names；形态关系经pokemon.csv回溯物种。原始缓存位于仓库外，可重建：

```sh
python3 scripts/stage_catalog.py --cache /absolute/cache --revision 2bf639e5572812a67d2b94903fdcb9d095f5aeb2 --offline --output /new/catalog.json
go run ./cmd/reference -catalog -mode preview
go run ./cmd/reference -catalog -mode apply
```

暂存输出仅供审查和构建打包；运行时CLI只应用程序内已审核的bundled快照，不自动应用任意输入或联网同步。旧版精简种子仍用于兼容性测试。

主形态口径v1：1,227 included、172 excluded、180 unknown。全国物种、默认形态、地区形态和明确列出的持久变体家族纳入；mega/来源标注的battle-only及特殊皮卡丘外观排除；其他变体待核实。该分类是基于来源标志及本地产品口径的规则，不宣称逐形态已获官方交叉验证，unknown不进入主形态完成率。分类亦不等于闪光可获取性。

语言回退：简体有源值显示但标source_provided/中文待核实；缺少简体不自动转繁体或机器翻译，回退英语再回退slug；来源conflict禁用中文回退。形态只接受完整pokemon_name翻译，不把物种中文名冒充形态译名。保留所有源语言。无跨站核验的条目cross_check_status=unknown，checked_at只是结构/策略检查日期，不是中文官方核对日期。

52Poké于2026-10-26核对[机器读取守则](https://wiki.52poke.com/wiki/神奇宝贝百科:机器读取守则)、[robots](https://wiki.52poke.com/robots.txt)和[常见问题](https://wiki.52poke.com/wiki/神奇宝贝百科:常见问题)。规则存在UA、串行与间隔、robots和CC BY-NC-SA 3.0约束；未据此推断无限制批量/图片使用权。本次未批量复制正文、未镜像图片；每条提供相关中文物种交叉核对链接，但**尚无条目完成52Poké内容交叉核验**。因此中文均不标verified。Serebii/Bulbapedia/官方逐游戏合法性核验待T026；目前全为未知，不能因物种/形态存在就计算合法闪光分母。图片未获复用核验，不打包第三方图集。

PokéAPI版权及再分发条款随[原许可](POKEAPI_LICENSE.md)保留；宝可梦及角色名商标属于Nintendo，无背书宣称。公开/商业发布仍需另审素材条件，本次仅本地开发。

新增API：`GET /api/v1/reference/catalog?kind=species|form&inclusion=included|excluded|unknown&limit=100&offset=0`，返回多语言names、display_name、name_status、分类原因、来源URL/版本/哈希/时间/待核对链接。`GET /api/v1/reference/catalog/coverage`返回实际本地计数、批次checksum和明确的部分核验覆盖提示。基础reference API保持兼容，新增全国数据需要分页读取。
