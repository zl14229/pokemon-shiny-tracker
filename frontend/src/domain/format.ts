import type { Catalog, Reference } from '../api/models'
export const statusLabels = {
  in_progress: '进行中',
  completed: '已完成',
  abandoned: '已放弃',
} as const
export const holdingLabels = {
  held: '持有中',
  traded: '已交换',
  released: '已放生',
  unknown: '未记录',
} as const
export const sourceLabels = {
  self_hunted: '自刷',
  trade: '交换',
  event: '活动',
  other: '其他',
  unknown: '未记录',
} as const
export const emptyCatalog = (): Catalog => ({ species: [], forms: [], games: [] })
export function label(item: Reference): string {
  return item.name_zh && item.name_status !== 'conflict' ? item.name_zh : item.name
}
export function referenceName(items: Reference[], id: number | null): string {
  if (id === null) return '未记录'
  const item = items.find((item) => item.id === id)
  return item ? label(item) : `记录 #${id}（资料暂不可用）`
}
export function referenceOption(item: Reference): string {
  return `${label(item)}${item.name_zh && item.name_status !== 'verified' ? ' · 中文待核验' : ''}`
}
export function duration(value: number | null): string {
  if (value === null) return '未记录'
  const hours = Math.floor(value / 3600),
    minutes = Math.floor((value % 3600) / 60),
    seconds = value % 60
  return `${hours ? `${hours}小时 ` : ''}${minutes ? `${minutes}分 ` : ''}${seconds || (!hours && !minutes) ? `${seconds}秒` : ''}`.trim()
}
