import { describe, it, expect } from 'vitest'
import { parsePlan } from './planner'
import { planFixture } from '../test/planner-fixture'
describe('Planner response boundaries', () => {
  it('accepts a scoped snapshot with unknown guide and separate unknown rates', () => {
    const p = planFixture()
    expect(parsePlan(p)).toEqual(p)
  })
  it.each([
    'duplicate',
    'locked',
    'bad source',
    'bad game',
    'bad key',
    'bad summary',
    'bad order',
    'wrong form',
  ])('rejects %s', (which) => {
    const p = planFixture()
    const t = p.targets[0]!
    switch (which) {
      case 'duplicate':
        p.targets[1] = structuredClone(t)
        break
      case 'locked':
        Object.assign(t.snapshot.choice.evidence[0]!.fact, {
          availability: 'confirmed_locked',
        })
        break
      case 'bad source':
        t.snapshot.choice.evidence[0]!.fact.evidence.url = 'javascript:alert(1)'
        break
      case 'bad game':
        t.game_id = 2
        break
      case 'bad key':
        t.target_key = 'form:1'
        break
      case 'bad summary':
        p.coverage.actual = 0
        break
      case 'bad order':
        t.position = 2
        break
      case 'wrong form':
        t.snapshot.form_id = 99
        break
    }
    expect(() => parsePlan(p)).toThrow()
  })
})
