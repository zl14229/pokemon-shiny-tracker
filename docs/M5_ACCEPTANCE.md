# M5 集成验收与资料覆盖报告

状态：T026、T027、T028 均 verified；三个实现 PR 及各自合并后的 dev CI 已确认成功。本报告随 M5 收尾 PR 交付，收尾自身 CI／merge 记录见下表动态链接及交付回复。

范围严格为 T026 → T027 → T028，依赖顺序执行；不进入 M6、不发布或修改 prod。冻结规格、Data Source Policy、ADR-005/006/007/009/010/015/016 均保持原文。每个 Task 独立 feature 分支，从当时最新远程 dev 开始；实现和 verified 收尾分别通过 PR。

## 正式资料覆盖

|项目|实际覆盖／状态|
|---|---|
|基础图鉴|沿用 M4：1025 物种、1579 形态；1227 纳入、172 排除、180 待核实；没有将形态口径未知自动纳入|
|已核实直接可刷|12 唯一物种／12 唯一形态；朱、紫各 12 条，共 24 个游戏与形态组合|
|适用范围|朱／紫本篇南第一区的所选普通野外来源；含帕底亚乌波，不含开场教学、赠送、团体战、活动或其他版本推论|
|负面状态|正式数据未新增锁闪／仅传送断言；四类状态与冲突／未知的排除通过隔离合成测试，不假装是正式资料|
|遭遇率|24 条均未知（null）；不把出现权重或群体率换成遭遇率|
|闪光概率|只保存普通野外无加成基础 1/4096 的条件规则，各条保留来源与范围；不实现完整跨游戏／加成概率库|
|攻略|24 条均 partial；完整时段、天气、最优路线仍有缺口|
|来源批次|m5-sv-wild-2026-10-28-v1；逐条 source/version/scope/checked_at、交叉核对链接、审核快照与校验和|
|中文、图片|沿用 M4 来源提供／缺失／冲突状态；52Poké 内容交叉核验仍为0；没有已授权图片集，继续合理占位|
|游戏资料|本地4个种子版本，只有朱／紫有上述可刷子集；版本类别仍沿用未核实状态，不宣称全系列或完整游戏覆盖|

数据数量由仓库实际快照统计，不能据此推算完整游戏覆盖百分比。完整来源与声明见 [HUNT_DATA.md](HUNT_DATA.md)。

