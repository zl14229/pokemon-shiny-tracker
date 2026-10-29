# M6 验收报告

状态：T019、T020、T021均verified，三个实现PR及各自必需检查均已确认成功并合入dev。本报告随最终验收收尾PR交付，收尾自身CI／merge和最终dev运行以实际记录及交付回复确认。

范围严格为T019→T020→T021；每项独立feature分支从当时最新远程dev开始，实现与验收收尾分别通过PR。冻结规格／Task依赖／ADR保持原文；不进入M7、不修改prod。没有新个人实体、数据库表、schema或会话数据层。

## 功能与口径

Dashboard在一致只读事务中汇总永久个体、Hunt状态与缺失字段，并直接调用M4 Dex引擎产生物种／形态×历史／当前四组summary与coverage。历史按有据的原始身份和历史证明，当前按确认持有的现形态；转移／进化不新增个体，Sleep等体系隔离。图鉴只显示本地已核实合法子集分母，未知分母百分比为null。

统计接口复用同一快照加载和T009查询验证：原始获得游戏与Hunt目标游戏分别筛选，当前位置不代替获得版本；方法／状态／数值条件要求关联匹配Hunt。来源只筛成果，任务数仍是任务队列。日期沿用区间重叠，未知日期不匹配日期范围；年份／模糊边界不擅自放到某月，不补造缺失月份的零投入。个人身份以ID计算，软删除排除，历史JOIN不会倍增。

次数、耗时明确覆盖已知Hunt数；完全未知为null，真实0保留。游戏／方法／单位构成独立上下文，单位文字相同也不直接推断语义可比；次数图需单一上下文，明细独立保留。没有HuntSession明细，不能从累计值计算每日投入，日期子期间不生成效率。

效率仅取已完成、游戏／方法明确、关联成果均明确自刷且原始游戏匹配的描述性子集，包含意外闪光；次数与耗时独立样本，正耗时才计算时间效率。均值按成果加权，中位数先算逐Hunt每成果值、至少2个Hunt才提供；单样本明确不足，多样本也不代表总体概率。排除原因可重叠，不能求和当总排除数。

疑似重复沿用Excel导入候选规则（当前物种＋原始游戏＋获得日期，含未知），可能是不同形态或多只成果；仅提示核对，仍保留各自永久ID，不自动合并、扣除或写历史。列表最多20组／每组100个ID，完整候选数量与截断状态可见。完整接口与边界见[STATISTICS.md](STATISTICS.md)。

## 图表验证

|图表／视图|数据与实际验证|
|---|---|
|Dashboard|真实数据库永久个体、三种Hunt状态、缺失覆盖；四组图鉴summary/coverage与M4同查询完全一致；进化、转移、放生、持有未知、软删除、Sleep隔离|
|月度任务开始／成果|只使用能确定月份且整段落在筛选内的日期；年／未知／边界不确定进入未放置数量；闰日和0001–9999边界与原列表筛选一致|
|原始游戏成果分布|原始1／当前2仍计原始1，进化或转移不新增个体|
|来源成果分布|自刷、交换、活动、其他、未知分离；关联Hunt不会使交换成果变成自刷|
|关联Hunt方法分布|精确已有方法文本；无关联／未知有独立分组，不凭概率资料猜方法|
|次数合计|保留原单位与已知覆盖；跨游戏／方法／单位不共图比较，未知值不补0|
|已记录耗时|标准秒换算显示小时；只合计已有数值，空值与0分离，无每日投入推论|
|自刷平均耗时与明细|120次／600秒、同Hunt两成果的合成验收值为60次／300秒每个体；多Hunt加权均值与逐Hunt中位数分别验证；来源／游戏不足、零耗时、期间筛选留空|
|身份候选／样本提示|3个Hunt／3个独立成果的真实API旅程，2个相似身份仍保留；2样本中位数与1样本不足并存；过滤后数量与明细准确，读统计不写持有或历史|

七张图使用锁定本地ECharts6.1.0 SVG与按需模块，无CDN／图片依赖。ResizeObserver响应容器变化，卸载dispose，tooltip使用richText。真实浏览器验证1280px／390px、七SVG实际宽度不超容器、页面无横向溢出、明细内部滚动；错误时清除旧图并提示重试。统计页延迟加载，不将图表依赖提前加载到首页。

## 检查证据

|检查|结果|
|---|---|
|冻结基线与治理|29 Task定义／依赖、16 ADR不变；治理校验与7拒绝／回归测试通过|
|Go|make check（格式／模块一致／vet／race／build）、make smoke通过；T021完整service race约125秒|
|前端|最终113单元／组件、typecheck/build/HTTPsmoke通过|
|真实前后端|12项接口集成通过；最终响应校验加强后再次12/12|
|浏览器|完整36/36通过（含MVP备份恢复、M4图鉴、M5规划/Bingo及M6）；最终响应加强后2个统计旅程再次通过|
|数值边界|未知／0、正耗时／零耗时、同名不同上下文单位、安全整数最大值／超界／累计上溢、无穷值拒绝、中位数与样本前提|
|身份与筛选|重复候选不扣除、永久身份进化转移、软删除、读操作不写历史、原始游戏／当前保存、体系关联边界、日期区间与部分精度|
|恢复回归|原MVP真实浏览器导出／ZIP备份／新目录恢复旅程通过；统计为派生只读数据，无新备份表或schema|

