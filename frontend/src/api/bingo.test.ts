import { describe, it, expect } from 'vitest'
import { parsePlan } from './planner'
import { bingoFixture, completeFixtureCell } from '../test/bingo-fixture'
describe('Bingo evidence response boundaries', () => {
  it('accepts a full grid with explicit proofs and derived row', () => {
    const p = bingoFixture()
    for (const t of p.targets.slice(0, 3)) completeFixtureCell(p, t)
    p.bingo!.lines = [[1, 2, 3]]
    expect(parsePlan(p)).toEqual(p)
  })
  it.each([
    'false line',
    'missing cell',
    'wrong proof identity',
    'trade proof',
    'wrong proof form',
  ])('rejects %s', (which) => {
    const p = bingoFixture()
    switch (which) {
      case 'false line':
        p.bingo!.lines = [[1, 5, 9]]
        break
      case 'missing cell':
        p.bingo!.cells.pop()
        break
      default: {
        const t = p.targets[0]!
        completeFixtureCell(p, t)
        const proof = p.bingo!.cells[0]!.proof!
        if (which === 'wrong proof identity') proof.record_id = 999
        if (which === 'trade proof') Object.assign(proof, { source: 'trade' })
        if (which === 'wrong proof form') proof.original_form_id = 999
      }
    }
    expect(() => parsePlan(p)).toThrow()
  })
})
