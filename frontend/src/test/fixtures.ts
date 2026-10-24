import type { Catalog, Hunt, ShinyRecord } from '../api/models'
export const base = {
  id: 1,
  created_at: '2026-10-24T00:00:00Z',
  updated_at: '2026-10-24T00:00:00Z',
}
export const catalog: Catalog = {
  species: [
    {
      ...base,
      slug: 'pikachu',
      name: 'Pikachu',
      name_zh: '皮卡丘',
      name_status: 'source_provided',
      national_dex: 25,
    },
  ],
  forms: [
    {
      ...base,
      slug: 'pikachu',
      name: 'Pikachu',
      name_zh: null,
      name_status: 'unknown',
      species_id: 1,
    },
  ],
  games: [
    {
      ...base,
      slug: 'red',
      name: 'Red',
      name_zh: '红',
      name_status: 'source_provided',
      kind: 'mainline',
      collection_system: 'pokemon',
    },
  ],
}
export const hunt: Hunt = {
  ...base,
  species_id: 1,
  form_id: 1,
  game_id: null,
  method: '野外遭遇',
  start_date: '2024',
  end_date: null,
  encounter_count: 0,
  count_unit: 'encounters',
  duration_seconds: null,
  status: 'in_progress',
  notes: '保留备注',
}
export const shiny: ShinyRecord = {
  ...base,
  hunt_id: 1,
  current_species_id: 1,
  current_form_id: null,
  original_species_id: null,
  original_form_id: null,
  original_game_id: null,
  current_game_id: null,
  current_location: null,
  obtained_date: null,
  source: 'unknown',
  holding_status: 'unknown',
  nickname: null,
  gender: null,
  nature: null,
  ability: null,
  ball: null,
  level: null,
  notes: null,
}
