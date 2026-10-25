# HTTP API 契约

后端默认 `http://127.0.0.1:8080`，API 前缀 `/api/v1`。健康仍仅表示进程可响应；T006 本地参考接口见 [来源与查询说明](REFERENCE_DATA.md)，查询不访问外部源。

| 请求 | HTTP | JSON |
|---|---|---|
| `GET /api/v1/health` | 200 | `{"data":{"status":"ok"}}` |
| 未注册路径 | 404 | `{"error":{"code":"not_found","message":"请求的接口不存在"}}` |
| 已注册路径的不支持方法 | 405，`Allow: GET`（health） | `{"error":{"code":"method_not_allowed","message":"请求方法不受支持"}}` |
| 尚未发送响应时的内部错误或 panic | 500 | `{"error":{"code":"internal_error","message":"服务暂时不可用，请稍后重试"}}` |

响应内容类型为 `application/json; charset=utf-8`。客户端以 HTTP 状态和 `error.code` 判别错误，`message` 用于提示。内部错误、panic 原值和堆栈不写入响应。服务端只记录固定的失败提示，不记录原始错误或请求输入。

健康结果仅表示 HTTP 进程可响应，不代表数据库、资料覆盖率或业务功能已就绪。`/health`、`/api/v2/health` 和 `/api/v1/health/` 不作为别名。HEAD 不隐式映射为 GET；HTTP 协议会省略 HEAD 响应体。

错误中间件处理 `c.Error` 和 panic。若 handler 已提交响应，则终止后续处理但不追加第二段 JSON，也无法撤销已发出的状态/内容；后续 handler 应先验证与完成可能失败的操作，再写响应。测试专用失败路由只在测试中注册，实际程序不提供故障触发入口。

目前不启用 CORS 或认证，服务限制在 loopback；健康不检查数据库。T004 前端使用 `/api/v1` 同源代理调用健康端点，健康 DTO 在运行时校验；非 JSON、格式不符、非 2xx、网络/超时均不得被视为健康。完整客户端行为见前端开发文档。

## Hunt（T007）

| 请求 | 成功 | 说明 |
|---|---|---|
| `POST /api/v1/hunts` | 201 `{data: Hunt}` | 新建；默认状态 `in_progress` |
| `GET /api/v1/hunts` | 200 `{data: Hunt[], meta: {total, page, page_size}}` | 查询规则见 T009 |
| `GET /api/v1/hunts/:id` | 200 `{data: Hunt}` | 不存在或已软删除为 404 |
| `PUT /api/v1/hunts/:id` | 200 `{data: Hunt}` | 完整替换可编辑字段，要求 species_id 与 status |
| `DELETE /api/v1/hunts/:id` | 204 无响应体 | 软删除；有关联个体（含软删除历史）返回 409 |

写入使用 `Content-Type: application/json`（否则 415），请求上限 64 KiB（413），未知字段、非法 JSON/ID/类型、超限或不合法字段为 400。DTO 不接受 id/created_at/deleted_at，内部错误统一脱敏 500；状态/关联冲突为 409 `conflict`。创建示例：

```json
{"species_id":1,"form_id":null,"game_id":null,"method":"孵蛋","encounter_count":0,"count_unit":"eggs","duration_seconds":null,"status":"in_progress"}
```

ID 来自本地参考接口，不是 PokéAPI ID。物种必须存在，形态须属于目标物种，游戏可空；`storage` 类游戏不能作为普通 Hunt 游戏。基本资料存在并不证明该游戏可合法刷闪，未实现的可获取性不伪造验证。

可选字段为 form_id、game_id、method、start_date、end_date、encounter_count、count_unit、duration_seconds、notes；省略或 null 保留未知，PUT 省略的可选字段被清空。文本去除首尾空白，空文本归 null；method 最长 128 字符、单位 32、备注 5000。次数/秒数为非负 int64，已知次数（包括 0）须提供非空单位，单位已知而次数未知允许保存。不计算跨单位效率。

