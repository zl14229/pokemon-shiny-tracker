import { ApiClient, ApiError, type RequestOptions } from './client'
export interface PlannerRules {
  game_ids: number[]
  dimension: 'species' | 'form'
  pool: 'all_missing' | 'selected_missing'
  count: number
  exclude_active: boolean
}
export interface Evidence {
  source: string
  url: string
  version: string
  scope: string
  checked_at: string
}
export interface Rate {
  numerator: number
  denominator: number
  conditions: string
  evidence: Evidence
}
export interface HuntMethod {
  name: string
  location: string
  conditions: string
  evidence: Evidence
  encounter_rate: Rate | null
  shiny_rate: Rate | null
}
export interface HuntFact {
  id: number
  game_id: number
  species_id: number
  form_id: number
  batch_version: string
  fact: {
    verification: 'verified'
    availability: 'confirmed_available'
    direct_hunt: 'yes'
    guide_status: string
    evidence: Evidence
    cross_checks: Evidence[]
    methods: HuntMethod[]
  }
}
export interface Choice {
  game_id: number
  game_name: string
  evidence: HuntFact[]
}
export interface Candidate {
  key: string
  species_id: number
  form_id: number | null
  name: string
  name_status: string
  games: Choice[]
}
export interface Coverage {
  status: 'partial'
  note: string
  games: {
    game_id: number
    name: string
    reviewed_entries: number
    verified_direct: number
    available_targets: number
  }[]
  versions: string[]
  verified_targets: number
  excluded_obtained: number
  excluded_active: number
  available: number
  requested: number
  actual: number
  shortage: boolean
}
export interface PlanTarget {
  id: number
  plan_id: number
  position: number
  target_key: string
  species_id: number
  form_id: number | null
  game_id: number
  status: 'pending' | 'started' | 'completed'
  hunt_id: number | null
  shiny_record_id: number | null
  snapshot: Candidate & { choice: Choice }
}
export interface BingoReview {
  record_id: number
  code: string
  reason: string
  needs_review: boolean
}
export interface BingoProof {
  method: 'record_date' | 'manual_timestamp'
  record_id: number
  obtained_date: string | null
  obtained_at: string | null
  evidence_note: string
  confirmed_at: string
  original_species_id: number
  original_form_id: number
  original_game_id: number
  source: 'self_hunted'
  availability: HuntFact
}
export interface BingoCell {
  target_id: number
  kind: string
  reviews: BingoReview[]
  proof?: BingoProof
}
export interface BingoDetails {
  size: 3 | 5
  started_at: string
  calendar_zone: string
  lines: number[][]
  cells: BingoCell[]
  events: {
    id: number
    plan_id: number
    target_id: number | null
    kind: string
    summary: string
    created_at: string
  }[]
}
export interface Plan {
  id: number
  kind: 'random' | 'bingo'
  status: 'draft' | 'active' | 'completed' | 'abandoned'
  created_at: string
  updated_at: string
  rules: PlannerRules
  coverage: Coverage
  targets: PlanTarget[]
  bingo?: BingoDetails
}
export interface PlanListItem {
  id: number
  kind: string
  status: string
  created_at: string
  target_count: number
}
function bad(): never {
  throw new ApiError('invalid_response', '计划响应不完整，请刷新后重试。')
}
function object(v: unknown): asserts v is Record<string, unknown> {
  if (!v || typeof v !== 'object' || Array.isArray(v)) bad()
}
const integer = (v: unknown, min = 0): v is number =>
  Number.isSafeInteger(v) && (v as number) >= min