全部数据库为临时隔离库，未读取／修改真实个人库。受限进程不能监听端口时，在允许启动本地服务的环境重验通过；本机15080／15173无监听却连接超时，已只读定位并为E2E增加有界端口覆盖，使用45080／45173通过全套。仅结束本轮已知挂起测试进程，未停止未知服务；应用及CI默认端口不变。浏览器依赖仅临时解压，无系统安装。

自审按现有code-review技能由主代理逐文件完成；无配套子技能，不虚报独立子审，均标code-reviewed，不发送额外GitHub评论。发现并修复关联Hunt筛选跨体系和未知耗时伪0响应边界，均补回归；未跳过检查或降低断言。

首次T021分支CI [40791611786](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40791611786) 的Bingo旧夹具35/36：快CI里个体时间截到整秒后可能早于挑战StartedAt，被正确拒绝。修复仅等待登记时间的整秒前提并加明确断言，保留业务与完成检查；修复后本机完整36/36再次通过，最终HEAD PR CI40792367276／push CI40792361688均四项success，未把旧失败当成功。

## PR、CI与Merge Commit

|阶段|PR|最终PR CI|Merge Commit|合并后dev CI|
|---|---|---|---|---|
|T019实现|[#45](https://github.com/zl14229/pokemon-shiny-tracker/pull/45)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40786398584)|ed3ebc4ef539a2850e3b9c59665c11873c40758c|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40787093786)|
|T019收尾|[#46](https://github.com/zl14229/pokemon-shiny-tracker/pull/46)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40787203727)|af17ba706b577dff5597e582f619b7af51292426|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40787777389)|
|T020实现|[#47](https://github.com/zl14229/pokemon-shiny-tracker/pull/47)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40789003802)|d57f5aa5b6612e728f429e5d7acef633a6b34144|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40789648343)|
|T020收尾|[#48](https://github.com/zl14229/pokemon-shiny-tracker/pull/48)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40789718067)|119beb986c02b651361c5b9acb73c1b9f81f3088|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40790417292)|
|T021实现|[#49](https://github.com/zl14229/pokemon-shiny-tracker/pull/49)|[success](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40792367276)|786c9b4e1350d07ec1506c6452abf9a9dc28fb45|[实际运行](https://github.com/zl14229/pokemon-shiny-tracker/actions/runs/40793219031)|
|T021／M6收尾|[#50](https://github.com/zl14229/pokemon-shiny-tracker/pull/50)|[实际检查记录](https://github.com/zl14229/pokemon-shiny-tracker/pull/50/checks)|[实际合并记录](https://github.com/zl14229/pokemon-shiny-tracker/pull/50)|[dev运行记录](https://github.com/zl14229/pokemon-shiny-tracker/actions?query=branch%3Adev)，精确运行见交付回复|

最终实现HEAD：T019 a2363e213c816a0034dbea8ac9f0936b9c6c71ed；T020 154f1d81eb7b755d939e00a06ac734add765965a；T021 c35bb5f9c75e0fd5bb0d85163e0ba0e228011bea。各Task记录包含push CI和逐项自审。收尾自身的精确merge SHA与最终dev CI在交付回复中提供，避免文件包含自身提交哈希的循环。

## 已知限制与停止边界

这里是个人登记数据的描述，不能估算实际闪光概率、完整游戏机制、运气或因果关系。未知日期不填造，累计投入不拆分，单样本不作总体结论，候选不等于重复身份。资料沿用M5已核实有限子集（朱／紫各12、共12唯一目标／24版本与形态组合）；中文内容跨站核验、完整游戏版本／概率／图片授权缺口仍保持未知，没有新爬取或降低资料可信度。

统计页约581KB（gzip199KB），首页约537KB（gzip178KB），构建有体积建议但成功；未声称Windows原生、深色主题、真实全游戏完整数据或正式发布验收。新增ECharts／zrender／tslib许可证保存在[licenses](licenses/README.md)，不等于百科内容再用授权。

GitHub实际回读：dev默认、四项strict必需检查、无bypass、仅Merge Commit、保留合并后分支，见[evidence/github-rulesets-m6.json](evidence/github-rulesets-m6.json)。prod基准dc68988b7c76bf23ae3b68b588d15858f50b742e，交付前再次只读确认；三个feature分支保留状态亦在交付前回读。M7 T022–T025保持planned，完成M6后停止。
