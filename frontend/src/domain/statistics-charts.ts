import type { EChartsCoreOption } from 'echarts/core'
import type { StatisticsResult } from '../api/statistics'
export function statisticsCharts(
  v: StatisticsResult,
): { title: string; option: EChartsCoreOption }[] {
  const mixedUnits = new Set(v.effort.map((e) => e.unit)).size > 1
  const grid = { left: 64, right: 20, top: 55, bottom: 75, containLabel: true }
  const bar = (title: string, names: string[], values: (number | null)[], yname: string) => ({
    title,
    option: {
      grid,
      xAxis: {
        type: 'category',
        data: names,
        axisLabel: { rotate: 25, overflow: 'truncate', width: 90 },
      },
      yAxis: { type: 'value', name: yname, min: 0 },
      series: [{ name: yname, type: 'bar', data: values, itemStyle: { color: '#6177b5' } }],
    } as EChartsCoreOption,
  })
  return [
    {
      title: '月度任务开始与成果',
      option: {
        grid,
        legend: { data: ['任务开始', '获得个体', '亲自刷闪'] },
        xAxis: { type: 'category', data: v.trend.map((m) => m.month) },
        yAxis: { type: 'value', min: 0, minInterval: 1 },
        series: [
          { name: '任务开始', type: 'line', data: v.trend.map((m) => m.started_hunts) },
          { name: '获得个体', type: 'line', data: v.trend.map((m) => m.individuals) },
          { name: '亲自刷闪', type: 'line', data: v.trend.map((m) => m.self_hunted) },
        ],
      },
    },
    bar(
      '按原始获得游戏的成果分布',
      v.by_game.map((g) => g.label),
      v.by_game.map((g) => g.individuals),
      '个体',
    ),
    {
      title: '成果来源分布',
      option: {
        legend: { bottom: 0, type: 'scroll' },
        series: [
          {
            name: '成果来源',
            type: 'pie',
            radius: ['35%', '60%'],
            data: v.by_source.map((s) => ({
              name:
                (
                  {
                    self_hunted: '亲自刷闪',
                    trade: '交换',
                    event: '活动',
                    other: '其他',
                    unknown: '未知',
                  } as Record<string, string>
                )[s.key] || s.label,
              value: s.individuals,
            })),
            label: { show: false },
          },
        ],
      },
    },
    bar(
      '按关联Hunt方法的成果分布',
      v.by_method.map((m) => m.label),
      v.by_method.map((m) => m.individuals),
      '个体',
    ),
    bar(
      '次数合计 · 按游戏／方法／单位分组',
      v.effort.map((g) => `${g.game} · ${g.method ?? '方法未知'} · ${g.unit ?? '单位未知'}`),
      v.effort.map((g) => (mixedUnits ? null : g.count)),
      '原单位',
    ),
    bar(
      '已记录耗时 · 按游戏／方法／单位分组',
      v.effort.map((g) => `${g.game} · ${g.method ?? '方法未知'} · ${g.unit ?? '单位未知'}`),
      v.effort.map((g) => (g.seconds === null ? null : g.seconds / 3600)),
      '小时',
    ),
    bar(
      '自刷成果平均耗时 · 可核算子集',
      v.effort.map((g) => `${g.game} · ${g.method ?? '方法未知'} · ${g.unit ?? '单位未知'}`),
      v.effort.map((g) => (g.seconds_per_shiny === null ? null : g.seconds_per_shiny / 3600)),
      '小时／个体',
    ),
  ]
}
