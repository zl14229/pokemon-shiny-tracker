import { describe, expect, it } from 'vitest'
import { shinyDraft, validateShiny } from './shiny-form'
import { shiny } from '../test/fixtures'
describe('individual form semantics', () => {
  it('allows an independent historical record without inventing original fields or ownership', () => {
    const d = shinyDraft()
    d.current_species_id = 1
    const input = validateShiny(d).input
    expect(input).toMatchObject({
      hunt_id: null,
      original_species_id: null,
      original_form_id: null,
      original_game_id: null,
      current_game_id: null,
      obtained_date: null,
      source: 'unknown',
      holding_status: 'unknown',
      ball: null,
      level: null,
    })
  })
  it('associating Hunt does not infer species/game/date/source', () => {
    const d = shinyDraft()
    d.current_species_id = 2
    d.hunt_id = 1
    expect(validateShiny(d).input).toMatchObject({
      hunt_id: 1,
      current_species_id: 2,
      original_species_id: null,
      original_game_id: null,
      obtained_date: null,
      source: 'unknown',
    })
  })
  it('requires explicit evolution/correction while retaining unknown original species', () => {
    const d = shinyDraft(shiny)
    d.current_species_id = 2
    expect(validateShiny(d, shiny).errors.change_type).toBeTruthy()
    d.change_type = 'evolution'
    d.current_game_id = 2
    d.current_location = 'HOME'
    d.holding_status = 'traded'
    d.ball = '精灵球'
    expect(validateShiny(d, shiny).input).toMatchObject({
      change_type: 'evolution',
      original_species_id: null,
      current_species_id: 2,
      current_game_id: 2,
      hunt_id: 1,
      ball: '精灵球',
      holding_status: 'traded',
    })
    d.current_species_id = 1
    expect(validateShiny(d, shiny).input).not.toHaveProperty('change_type')
  })
  it.each(['0', '101', '1.5', '-1'])('rejects invalid level %s', (level) => {
    const d = shinyDraft(shiny)
    d.level = level
    expect(validateShiny(d, shiny).errors.level).toBeTruthy()
  })
  it('checks date precision and lengths without forcing complete dates', () => {
    const d = shinyDraft(shiny)
    d.obtained_date = '2018'
    d.event_date = '2024-02'
    expect(validateShiny(d, shiny).errors).toEqual({})
    d.obtained_date = '2023-02-29'
    d.nickname = '宝'.repeat(129)
    expect(validateShiny(d, shiny).errors).toHaveProperty('obtained_date')
    expect(validateShiny(d, shiny).errors).toHaveProperty('nickname')
  })
})
