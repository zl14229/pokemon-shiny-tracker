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

[PR CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40310199039)与[dev CI](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40310771281)全部success。实际merge `25c355fd56756a52c0c8aaf077cdfe620bec5d14`，最终HEAD `4b806dcd6f7452d0ba1d73907e63993ea8d4b5c2`。T016 verified；同源分支收尾PR归档，后续补链接。

## T017 / T018

待T016 verified后顺序执行。统计口径、端到端验收、所有PR/CI/merge及最终风险将在各Task实际通过后补全。

## 数据保护与运行边界

所有验收只使用自建临时SQLite、匿名个体与临时附件，不触碰真实个人库。基础资料导入不写Hunt/ShinyRecord/历史；内置资料有冲突时事务拒绝并保留旧资料继续运行。schema5 ZIP备份/导出含新增资料表；旧schema4 ZIP需旧版先恢复到新目录后升级，不能在新版本中原地覆盖。Windows原生、跨机器字体/文件系统、全游戏资料完整性及正式发布未验收。