function strings(v: unknown): asserts v is string[] {
  if (!Array.isArray(v) || !v.every((x) => typeof x === 'string')) bad()
}
function evidence(v: unknown) {
  object(v)
  if (
    !['source', 'version', 'scope'].every((k) => typeof v[k] === 'string' && v[k] !== '') ||
    typeof v.checked_at !== 'string' ||
    !Number.isFinite(Date.parse(v.checked_at)) ||
    typeof v.url !== 'string'
  )
    bad()
  try {
    const u = new URL(v.url)
    if (!['http:', 'https:'].includes(u.protocol) || u.username || u.password) bad()
  } catch {
    bad()
  }
}
function rate(v: unknown) {
  if (v === null) return
  object(v)
  if (
    !integer(v.numerator) ||
    !integer(v.denominator, 1) ||
    (v.numerator as number) > (v.denominator as number) ||
    typeof v.conditions !== 'string' ||
    !v.conditions
  )
    bad()
  evidence(v.evidence)
}
function choice(v: unknown) {
  object(v)
  if (
    !integer(v.game_id, 1) ||
    typeof v.game_name !== 'string' ||
    !Array.isArray(v.evidence) ||
    v.evidence.length === 0
  )
    bad()
  for (const row of v.evidence) {
    object(row)
    if (
      !integer(row.id, 1) ||
      row.game_id !== v.game_id ||
      !integer(row.species_id, 1) ||
      !integer(row.form_id, 1) ||
      typeof row.batch_version !== 'string'
    )
      bad()
    object(row.fact)
    const f = row.fact
    if (
      f.verification !== 'verified' ||
      f.availability !== 'confirmed_available' ||
      f.direct_hunt !== 'yes' ||
      !['partial', 'complete', 'unknown'].includes(String(f.guide_status))
    )
      bad()
    evidence(f.evidence)
    if (!Array.isArray(f.cross_checks) || !Array.isArray(f.methods)) bad()
    f.cross_checks.forEach(evidence)
    for (const m of f.methods) {
      object(m)
      if (!['name', 'location', 'conditions'].every((k) => typeof m[k] === 'string')) bad()
      evidence(m.evidence)
      rate(m.encounter_rate)
      rate(m.shiny_rate)
    }
  }
}
function candidate(v: unknown) {
  object(v)
  if (
    typeof v.key !== 'string' ||
    !integer(v.species_id, 1) ||
    (v.form_id !== null && !integer(v.form_id, 1)) ||
    typeof v.name !== 'string' ||
    typeof v.name_status !== 'string' ||
    !Array.isArray(v.games) ||
    v.games.length === 0
  )
    bad()
  v.games.forEach(choice)
  if (new Set(v.games.map((g) => (g as Choice).game_id)).size !== v.games.length) bad()
  for (const g of v.games as Choice[]) {
    if (
      g.evidence.some(
        (r) => r.species_id !== v.species_id || (v.form_id !== null && r.form_id !== v.form_id),
      )
    )
      bad()
  }
}
function coverage(v: unknown): asserts v is Coverage {
  object(v)
  if (
    v.status !== 'partial' ||
    typeof v.note !== 'string' ||
    typeof v.shortage !== 'boolean' ||
    !Array.isArray(v.games)
  )
    bad()
  strings(v.versions)
  for (const k of [
    'verified_targets',
    'excluded_obtained',
    'excluded_active',
    'available',
    'requested',
    'actual',
  ]) {
    if (!integer(v[k])) bad()
  }
  const c = v as unknown as Coverage
  if (
    c.verified_targets !== c.excluded_obtained + c.excluded_active + c.available ||
    c.actual > c.available ||
    c.actual > c.requested ||
    c.shortage !== c.actual < c.requested
  )
    bad()
  for (const g of c.games) {
    object(g)
    if (
      !integer(g.game_id, 1) ||
      typeof g.name !== 'string' ||
      !integer(g.reviewed_entries) ||
      !integer(g.verified_direct) ||
      !integer(g.available_targets) ||
      g.verified_direct > g.reviewed_entries
    )
      bad()
  }
}
function rules(v: unknown) {
  object(v)
  if (
    !Array.isArray(v.game_ids) ||
    !v.game_ids.length ||
    v.game_ids.some((x) => !integer(x, 1)) ||
    new Set(v.game_ids).size !== v.game_ids.length ||
    !['species', 'form'].includes(String(v.dimension)) ||
    !['all_missing', 'selected_missing'].includes(String(v.pool)) ||
    !integer(v.count, 1) ||
    (v.count as number) > 100 ||
    typeof v.exclude_active !== 'boolean'
  )
    bad()
}
function bingo(v: Record<string, unknown>) {
  const b = v.bingo
  object(b)
  if (
    (b.size !== 3 && b.size !== 5) ||
    typeof b.started_at !== 'string' ||
    !Number.isFinite(Date.parse(b.started_at)) ||
    typeof b.calendar_zone !== 'string' ||
    !Array.isArray(b.lines) ||
    !Array.isArray(b.cells) ||
    !Array.isArray(b.events)
  )
    bad()
  const targets = v.targets as PlanTarget[]
  if (targets.length !== b.size * b.size || b.cells.length !== targets.length) bad()
  const complete = new Set(targets.filter((t) => t.status === 'completed').map((t) => t.position)),
    expected: string[] = []
  const line = (positions: number[]) => {
    if (positions.every((p) => complete.has(p))) expected.push(positions.join(','))
  }
  for (let i = 0; i < b.size; i++) {
    line(Array.from({ length: b.size }, (_, j) => i * (b.size as number) + j + 1))
    line(Array.from({ length: b.size }, (_, j) => j * (b.size as number) + i + 1))
  }
  line(Array.from({ length: b.size }, (_, i) => i * (b.size as number) + i + 1))
  line(Array.from({ length: b.size }, (_, i) => i * (b.size as number) + (b.size as number) - i))
  const actual: string[] = []
  for (const l of b.lines) {
    if (
      !Array.isArray(l) ||
      l.length !== b.size ||
      !l.every((x) => integer(x, 1) && x <= targets.length)
    )
      bad()
    actual.push(l.join(','))
  }
  if (JSON.stringify(actual.sort()) !== JSON.stringify(expected.sort())) bad()
  const seen = new Set<number>()
  for (const c of b.cells) {
    object(c)
    if (
      !integer(c.target_id, 1) ||
      seen.has(c.target_id) ||
      !['none', 'excluded', 'needs_review', 'completed'].includes(String(c.kind)) ||
      !Array.isArray(c.reviews)
    )
      bad()
    seen.add(c.target_id)
    const target = targets.find((t) => t.id === c.target_id)
    if (!target) bad()
    for (const r of c.reviews) {
      object(r)
      if (
        !integer(r.record_id, 1) ||
        typeof r.code !== 'string' ||
        typeof r.reason !== 'string' ||
        typeof r.needs_review !== 'boolean'
      )
        bad()
    }
    if (target.status === 'completed') {
      const p = c.proof
      object(p)
      if (
        c.kind !== 'completed' ||
        p.record_id !== target.shiny_record_id ||
        p.original_species_id !== target.species_id ||
        p.original_game_id !== target.game_id ||
        !integer(p.original_form_id, 1) ||
        (target.form_id !== null && p.original_form_id !== target.form_id) ||
        !['record_date', 'manual_timestamp'].includes(String(p.method)) ||
        p.source !== 'self_hunted' ||
        typeof p.confirmed_at !== 'string' ||
        !Number.isFinite(Date.parse(p.confirmed_at)) ||
        typeof p.evidence_note !== 'string'
      )
        bad()
      if (
        p.method === 'manual_timestamp' &&
        (typeof p.obtained_at !== 'string' ||
          !Number.isFinite(Date.parse(p.obtained_at)) ||
          !p.evidence_note)
      )
        bad()
      choice({ game_id: p.original_game_id, game_name: 'evidence', evidence: [p.availability] })
      object(p.availability)
      if (
        p.availability.form_id !== p.original_form_id ||
        p.availability.species_id !== p.original_species_id
      )
        bad()
    } else if (c.kind === 'completed') bad()
  }
  for (const e of b.events) {
    object(e)
    if (
      !integer(e.id, 1) ||
      e.plan_id !== v.id ||
      typeof e.kind !== 'string' ||
      typeof e.summary !== 'string' ||
      typeof e.created_at !== 'string' ||
      !Number.isFinite(Date.parse(e.created_at)) ||
      (e.target_id !== null && !targets.some((t) => t.id === e.target_id))
    )
      bad()
  }
}
export function parsePlan(v: unknown): Plan {
  object(v)
  if (
    !integer(v.id, 1) ||
    !['random', 'bingo'].includes(String(v.kind)) ||
    !['draft', 'active', 'completed', 'abandoned'].includes(String(v.status)) ||
    typeof v.created_at !== 'string' ||
    !Number.isFinite(Date.parse(v.created_at)) ||
    typeof v.updated_at !== 'string' ||
    !Array.isArray(v.targets)
  )
    bad()
  rules(v.rules)
  coverage(v.coverage)
  if (
    v.targets.length !== v.coverage.actual ||
    (v.rules as PlannerRules).count !== v.coverage.requested
  )
    bad()
  const keys = new Set(),
    ids = new Set()
  for (const t of v.targets) {
    object(t)
    if (
      !integer(t.id, 1) ||
      ids.has(t.id) ||
      t.plan_id !== v.id ||
      !integer(t.position, 1) ||
      !['pending', 'started', 'completed'].includes(String(t.status)) ||
      (t.hunt_id !== null && !integer(t.hunt_id, 1)) ||
      (t.shiny_record_id !== null && !integer(t.shiny_record_id, 1))
    )
      bad()
    candidate(t.snapshot)
    object(t.snapshot)
    choice(t.snapshot.choice)
    object(t.snapshot.choice)
    if (
      t.target_key !==
        `${(v.rules as PlannerRules).dimension}:${(v.rules as PlannerRules).dimension === 'form' ? t.form_id : t.species_id}` ||
      !(v.rules as PlannerRules).game_ids.includes(t.game_id as number) ||
      (t.status === 'pending' && (t.hunt_id !== null || t.shiny_record_id !== null)) ||
      (t.status === 'started' && t.hunt_id === null) ||
      t.game_id !== t.snapshot.choice.game_id ||
      t.target_key !== t.snapshot.key ||
      t.species_id !== t.snapshot.species_id ||
      t.form_id !== t.snapshot.form_id ||
      keys.has(t.target_key) ||
      !(t.snapshot.games as Choice[]).some((g) => g.game_id === t.game_id) ||
      !v.targets.every((x, i) => (x as PlanTarget).position === i + 1)
    )
      bad()
    keys.add(t.target_key)
    ids.add(t.id)
  }
  if (v.kind === 'bingo') bingo(v)
  return v as unknown as Plan
}
const client = new ApiClient()
const unwrap = (v: unknown) => {
  object(v)
  return v.data
}
export const planner = {
  async bingo(input: {
    request_id: string
    rules: PlannerRules
    size: 3 | 5
    calendar_zone: string
  }) {
    return parsePlan(unwrap(await client.postJSON('/planner/bingo', input, { timeoutMs: 30000 })))
  },
  async refresh(id: number) {
    return parsePlan(
      unwrap(await client.postJSON(`/planner/plans/${id}/refresh`, {}, { timeoutMs: 30000 })),
    )
  },
  async reroll(
    id: number,
    target: number,
    input: { request_id: string; expected_target_key: string },
  ) {
    return parsePlan(
      unwrap(
        await client.postJSON(`/planner/plans/${id}/targets/${target}/reroll`, input, {
          timeoutMs: 30000,
        }),
      ),
    )
  },
  async verifyDate(
    id: number,
    target: number,
    input: { record_id: number; obtained_at: string; evidence_note: string; confirmed: boolean },
  ) {
    return parsePlan(
      unwrap(
        await client.postJSON(`/planner/plans/${id}/targets/${target}/verify-date`, input, {
          timeoutMs: 30000,
        }),
      ),
    )
  },
  async close(id: number, status: 'completed' | 'abandoned') {
    return parsePlan(
      unwrap(await client.postJSON(`/planner/plans/${id}/close`, { status }, { timeoutMs: 30000 })),
    )
  },
  async pool(r: PlannerRules, options: RequestOptions = {}) {
    const v = unwrap(
      await client.postJSON('/planner/pool', { rules: r }, { timeoutMs: 30000, ...options }),
    )
    object(v)
    coverage(v.coverage)
    rules(v.rules)
    if (!Array.isArray(v.candidates) || v.candidates.length !== v.coverage.available) bad()
    v.candidates.forEach(candidate)
    return v as unknown as {
      rules: PlannerRules
      coverage: Coverage
      candidates: Candidate[]
    }
  },
  async draw(input: { request_id: string; rules: PlannerRules }, options: RequestOptions = {}) {
    return parsePlan(
      unwrap(
        await client.postJSON('/planner/random', input, {
          timeoutMs: 30000,
          ...options,
        }),
      ),
    )
  },
  async get(id: number, options: RequestOptions = {}) {
    return parsePlan(
      unwrap(
        await client.getJSON(`/planner/plans/${id}`, {
          timeoutMs: 30000,
          ...options,
        }),
      ),
    )
  },
  async save(id: number) {
    return parsePlan(unwrap(await client.postJSON(`/planner/plans/${id}/save`, {})))
  },
  async start(id: number, target: number, huntId?: number, expectedKey?: string) {
    return parsePlan(
      unwrap(
        await client.postJSON(
          `/planner/plans/${id}/targets/${target}/hunt`,
          {
            ...(huntId ? { hunt_id: huntId } : {}),
            ...(expectedKey ? { expected_target_key: expectedKey } : {}),
          },
          { timeoutMs: 30000 },
        ),
      ),
    )
  },
  async list(offset = 0, options: RequestOptions = {}) {
    const v = await client.getJSON(`/planner/plans?limit=20&offset=${offset}`, options)
    object(v)
    object(v.meta)
    if (!Array.isArray(v.data) || !integer(v.meta.total)) bad()
    for (const p of v.data) {
      object(p)
      if (
        !integer(p.id, 1) ||
        typeof p.kind !== 'string' ||
        typeof p.status !== 'string' ||
        typeof p.created_at !== 'string' ||
        !integer(p.target_count)
      )
        bad()
    }
    return v as unknown as {
      data: PlanListItem[]
      meta: { total: number; limit: number; offset: number }
    }
  },
}
