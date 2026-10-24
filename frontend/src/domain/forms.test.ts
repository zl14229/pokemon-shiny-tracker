import { describe, expect, it } from 'vitest'
import { dateBounds, huntDraft, validateHunt } from './forms'
import { hunt } from '../test/fixtures'
import { duration, referenceName } from './format'

describe('Hunt data boundaries', () => {
  it('keeps missing values distinct from known zero and retains partial dates', () => {
    const draft = huntDraft(hunt),
      result = validateHunt(draft)
    expect(draft.encounter_count).toBe('0')
    expect(result.input).toMatchObject({
      encounter_count: 0,
      duration_seconds: null,
      start_date: '2024',
      form_id: 1,
      game_id: null,
      notes: '保留备注',
    })
    const unknown = huntDraft()
    unknown.species_id = 1
    expect(validateHunt(unknown).input).toMatchObject({
      encounter_count: null,
      count_unit: null,
      duration_seconds: null,
      start_date: null,
    })
    expect(duration(0)).toBe('0秒')
    expect(duration(null)).toBe('未记录')
    expect(referenceName([], null)).toBe('未记录')
  })
  it.each(['-1', '1.5', '1e3', '9007199254740992'])(
    'rejects invalid or unsafe count %s',
    (value) => {
      const draft = huntDraft(hunt)
      draft.encounter_count = value
      expect(validateHunt(draft).errors.encounter_count).toBeTruthy()
    },
  )
  it('requires a unit for zero and checks date range without inventing precision', () => {
    const draft = huntDraft(hunt)
    draft.count_unit = ''
    expect(validateHunt(draft).errors.count_unit).toBeTruthy()
    draft.count_unit = 'eggs'
    draft.start_date = '2024'
    draft.end_date = '2024-01'
    expect(validateHunt(draft).errors).toEqual({})
    draft.start_date = '2025'
    expect(validateHunt(draft).errors.end_date).toBeTruthy()
    expect(dateBounds('2024-02-29')).not.toBeNull()
    expect(dateBounds('2023-02-29')).toBeNull()
    expect(dateBounds('0001')).not.toBeNull()
    expect(dateBounds('0000')).toBeNull()
    expect(dateBounds('2024-13')).toBeNull()
  })
})
