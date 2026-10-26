# M4 — 全国资料、双层图鉴与收藏展示验收

状态：in_progress。授权仅T016–T018；M5/Random Hunt/Bingo/完整机制概率库未开始，prod不修改。时间：2026-10-26（Asia/Shanghai）。完成状态必须以最终PR/CI/合并证据为准。

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

[52Poké机器规则](https://wiki.52poke.com/wiki/神奇宝贝百科:机器读取守则)、[robots](https://wiki.52poke.com/robots.txt)、[版权FAQ](https://wiki.52poke.com/wiki/神奇宝贝百科:常见问题)已查阅，不据此假定无限制批量再用/图片授权；本次无正文批量抓取、无图集镜像。Serebii、Bulbapedia、Pokémon Database和官方的逐游戏合法性与机制交叉核验未实施，保持未知，等待T026，未未经验证标为可信事实。

## T016

实现PR：[33](https://github.com/zl14229/pokemon-shiny-tracker/pull/33)。稳定映射、多语言/冲突回退、保守形态策略、来源与覆盖API、schema5离线事务增补/预览/幂等/冲突回滚完成；旧个人表和身份不改写。

本地make check/make smoke、前端typecheck/63测试/build/HTTP smoke/8真实API、26真实Chromium全部通过。首次浏览器缺共享库，解包临时依赖后复验；全国资料扩大暴露旧测试编辑页就绪竞态，改为等待disabled选择器可用后操作并完整通过，未跳过断言/放宽超时。

[PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40310199039)与[dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40310771281)全部success。实际merge `25c355fd56756a52c0c8aaf077cdfe620bec5d14`，最终HEAD `4b806dcd6f7452d0ba1d73907e63993ea8d4b5c2`。T016 verified；[收尾PR #34](https://github.com/zl14229/pokemon-shiny-tracker/pull/34)已Merge Commit `decac12a60b1a48f32a31d7455014c287451027b`，[收尾PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40311239858)及[收尾dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40311651424)全部success。

## T017

实现[PR #35](https://github.com/zl14229/pokemon-shiny-tracker/pull/35)，最终HEAD `d2b7008e5bae10dc52f5a83a7499f0996a5c4f30`；[PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40313923837)与[push CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40313917960)全部success。Merge Commit `edff5bbf1f9e4f0583877d0b466214e34243d17b`，[dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40314285286)全部success。T017 verified，收尾PR随后补链接。

只读离线引擎完成物种/持久形态、历史/当前、全系列/主系列/指定游戏/独立体系、完整/合法子集和组合查询。按每个体每目标去重；明确进化/形态事件贡献历史，转移/普通纠错不虚增；当前仅held且当前形态，空form不推断；历史原始游戏与当前保存游戏不混用，Sleep隔离。分母仅本地verified confirmed_available+included，分子取交集、零分母null，未知/争议/仅传送/storage排除；不将个别游戏的仅传送/锁闪事实推广到未覆盖版本。精确口径见[DEX.md](DEX.md)。

本地make check/make smoke、前端typecheck/63单元组件/build/8真实API/26Chromium通过。核心自动验收覆盖物种/形态去重、连续进化/形态、跨游戏转移/交易、历史/held、纠错/软删除、未知形态/持有、Sleep未知当前位置隔离、缺失/争议/不可信证据/仅传送/storage、部分范围不外推、分母交集/零分母、分页/组合/拒绝错误查询。正式可获取性表0行；用于50%等正例的合成事实只在临时测试库，未打包为游戏事实。schema6备份含新表。

初次PR CI成功但重复push的浏览器依赖安装长时运行，GitHub 405实际阻止合并；无bypass。复审发现部分范围transfer-only聚合问题后修复/加测，最终HEAD的PR与push完整成功后才合并。首次本机端口瞬时占用重新独立启动，未停止他人服务。

## T018

待T017 verified及收尾后执行图鉴界面与M4端到端验收。


## 数据保护与运行边界

所有验收只使用自建临时SQLite、匿名个体与临时附件，不触碰真实个人库。基础资料导入不写Hunt/ShinyRecord/历史；内置资料有冲突时事务拒绝并保留旧资料继续运行。当前schema6 ZIP备份/导出含新增资料表；旧schema4/5 ZIP需对应旧版先恢复到新目录后升级，不能在新版本中原地覆盖。Windows原生、跨机器字体/文件系统、全游戏资料完整性及正式发布未验收。
