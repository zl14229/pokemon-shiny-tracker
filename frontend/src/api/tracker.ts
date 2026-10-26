import { ApiClient, ApiError, api, type RequestOptions } from './client'
import {
  type Catalog,
  type Guard,
  type Hunt,
  type HuntInput,
  type Page,
  type ShinyRecord,
  type ShinyInput,
  type ShinyWrite,
  type HistoryPage,
  isHistory,
  isHunt,
  isShiny,
  isSpecies,
  isForm,
  isGame,
  record,
  integer,
} from './models'

export class BeforeWriteError extends Error {
  readonly name = 'BeforeWriteError'
  constructor(error: unknown) {
    super(error instanceof ApiError ? error.message : '无法读取最新记录，请刷新后重试。')
  }
}
export function invalidResponse(): never {
  throw new ApiError(
    'invalid_response',
    '记录响应不完整或数值超出安全范围，请刷新后检查；暂不能在此编辑。',
  )
}
function entity<T>(body: unknown, guard: Guard<T>): T {
  if (!record(body) || !guard(body.data)) return invalidResponse()
  return body.data
}
export function page<T>(body: unknown, guard: Guard<T>): Page<T> {
  if (
    !record(body) ||
    !Array.isArray(body.data) ||
    !body.data.every(guard) ||
    !record(body.meta) ||
    !integer(body.meta.total) ||
    !integer(body.meta.page) ||
    body.meta.page < 1 ||
    !integer(body.meta.page_size) ||
    body.meta.page_size < 1 ||
    body.meta.page_size > 100 ||
    body.data.length > body.meta.page_size ||
    body.data.length > body.meta.total
  )
    return invalidResponse()
  return body as unknown as Page<T>
}
export function huntInput(h: Hunt): HuntInput {
  return {
    species_id: h.species_id,
    form_id: h.form_id,
    game_id: h.game_id,
    method: h.method,
    start_date: h.start_date,
    end_date: h.end_date,
    encounter_count: h.encounter_count,
    count_unit: h.count_unit,
    duration_seconds: h.duration_seconds,
    status: h.status,
    notes: h.notes,
  }
}
export class TrackerAPI {
  constructor(readonly client: ApiClient = api) {}
  async catalog(options?: RequestOptions): Promise<Catalog> {
    const [species, forms, games] = await Promise.all([
      this.references('species', isSpecies, options),
      this.references('forms', isForm, options),
      this.references('games', isGame, options),
    ])
    return { species, forms, games }
  }
  async games(options?: RequestOptions) {
    return this.references('games', isGame, options)
  }
  private async references<T>(
    path: string,
    guard: Guard<T>,
    options?: RequestOptions,
  ): Promise<T[]> {
    const items: T[] = []
    let total: number | undefined
    do {
      const body = await this.client.getJSON(
        `/reference/${path}?limit=500&offset=${items.length}`,
        options,
      )
      if (
        !record(body) ||
        !Array.isArray(body.data) ||
        !body.data.every(guard) ||
        !record(body.meta) ||
        !integer(body.meta.total) ||
        body.meta.total > 100_000 ||
        body.meta.offset !== items.length ||
        (total !== undefined && body.meta.total !== total) ||
        body.data.length > 500
      )
        return invalidResponse()
      total = body.meta.total
      if (body.data.length === 0 && items.length < total) return invalidResponse()
      items.push(...(body.data as T[]))
      if (items.length > total) return invalidResponse()
    } while (items.length < total)
    return items
  }
  async hunts(query = '', options?: RequestOptions): Promise<Page<Hunt>> {
    return page(await this.client.getJSON(`/hunts${query ? `?${query}` : ''}`, options), isHunt)
  }
  async hunt(id: number, options?: RequestOptions): Promise<Hunt> {
    return entity(await this.client.getJSON(`/hunts/${id}`, options), isHunt)
  }
  async createHunt(input: HuntInput): Promise<Hunt> {
    return entity(await this.client.postJSON('/hunts', input), isHunt)
  }
  async updateHunt(id: number, input: HuntInput, expectedUpdatedAt: string): Promise<Hunt> {
    let current: Hunt
    try {
      current = await this.hunt(id)
    } catch (e) {
      throw new BeforeWriteError(e)
    }
    if (current.updated_at !== expectedUpdatedAt)
      throw new ApiError('http', '记录已在其他页面修改，请先重新加载再编辑。', 409, 'stale_record')
    return entity(await this.client.putJSON(`/hunts/${id}`, input), isHunt)
  }
  async shinies(query = '', options?: RequestOptions): Promise<Page<ShinyRecord>> {
    return page(
      await this.client.getJSON(`/shiny-records${query ? `?${query}` : ''}`, options),
      isShiny,
    )
  }
  async shiny(id: number, options?: RequestOptions): Promise<ShinyRecord> {
    return entity(await this.client.getJSON(`/shiny-records/${id}`, options), isShiny)
  }
  async createShiny(input: ShinyWrite): Promise<ShinyRecord> {
    return entity(await this.client.postJSON('/shiny-records', input), isShiny)
  }
  async updateShiny(
    id: number,
    input: ShinyWrite,
    expectedUpdatedAt: string,
  ): Promise<ShinyRecord> {
    let current: ShinyRecord
    try {
      current = await this.shiny(id)
    } catch (e) {
      throw new BeforeWriteError(e)
    }
    if (current.updated_at !== expectedUpdatedAt)
      throw new ApiError('http', '记录已在其他页面修改，请先重新加载再编辑。', 409, 'stale_record')
    return entity(await this.client.putJSON(`/shiny-records/${id}`, input), isShiny)
  }
  async history(id: number, currentPage = 1, options?: RequestOptions): Promise<HistoryPage> {
    const body = await this.client.getJSON(
      `/shiny-records/${id}/history?limit=20&offset=${(currentPage - 1) * 20}`,
      options,
    )
    if (
      !record(body) ||
      !Array.isArray(body.data) ||
      !body.data.every(isHistory) ||
      !record(body.meta) ||
      !integer(body.meta.total) ||
      body.meta.limit !== 20 ||
      body.meta.offset !== (currentPage - 1) * 20 ||
      typeof body.meta.record_deleted !== 'boolean' ||
      body.data.length > 20
    )
      return invalidResponse()
    return body as unknown as HistoryPage
  }
  async results(
    huntID: number,
    currentPage = 1,
    options?: RequestOptions,
  ): Promise<Page<ShinyRecord>> {
    return page(
      await this.client.getJSON(
        `/shiny-records?hunt_id=${huntID}&page=${currentPage}&page_size=20&sort=id&order=asc`,
        options,
      ),
      isShiny,
    )
  }
}
export const tracker = new TrackerAPI()

export function shinyInput(s: ShinyRecord): ShinyInput {
  return {
    hunt_id: s.hunt_id,
    original_species_id: s.original_species_id,
    original_form_id: s.original_form_id,
    current_species_id: s.current_species_id,
    current_form_id: s.current_form_id,
    original_game_id: s.original_game_id,
    current_game_id: s.current_game_id,
    current_location: s.current_location,
    obtained_date: s.obtained_date,
    source: s.source,
    holding_status: s.holding_status,
    nickname: s.nickname,
    gender: s.gender,
    nature: s.nature,
    ability: s.ability,
    ball: s.ball,
    level: s.level,
    notes: s.notes,
  }
}