查阅少量来源页面并记录最小结构化事实与链接：[Serebii 地点](https://www.serebii.net/pokearth/paldea/southprovinceareaone.shtml)、[Bulbapedia 地点](https://bulbapedia.bulbagarden.net/wiki/South_Province_(Area_One))、[Serebii 机制](https://www.serebii.net/scarletviolet/shinypokemon.shtml)、[Bulbapedia 第九世代机制](https://bulbapedia.bulbagarden.net/wiki/Shiny_Pok%C3%A9mon#Generation_IX)。复核时间 2026-10-28T16:00:44Z（Asia/Shanghai 2026-10-29 00:00）。版本范围明确到作品和方法；源页未固定软件构建号，不宣称验证任意补丁。

未批量抓取站点、复制网页正文／图片、镜像地图或实现未经授权的同步器。[52Poké 机器读取规则](https://wiki.52poke.com/wiki/神奇宝贝百科:机器读取守则) 已再次核对，但未据此宣称资料或图片可无限再分发。批量、全文与图片再用许可尚未确认，相关能力未启用。基础实体继续使用已缓存 PokéAPI v2 官方资料与原始映射，未替换主来源。

## 集成功能与业务口径

T026：分别建模可获取性、直接可刷、攻略完整性、方法、地点、条件、遭遇率与闪光率。只读本地查询、暂存校验、完整新增差异、审核校验和、原子添加／冲突拒绝、幂等与批次撤销；不覆盖个人表、名字或稳定身份。完整增量替换／联网同步仍属于 T024。

T027：复用 T017 的历史证明和原始游戏范围，不把当前位置当作获得版本；默认全系列历史未获得，支持所选版本缺失池、物种／形态、多版本、数量和进行中排除。先唯一目标均匀不放回，再均匀选择可刷版本。空池／不足保留实际数量，不添加未知目标。生成只产生草稿，保存后进入历史；逐个明确转换为／关联 Hunt，幂等且重验当前证据。详见 [PLANNER.md](PLANNER.md)。

T028：3×3／5×5，无免费格和重复格；多卡持久化、关联 Hunt、未开始格重抽与前后历史。已开始／完成／封存格不能重抽；未刷新但已有合格新成果也不能靠重抽绕过。行、列、两条对角线自动判定；全格完成自动封存，已有连线可明确结束，任意进行中挑战可放弃并保留历史。详见 [BINGO.md](BINGO.md)。

默认 Bingo 成果必须是挑战后新登记、亲自刷闪、原始目标／游戏／形态明确匹配、当前本地合法直接可刷，且日期精度足以证明在挑战之后。旧收藏、交换、活动、单纯传送、来源或形态未知、年／月／缺失日期、未来日期及同日仅有日期，均不自动完成。按创建时固定日历时区比较，不能用 UTC 日期误判另一时区的同一天。

仅时间精度不足时，可以显式补充获得时刻、核验依据并确认。时间必须晚于创建且不在未来，与原记录已知日期一致；不能覆盖来源、旧身份、原始目标、游戏、形态或合法性。证据标为用户时间核验，不当作官方来源，不覆盖个体日期。完成使用同一个永久 ShinyRecord ID，不因进化、转移或格子确认新增个体。

当前正式12目标不足25，5×5界面诚实拒绝；完整5×5通过仅存在于隔离测试库的26目标夹具验证，并非新增正式事实。3×3也只有筛选后至少9个已核实唯一目标时才可创建。

## 验证证据

|检查|实际结果／证据|
|---|---|
|冻结基线与治理|校验通过，7个拒绝／回归测试通过，29 Task 编号与依赖未变|
|后端|make check（格式、模块一致、vet、race、build）、make smoke 通过；覆盖实际启动／停止、引用和事务|
|前端|最终102单元／组件通过，typecheck、build、HTTP启动／预览smoke通过|
|真实前后端接口|10项通过，含来源快照、保存／幂等Hunt与Bingo重抽／核验状态|
|完整浏览器|最终实现完整33项通过；网络结果不明重抽自审修复后再次全套33/33通过|
|抽样|30,000次固定序列多版本偏权反例、不同版本数的目标仍等权，唯一目标不放回|
|非法／未知候选|锁闪、仅传送、未知、冲突不抽取；方法缺失／概率未知不补造；空池／不足如实返回|
|历史与身份|原始获得与当前保存游戏分离、历史进化证明、未知形态不推断、旧记录不能当新成果|
|Bingo状态|未开始重抽、开始／完成／封存限制、期望目标检查、幂等、未同步合格成果保护、无替代目标整体取消|
|日期与来源|旧收藏、交换、活动、转移、来源未知、缺失／年／月／同日／未来日期、Asia/Shanghai日历边界、人工时间证据不能绕过来源|
|连线与恢复|3×3／5×5行列与两对角线；完整25格12连线，卡片／个体／证据／审计新目录备份恢复一致|
|真实浏览器旅程|多卡 → 开始格子 → 重抽 → 新测试个体 → 同日待核验 → 显式时间核验 → 连线 → 完成／放弃历史 → 离线重开；390px页面无横向溢出，棋盘内部滚动|
|个人资料隔离|暂存／预览／冲突／撤销不改变个人记录；生成／保存不创建Hunt或个体；明确转换只建一个Hunt，重复安全；全部测试使用临时库|

本机曾遇到中文字体下既有导入说明的长英文状态串溢出，修复标签换行后完整回归通过；一次浏览器端口临时占用只读检查后重试成功，未停止未知服务。测试端口现为15080/15173，避开本机临时客户端范围。收尾push CI [40738662047](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40738662047) 又发现既有源客户端按请求开始计时，使慢响应可能消耗冷却间隔。修复为完整网络响应结束后等待原有间隔，保留串行与原阈值；注入延迟响应强化原测试，重复20次race、完整后端检查和启动smoke通过，随收尾PR完整CI再验。没有降低检查要求或跳过失败。

浏览器运行依赖仅解压在临时目录，没有系统安装；测试数据库均在临时目录，未读取或修改真实个人库。未声称 Windows 原生、深色主题或公开发布验收。前端本地产物约526KB（gzip约174KB），构建有体积建议提示，构建与运行验收成功。

## PR、CI 与 Merge Commit

已合并的实现与收尾阶段均为工具实际确认的证据。最后收尾 PR 自身的检查与 Merge Commit 使用实际 GitHub 记录链接，最终 dev CI 和自身 merge SHA 在交付回复中确认，避免文件引用包含自身的提交哈希循环。

|任务／阶段|PR|最终PR CI|Merge Commit|合并后dev CI|
|---|---|---|---|---|
|T026 实现|[#39](https://github.com/zl14229/pokemon-shiny-tracker/pull/39)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40723403175)|0c64c53530877f9b9f88ba908c4ebf504e3c3a26|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40723955544)|
|T026 收尾|[#40](https://github.com/zl14229/pokemon-shiny-tracker/pull/40)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40724775480)|d6e3a821c069b0892d9c415f7d03d7b26b7a7ca2|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40725350391)|
|T027 实现|[#41](https://github.com/zl14229/pokemon-shiny-tracker/pull/41)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40727997060)|c4103561a5995c9c5083c7b6f9f3dd578ead7c2f|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40729655636)|
|T027 收尾|[#42](https://github.com/zl14229/pokemon-shiny-tracker/pull/42)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40730314957)|770a96a288186bc62cd16af6f569b3705138b979|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40731222456)|
|T028 实现|[#43](https://github.com/zl14229/pokemon-shiny-tracker/pull/43)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40736415489)|0e3b606c6d42ce636f577ad593ccbc9f3add36e8|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40737558145)|
|T028 / M5 收尾|[#44](https://github.com/zl14229/pokemon-shiny-tracker/pull/44)|[实际检查记录](https://github.com/zl14229/pokemon-shiny-tracker/pull/44/checks)|[实际 Merge Commit 记录](https://github.com/zl14229/pokemon-shiny-tracker/pull/44)|[dev 运行记录](https://github.com/zl14229/pokemon-shiny-tracker/actions?query=branch%3Adev)，精确运行见交付回复|

T026最终实现HEAD58a429e3b7e59e6d16815dc57aadc06973adb408；T027最终实现HEAD28463f6b924cf75ef5e30950cbaf6067d81ab196；T028最终实现HEADbbbd2b2aa19f65c56d8101a757465c5051695636。任务记录保留push CI等细节。每个实现和收尾均要求实际四项必需检查成功、主代理逐文件自审；仅有code-review orchestrator而无子技能，不虚报独立子审。没有绕过GitHub规则或虚报成功。

## 已知缺口与停止边界

资料未覆盖全游戏；中文内容交叉核对、全部天气时段／路线、补丁构建号、条件加成概率、图片再用与批量访问授权仍未核实。默认保持未知，未添加完整概率库或未经授权的采集。生成草稿保留审计，尚无清理器。已确认格子使用历史证据快照，后续编辑或删除个人记录不会静默改写挑战历史。

schema9备份包含20表；旧schema备份应以对应旧版恢复到新目录后升级，不覆盖真实数据。M6 T019–T021和后续Task保持planned。prod实际回读仍为dc68988b7c76bf23ae3b68b588d15858f50b742e，本轮未修改。GitHub规则回读见[evidence/github-rulesets-m5.json](evidence/github-rulesets-m5.json)：dev默认、四项strict必需检查、无bypass、Merge Commit、合并后不删除源分支。最终交付附三个源分支保留及prod不变的实际核对结果。
