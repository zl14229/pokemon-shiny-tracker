# M4 — 全国资料、双层图鉴与收藏展示验收

状态：verified（T016–T018工程验收完成；资料核验仍按下表明确分级）。授权仅T016–T018；M5/Random Hunt/Bingo/完整机制概率库未开始，prod不修改。时间：2026-10-26（Asia/Shanghai）。完成状态必须以最终PR/CI/合并证据为准。

## 全国资料覆盖与来源

|项目|本次快照|核验边界|
|---|---:|---|
|全国物种|1,025（#1–#1025）|PokéAPI官方固定版本的v2同源CSV；不是未来版本的永久全量声明|
|形态原始资料|1,579|持久/临时/未审核分类均保留|
|主形态纳入|1,227|基于来源标志及本地保守策略v1，不代表逐条跨站核验|
|临时/排除形态|172|主形态完成率排除|
|未审核形态|180|不自动纳入主形态分母|
|简体名称源值|物种1,025；形态961|均source_provided，不能称为官方译名已核验|
|简体形态名称缺失|618|英语→稳定slug回退，不推断或机器翻译|
|英语形态名称缺失|72|稳定slug回退|
|源语言|12|de/en/es/es-419/fr/it/ja/ja-hrkt/ja-roma/ko/zh-hans/zh-hant|
|52Poké条目内容交叉核验|0|全部待核实；逐物种核对链接可查询|
|游戏种子|4|red/blue/scarlet/violet；未宣称全游戏覆盖/游戏类别全部核实|
|正式合法闪光可获取性证据|0|M4引擎可读取带证据的本地状态；实际游戏事实补充归T026|
|图片复用核验|0|不打包第三方图集，使用占位|

