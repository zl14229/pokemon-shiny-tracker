# M4 图鉴统计口径 v1

`GET /api/v1/dex`只读取本地SQLite，一致事务快照中统计，无外网请求。不修改个体/资料，不能根据收藏反推合法性。`GET /api/v1/reference/availability?game_id=&form_id=`可分页读取最小证据表；M4正式数据中为空，合成测试夹具不打包、不当作真实游戏证据。

参数：dimension=species|form（默认species），basis=historical|current（默认historical），scope=all|mainline|game（默认all；game必填game_id），collection_system（默认pokemon，指定游戏时默认其体系），display=all|legal，acquired=all|obtained|missing，holding=all|held|no_confirmed_held|unknown，q（多语言/slug/全国编号），page/page_size（1..100）。重复、空、未知参数拒绝。主系列仅采用已分类mainline的版本；游戏类别unknown不擅自归类，覆盖中报告未分类数。

- 物种目标按稳定species ID（对应唯一全国编号）去重，地区/多只个体不增加目标数。形态目标按稳定form ID去重，外部form ID与内部ID分开。
- 历史证据：当前物种/已明确原始物种形态，以及evolution/form_change事件before/after快照；每条个体对同一目标最多计1。无记录的进化前形态、空form_id不推断为默认形态。updated/纠错、location_change/转移事件不增加旧物种证据，registered的过时快照不重新计作历史。已释放/交换仍保留历史；软删除的错误/重复个体不参与统计。既有明确历史事件不会被后来的普通纠错自动撤销，完整历史编辑属于T023。
- 当前：只有holding_status=held且个体当前物种/形态才贡献持有；unknown不当作held，保留unknown_holding_count；历史进化前形态不贡献当前持有。
- 游戏：历史筛选使用个体原始获得游戏，表示“来源于该游戏的个体的已记录历程”；不是宣称其每次进化都在原游戏发生。当前持有筛选使用当前保存游戏。当前保存游戏绝不补填为原始获得游戏；HOME/storage可以筛选收藏位置但不能提供直接合法刷闪分母。转移不会新增个体或目标。all范围包含未知游戏的普通宝可梦记录；有游戏体系证据时按体系隔离，Sleep等单独统计，指定游戏自动采用该体系。
- 分母：当前范围中至少存在一条verified且来源URL/核对时间/数据版本有效的confirmed_available，且形态策略included。物种再去重。unknown/conflict/locked/transfer_only、storage和未审核/排除形态不入分母。这里是**本地已确认直接合法可获取子集**的完成率，不是完整游戏图鉴，也不是Planner/攻略完整性。某版本transfer_only不影响另一个版本独立confirmed_available的证据。
- 分子：选定历史/当前口径的已获得集合与分母取交集；observed_outside_denominator单列。0分母percentage=null，不显示0%或100%；已获未知目标仍可在完整资料中展示。
- 汇总针对所选dimension/basis/game/system，搜索、已获/缺失、holding、display、分页只改变列表，不偷偷改变总进度。missing_eligible只表示已核实分母中的缺失，不代表缺少全国所有目标。

响应：data返回目标、名称/来源/形态纳入状态、availability/eligible、historical_count/held_count/unknown_holding_count、所选口径obtained与去重record_ids。meta是筛选后列表分页；summary是范围内统计/分母规则；coverage包含本地实际物种/形态、版本分类缺口、核验记录数、资料和可获取性版本、中文待核实及明确partial提示。图鉴不返回第三方图片URL，未核验素材不镜像。

ShinyAvailability为schema6新增最小读取契约：游戏+形态唯一，复合外键防错配，四态与verification_status分开；verified必须有来源、链接、时间和版本。无写入API或自动联网填充；详细来源/方法/条件概率库是T026。备份/导出包括新表；恢复仅支持本程序当前schema，旧schema4/5备份先由对应旧版恢复新目录再升级。

多游戏/多形态聚合：存在一条直接可获取证据即可证明“在范围中至少一处已确认可获取”；反向断言locked/transfer_only需覆盖所有适用游戏及物种的全部已纳入形态。范围中存在未覆盖/冲突状态时保持unknown，不把一条仅传送/锁闪事实外推到整个范围。主形态未审核集合继续独立显示缺口。