日期只接受有效 `YYYY` / `YYYY-MM` / `YYYY-MM-DD`，原样保留精度；仅在开始最早日晚于结束最晚日时拒绝倒置，不擅自填补日期。状态为 `in_progress` / `completed` / `abandoned`，可重新打开。完成须至少一个未删除的关联成果；放弃不删除成果，进行中可有多个成果，非目标意外闪光不被排除。个体登记与关联 API 见下方 T008。所有状态与删除保护均在 SQLite 事务内检查。

## 闪光个体与历史（T008）

| 请求 | 成功 | 说明 |
|---|---|---|
| `POST /api/v1/shiny-records` | 201 `{data: ShinyRecord}` | 新建永久身份；只要求 current_species_id |
| `GET /api/v1/shiny-records` | 200 `{data: ShinyRecord[], meta: {total, page, page_size}}` | 查询规则见 T009 |
| `GET /api/v1/shiny-records/:id` | 200 `{data: ShinyRecord}` | 软删除后为404 |
| `PUT /api/v1/shiny-records/:id` | 200 `{data: ShinyRecord}` | 完整替换，要求 current_species_id、source、holding_status |
| `DELETE /api/v1/shiny-records/:id` | 204 | 软删除，永久身份与历史保留 |
| `GET /api/v1/shiny-records/:id/history` | 200 `{data: ShinyHistory[], meta: {record_deleted,total,limit,offset}}` | 软删除后仍可查询，ID升序；limit默认100/最大500，offset非负 |

同 Hunt 的 JSON/大小/错误契约。可选字段：hunt_id、original_species_id、original_form_id、current_form_id、original_game_id、current_game_id、current_location、obtained_date、nickname、gender、nature、ability、ball、level、notes。所有引用均为本地ID；可选字段省略/null均不猜测事实。**PUT 是完整替换，调用者须提交希望保留的原始资料及其他可选字段；省略会清空，历史仍保留改前快照。**

```json
{"current_species_id":1,"hunt_id":null,"original_species_id":null,"original_game_id":null,"current_location":"HOME Box 1","obtained_date":"2018","source":"trade","holding_status":"held"}
```

source 为 self_hunted / trade / event / other / unknown，holding_status 为 held / traded / released / unknown；创建省略二者默认 unknown，不默认仍持有。关联 Hunt 不推断 source、获得日期/原始游戏/物种；不同目标的意外闪光允许关联。交换/活动不因此变成亲自刷闪，后续统计须按来源区分。本轮未实现自刷统计。

原始形态必须有原始物种，当前形态必须属于当前物种；保存位置类游戏可作为当前位置/特定来源。已知不同收藏体系（例如 sleep 与 pokemon）不能登记为同一个体的转移；未知体系不推断兼容或不兼容。未知原始游戏、日期允许 null。level 1–100；nickname128字、gender32字、nature64字、ability/ball128字、current_location256字、notes5000字，文本归一规则同 Hunt。obtained_date 和 event_date 支持三种日期精度，不用系统时间补获得日期。

当前物种变化必须声明 `change_type: "evolution"` 或 `"correction"`；evolution 表示用户报告的进化，并不假称已验证完整进化图或游戏条件；纠错记录为 updated。当前形态、位置/游戏、持有状态变化分别保留 form_change、location_change、holding_change；其他编辑为 updated。创建有 registered，删除有 deleted。可选 event_date 与 history_note（1000字）为事件证据，未提供时为空；created_at/updated_at只是系统审计时间。同次更新多类别历史共享完整 before_json/after_json 快照，不臆造中间状态或先后时间；相同内容重提不新增历史或个体。历史只读，无修改/删除端点，完整转移事件管理仍归 T023。

历史、个体和状态保护均在同一事务中完成，历史写入失败会回滚创建/更新/删除。删除或移走已完成 Hunt 的最后有效成果返回409，须先重新打开/放弃该 Hunt；已有多个成果时可逐项调整。交换出去/放生应更新持有状态而非删除，以保留历史获得事实；软删除用于撤销错误登记，不计入有效成果或正常列表。

## 查询、搜索、筛选和分页（T009）