主来源：PokéAPI v2及其[官方v2数据仓库](https://github.com/PokeAPI/pokeapi/tree/2bf639e5572812a67d2b94903fdcb9d095f5aeb2/data/v2/csv)。固定revision `2bf639e5572812a67d2b94903fdcb9d095f5aeb2`，逐文件URL/SHA256/采集时间及逐实体外部身份/本地检查时间保存在离线快照/SQLite中，原始缓存仓库外。`scripts/stage_catalog.py --offline`重建与内置快照逐字节一致。3条v2 ID/name抽样结果见[实际证据](evidence/m4-pokeapi-samples.json)，抽样不声称逐条API核验。来源许可附[PokéAPI原文](POKEAPI_LICENSE.md)。

[52Poké机器规则](https://wiki.52poke.com/wiki/神奇宝贝百科:机器读取守则)、[robots](https://wiki.52poke.com/robots.txt)、[版权FAQ](https://wiki.52poke.com/wiki/神奇宝贝百科:常见问题)已查阅，不据此假定无限制批量再用/图片授权；本次无正文批量抓取、无图集镜像。Serebii、Bulbapedia、Pokémon Database和官方的逐游戏合法性与机制交叉核验未实施，保持未知，等待T026，未将未经验证的资料标为可信事实。

## T016

实现PR：[33](https://github.com/zl14229/pokemon-shiny-tracker/pull/33)。稳定映射、多语言/冲突回退、保守形态策略、来源与覆盖API、schema5离线事务增补/预览/幂等/冲突回滚完成；旧个人表和身份不改写。

本地make check/make smoke、前端typecheck/63测试/build/HTTP smoke/8真实API、26真实Chromium全部通过。首次浏览器缺共享库，解包临时依赖后复验；全国资料扩大暴露旧测试编辑页就绪竞态，改为等待disabled选择器可用后操作并完整通过，未跳过断言/放宽超时。

[PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40310199039)与[dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40310771281)全部success。实际merge `25c355fd56756a52c0c8aaf077cdfe620bec5d14`，最终HEAD `4b806dcd6f7452d0ba1d73907e63993ea8d4b5c2`。T016 verified；[收尾PR #34](https://github.com/zl14229/pokemon-shiny-tracker/pull/34)已Merge Commit `decac12a60b1a48f32a31d7455014c287451027b`，[收尾PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40311239858)及[收尾dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40311651424)全部success。

## T017

实现[PR #35](https://github.com/zl14229/pokemon-shiny-tracker/pull/35)，最终HEAD `d2b7008e5bae10dc52f5a83a7499f0996a5c4f30`；[PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40313923837)与[push CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40313917960)全部success。Merge Commit `edff5bbf1f9e4f0583877d0b466214e34243d17b`，[dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40314285286)全部success。T017 verified。[收尾PR #36](https://github.com/zl14229/pokemon-shiny-tracker/pull/36)实际merge `922781666921e7f58b39be1f974430fea9a393f0`，[收尾PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40314837778)及[收尾dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40315237032)全部success。

只读离线引擎完成物种/持久形态、历史/当前、全系列/主系列/指定游戏/独立体系、完整/合法子集和组合查询。按每个体每目标去重；明确进化/形态事件贡献历史，转移/普通纠错不虚增；当前仅held且当前形态，空form不推断；历史原始游戏与当前保存游戏不混用，Sleep隔离。分母仅本地verified confirmed_available+included，分子取交集、零分母null，未知/争议/仅传送/storage排除；不将个别游戏的仅传送/锁闪事实推广到未覆盖版本。精确口径见[DEX.md](DEX.md)。

本地make check/make smoke、前端typecheck/63单元组件/build/8真实API/26Chromium通过。核心自动验收覆盖物种/形态去重、连续进化/形态、跨游戏转移/交易、历史/held、纠错/软删除、未知形态/持有、Sleep未知当前位置隔离、缺失/争议/不可信证据/仅传送/storage、部分范围不外推、分母交集/零分母、分页/组合/拒绝错误查询。正式可获取性表0行；用于50%等正例的合成事实只在临时测试库，未打包为游戏事实。schema6备份含新表。

初次PR CI成功但重复push的浏览器依赖安装长时运行，GitHub 405实际阻止合并；无bypass。复审发现部分范围transfer-only聚合问题后修复/加测，最终HEAD的PR与push完整成功后才合并。首次本机端口瞬时占用重新独立启动，未停止他人服务。

## T018

图鉴页面与收藏卡片已实现：物种/形态、历史/当前、游戏/体系、记录/持有/合法子集、多语言搜索/分页、完成度/分母外/已核实缺失、覆盖与名称/形态状态、来源/具体个体证明。只有本地API请求，无第三方图集镜像；图片缺少或加载失败有具名占位。应用筛选同步隐藏旧结果，取消旧请求，错误清除旧数据并支持重试，客户端验证安全整数、分母/分子/百分比和个体证明。

本地75项单元组件、类型/构建/HTTP smoke/8真实API/29真实Chromium、Go check/race/vet/build/smoke、治理及7回归全部通过。修复首次全套（28通过1失败）暴露的hash更新窗口：旧卡片断言提前满足、下一次选择被上一URL覆盖；现提交即pending并加组件回归，完整29场景通过，未放宽超时/断言。占位图使用CSS避免缺字体图标，额外图鉴3场景复验通过。[实现PR #37](https://github.com/zl14229/pokemon-shiny-tracker/pull/37)，最终HEAD `f4db8cbe727f6a6c5d5b3b4143a3395786261726`；[PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40317598643)、[push CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40317535019)与[dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40317953752)全部success。实际双父merge `cdb7e424b3721650388ba74a4135a1065b0ab47b`。最终实现HEAD本机完整29场景也再次通过。T018 verified；最后[收尾PR #38](https://github.com/zl14229/pokemon-shiny-tracker/pull/38)自身检查/合并由其GitHub记录提供。


## 数据保护与运行边界

所有验收只使用自建临时SQLite、匿名个体与临时附件，不触碰真实个人库。基础资料导入不写Hunt/ShinyRecord/历史；内置资料有冲突时事务拒绝并保留旧资料继续运行。当前schema6 ZIP备份/导出含新增资料表；旧schema4/5 ZIP需对应旧版先恢复到新目录后升级，不能在新版本中原地覆盖。Windows原生、跨机器字体/文件系统、全游戏资料完整性及正式发布未验收。

## M4端到端验收矩阵

|验收|证据/结果|
|---|---|
|全国ID、地区形态/多语言、离线暂存一致|Go reference/catalog与API测试；离线重建逐字节一致，v2三条抽样匹配；1025/1579计数通过|
|外部资料不覆盖个人收藏|事务预览/冲突回滚/幂等；已有个体前后JSON/稳定ID一致，只有资料表增补|
|物种/持久形态去重|同种多形态/多只个体分别计目标与个体，分子只与合法持久分母交集；自动测试通过|
|连续进化、形态和转移|同一ID保持；历史证明原始/中间/当前形态，转移不新增个体；Go与真实浏览器通过|
|历史/held/未知、交易/释放、纠错/软删除|历史保留、仅held当前形态计当前；纠错/缺失形态不推断；自动验收通过|
|原始游戏/当前保存/Sleep独立体系|指定游戏两种口径与未知当前位置体系隔离通过；存储游戏不提供直接可刷分母|
|未知、冲突、仅传送/锁闪、部分范围|无证据不入分母，部分版本不推广反向状态，0分母null；合成证据正例50%与UI100%仅测试库/组件|
|前端中文与图片回退|冲突/缺失回退测试；待核实标记可见；图片error切占位/换源重试通过，默认无图集|
|实际离线重载|真实Chromium禁用外部HTTPS后重载图鉴、中文搜索和个体证明仍可读，通过|
|缺失目标/分页/移动端/错误|零分母缺失清单待核验、24项稳定页、390px无溢出、失败脱敏/重试/旧数据隐藏通过|
|MVP完整故事及备份恢复|全部29真实浏览器包含既有MVP创建/导入/附件/导出/ZIP新目录恢复及恢复后UI/API/个体历史一致，通过|
|GitHub实际保护|[2026-10-26实测回读](evidence/github-rulesets-m4.json)：dev默认、4项strict必需检查、无bypass、Merge Commit、保留源分支；pending检查曾实际阻止合并；prod初始SHA不变|

工程功能的verified不表示资料全量可信：52Poké逐条内容核对0、正式合法可获取性0、无核验图集。当前默认完成率和已核实缺失列表仍待资料核验；具体补充归T026及后续授权，M4结束后不开始它。所有正例事实均标明synthetic，只在临时测试环境。

## 全部PR与CI索引

|Task/阶段|PR|最终PR CI|实现或收尾merge|合并后dev CI|
|---|---|---|---|---|
|T016实现|[#33](https://github.com/zl14229/pokemon-shiny-tracker/pull/33)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40310199039)|25c355fd56756a52c0c8aaf077cdfe620bec5d14|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40310771281)|
|T016收尾|[#34](https://github.com/zl14229/pokemon-shiny-tracker/pull/34)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40311239858)|decac12a60b1a48f32a31d7455014c287451027b|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40311651424)|
|T017实现|[#35](https://github.com/zl14229/pokemon-shiny-tracker/pull/35)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40313923837)|edff5bbf1f9e4f0583877d0b466214e34243d17b|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40314285286)|
|T017收尾|[#36](https://github.com/zl14229/pokemon-shiny-tracker/pull/36)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40314837778)|922781666921e7f58b39be1f974430fea9a393f0|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40315237032)|
|T018实现|[#37](https://github.com/zl14229/pokemon-shiny-tracker/pull/37)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40317598643)|cdb7e424b3721650388ba74a4135a1065b0ab47b|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40317953752)|
|T018/M4最终收尾|[#38](https://github.com/zl14229/pokemon-shiny-tracker/pull/38)|[实际最终checks](https://github.com/zl14229/pokemon-shiny-tracker/pull/38/checks)|自身双父merge以PR实际记录为准|由最终交付记录提供|

最后收尾[PR #38](https://github.com/zl14229/pokemon-shiny-tracker/pull/38)的固定入口已归档；其自身merge SHA/最后run由GitHub与最终交付记录提供，避免无限自引用。报告不将尚未结束的CI写为success；只在最终checks/合并/dev检查确已成功后结束本次开发。

## 最终范围

T000–T018已verified；T026/T027/T028及M6/M7全部保持planned。没有Random Hunt、Bingo或完整游戏机制概率库、prod提交、Release/Tag或公网部署。所有M4源分支保留。`prod`仍为初始提交 `dc68988b7c76bf23ae3b68b588d15858f50b742e`。资料授权/覆盖不足的部分保持unknown，未以工程verified替代资料真实性核验。
