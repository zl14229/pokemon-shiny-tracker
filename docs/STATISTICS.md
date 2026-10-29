# M6 统计口径与接口

统计只读取本地SQLite，不联网、不改个人记录、历史或引用资料；复用T019快照加载、T009白名单／绑定值／部分日期区间筛选、T017图鉴口径。统计新接口不建立第二套个人实体或会话层，无schema变更。

## Dashboard

`GET /api/v1/dashboard?collection_system=pokemon`：永久个体数（历史曾获得、确认持有、持有未知、自刷、来源未知、原始游戏未知）、Hunt三状态与缺失字段覆盖，以及物种／持久形态×历史／当前四组M4引擎的summary和coverage。图鉴分母仅已核实本地合法子集；零分母百分比null，不代表全图鉴0%。Sleep等体系独立。未知游戏沿用M4全范围体系回退，不填造原始获得游戏。

## 统一统计 API

`GET /api/v1/statistics` 返回 `data`：filters、individuals、hunts、by_game、by_method、by_source、trend、effort、coverage、rule。分组数组有稳定键／排序；同一事务快照内完成。

|参数|规则|
|---|---|
|collection_system|默认pokemon；独立体系不混合|
|game_id|精确版本；Hunt目标游戏、成果原始获得游戏；当前位置不代替|
|method|精确匹配关联Hunt的既有方法文本，不自行归并相似名称|
|source|self_hunted/trade/event/other/unknown；只筛成果，Hunt总数仍是任务队列|
|status|in_progress/completed/abandoned，筛Hunt及其关联成果|
|from/to|精确YYYY-MM-DD闭边界；分别筛Hunt.start_date、ShinyRecord.obtained_date；不按登记时间补齐|
|count_unit|保留原样，精确匹配既有单位；不同名称不擅自换算|
|min_count/max_count|非负整数，必须提供count_unit；未知值不匹配|
|min_duration/max_duration|非负标准秒整数，未知值不匹配|

所有字段单值，重复／空值／未知参数／倒置边界为400；game_id必须存在且与体系相符。方法、状态或数值筛选时，成果必须关联一个符合当前Hunt筛选的任务；无关联历史个体因此不会匹配。日期单独使用时，两个实体各自日期筛选；与关联筛选组合时同时满足。沿用T009区间重叠语义，不把模糊日期填成精确日。

## 图表和未知值

成果按永久ID计数，多个同种同日个体分别保留；进化／转移历史不JOIN倍增。软删除排除，已放弃Hunt不抹去此前成果。

七张ECharts图：月度任务开始与成果、原始获得游戏分布、来源分布、关联Hunt方法分布、次数、已记录耗时、自刷成果平均耗时。SVG本地依赖、按需模块、统计页延迟加载，ResizeObserver响应容器变化，卸载dispose；不加载外部CDN。tooltip用richText而非用户文本HTML。完整数据表补足图表标签截断及未知值说明。

月趋势只放置有确定月份（YYYY-MM或精确日），并且整个可能日期区间在筛选范围内的记录；年／未知／边界不确定记录留在coverage，不补造月份。缺少月份不代表零投入。Hunt目前只有累计值，没有M7会话明细，不能绘制每日次数／耗时，亦不把开始日期视为全部投入发生日。

投入按游戏+方法+原样单位分组。count/seconds仅合计已知值，并附known_count_hunts/known_duration_hunts与总Hunt数；完全缺失为null，真实0仍为0。不同游戏、方法或单位上下文不跨组合计或共用次数图比较；需收窄到单个上下文才绘制次数，明细仍分别保留。不因为单位文字相同就推断跨游戏／方法的次数含义相同。超出JavaScript安全整数或累计上溢的组合计保持null并显式overflow，不输出失真数字。

效率为描述性已记录子集：仅已完成、游戏／方法明确，所有未删除关联成果均明确self_hunted且原始游戏匹配，至少一个成果；包含意外闪光，不将交换／活动关联伪称自刷。次数效率还要求次数与单位已知，耗时效率要求正耗时；两个指标分别显示样本Hunt／成果数量，不共享错误分母。进行中／放弃、未知／混入其他来源、游戏不一致不自动计算。日期筛选时，累计投入无法拆分到子期间，效率保持null。来源筛选为非自刷时亦不算效率。不能据此推算真实概率、加成规则或运气。

资料覆盖仍为M5有限子集；本模块统计个人记录，不把未核验参考资料变成真实游戏机制。

## 实现依据

ECharts 6.1.0按npm registry元数据锁定，Apache-2.0；[按需导入](https://echarts.apache.org/handbook/en/basics/import/)和[容器大小与销毁](https://echarts.apache.org/handbook/en/concepts/chart-size/)遵照官方文档。前端依赖锁文件保留完整性校验；不改变Vue／Go／SQLite技术栈。


## T021 样本与身份校验

API在原effort中增加count_sample_status/time_sample_status（no_eligible_samples、single_sample、descriptive_subset）、median_count_per_shiny/median_seconds_per_shiny以及exclusions。均值仍按成果数加权；中位数取各合格Hunt的累计值／该Hunt成果数后求中位数，至少2个Hunt才给值，不能与按个体加权均值混称。单个样本仍可展示原描述值，明确提示样本不足；多个样本也不宣称代表总体概率。次数和耗时独立样本。

exclusions逐组提供未完成／未知游戏或方法／期间或来源筛选／无成果／关联成果证据不匹配／缺失次数、单位、耗时／零耗时／超安全范围等原因的Hunt数。原因可重叠，不能求和作为总排除数。累计总和上溢时null；单Hunt输入仍安全时，逐Hunt中位数可保留，并明示上溢。没有用0替代未知或生成无穷效率。

quality包含distinct_records、candidate_groups/candidate_records、candidates、candidates_truncated、count_comparison和note。永久ID按实际身份计数；相似候选沿用Excel导入的duplicateImport（当前物种+原始游戏+获得日期相同，包含未知）。这些字段不足以断言同一只，候选也可能是不同形态或多只同日成果；全部独立计入，不自动删除／合并／扣减或修改历史。候选最多显示20组，每组100个ID，完整候选总数保留并明确截断。

前端校验汇总一致、分组／月份／身份／候选计数、安全数值、未知值与已知覆盖、样本状态和中位数前提；异常响应清空旧图并报错。查询保持只读、事务一致，没有新增表或第二套数据层。