适用于 `GET /api/v1/hunts` 与 `GET /api/v1/shiny-records`；参考资料与历史端点继续使用各自的 limit/offset。

- `page` 从1开始，默认1；`page_size` 默认50、范围1–100；偏移 `(page-1)*page_size` 不得超过2147483647。越过末页返回空数组，total仍是匹配总数。未知、重复、空值、非法编码/UTF-8或不合法参数返回400，不静默忽略；旧临时列表的 limit 参数已由 page_size 替代。
- `q` 最多200字符，匹配目标/原始/当前物种、形态、游戏的 slug/英文/源中文名，以及 Hunt 方法/备注/单位或个体昵称/备注/当前位置。搜索为字面子串，`%`、`_`、反斜线不会被当成用户通配符；ASCII匹配使用 SQLite LIKE 的默认大小写行为，不承诺中文分词或复杂 Unicode 大小写折叠。
- 所有筛选和 q 以 AND 组合；枚举与文本精确筛选区分大小写，首尾空白被去除。ID必须为正的 int64 范围本地ID，不存在的筛选ID正常返回空结果。
- `sort` 默认 created_at；共同支持 id/created_at/updated_at，Hunt另支持 start_date/end_date/encounter_count/duration_seconds，个体另支持 obtained_date/level。`order` 为 asc/desc，默认desc。空值始终排后，同值以相同方向的ID稳定排序；日期按已记录的ISO文本排序，部分精度不代表精确先后。
- 单次请求的总数与当前页在同一 SQLite 事务快照中读取。数据不变时跨页排序稳定；多次请求之间如有新增/编辑/删除，页边界可以移动，未实现跨请求快照或游标分页。

| Hunt 参数 | 语义 |
|---|---|
| species_id / form_id / game_id | 目标引用精确匹配 |
| status / method / count_unit | 状态、方法、单位精确匹配 |
| has_results=true/false | 至少一个有效关联个体 / 无有效个体；不计软删除个体 |
| min_count / max_count | 非负整数边界；须同时指定 count_unit，次数排序也须指定单位，避免跨单位比较 |
| min_duration / max_duration | 非负标准秒数边界 |
| start_from / start_to / end_from / end_to | 对相应日期的闭区间重叠筛选 |

| 个体参数 | 语义 |
|---|---|
| original_species_id / original_form_id / original_game_id | 原始获得引用精确匹配 |
| current_species_id / current_form_id / current_game_id | 当前引用精确匹配，与原始字段分开 |
| hunt_id / has_hunt=true/false | 特定 Hunt / 是否关联 Hunt |
| source / holding_status / current_location | 来源、持有、当前位置精确匹配 |
| obtained_from / obtained_to | 获得日期闭区间重叠筛选 |

日期界限支持 YYYY、YYYY-MM、YYYY-MM-DD，按各自可能日期区间重叠判断；未知日期不匹配日期范围，未知次数不匹配数值范围。不把模糊日期补成真实日期，也不声称交集内实际发生：例如日期“2024”的记录可匹配2024年2月，但仍返回“2024”，供调用者提示精度。确定倒置的日期/数值边界为400。

```text
/api/v1/hunts?status=in_progress&game_id=1&count_unit=eggs&min_count=0&sort=encounter_count&order=asc&page=1&page_size=20
/api/v1/shiny-records?original_species_id=1&current_species_id=2&holding_status=held&has_hunt=false&sort=obtained_date
```

## T012 Excel 导入

新增只读文件检查、字段/值映射预览、逐行错误与疑似重复、显式选行确认和批次查询。完整契约、大小边界、确认幂等与隐私见 [IMPORT.md](IMPORT.md)。接口前缀仍为 `/api/v1`，既有 Hunt/ShinyRecord CRUD 契约不变。

## T013 附件与链接

个体附件列表/上传、外部链接登记、完整性校验后的内容访问与软删除契约见 [ATTACHMENTS.md](ATTACHMENTS.md)。文件与元数据私有，链接不由服务端抓取。既有个体身份、业务与历史不变。
