import { spawn, type ChildProcess } from 'node:child_process'
import { createServer as createTCPServer } from 'node:net'
import { resolve, join } from 'node:path'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { createServer, preview, type ViteDevServer } from 'vite'
import { parsePlan } from '../../src/api/planner'
import { ApiClient } from '../../src/api/client'
import { TrackerAPI, huntInput, shinyInput } from '../../src/api/tracker'

let databaseDir: string
let backend: ChildProcess | undefined
let frontend: ViteDevServer | undefined
let port: number
let origin: string
let client: ApiClient

async function freePort(): Promise<number> {
  const server = createTCPServer()
  await new Promise<void>((done, fail) => {
    server.once('error', fail)
    server.listen(0, '127.0.0.1', done)
  })
  const address = server.address()
  if (!address || typeof address === 'string') throw new Error('No TCP address')
  await new Promise<void>((done, fail) => server.close((error) => (error ? fail(error) : done())))
  return address.port
}

async function startBackend() {
  backend = spawn(resolve('../bin/shiny-tracker'), [], {
    env: {
      ...process.env,
      SHINY_HTTP_PORT: String(port),
      SHINY_DB_PATH: join(databaseDir, 'test.sqlite3'),
    },
    stdio: 'ignore',
  })
  let launchError: Error | undefined
  backend.once('error', (error) => {
    launchError = error
  })
  for (let attempt = 0; attempt < 100; attempt++) {
    if (launchError) throw launchError
    if (backend.exitCode !== null) throw new Error('Backend exited before becoming ready')
    try {
      const response = await fetch(`http://127.0.0.1:${port}/api/v1/health`, {
        signal: AbortSignal.timeout(200),
      })
      if (response.status === 200 && (await response.json()).data.status === 'ok') return
    } catch {
      /* Retry only while this freshly spawned process is starting. */
    }
    await new Promise((done) => setTimeout(done, 30))
  }
  throw new Error('Backend did not become ready')
}

async function stopBackend() {
  if (!backend || backend.exitCode !== null) return
  const current = backend
  await new Promise<void>((done, fail) => {
    const timer = setTimeout(() => {
      current.kill('SIGKILL')
      fail(new Error('Backend failed to stop'))
    }, 7_000)
    current.once('exit', (code) => {
      clearTimeout(timer)
      code === 0 ? done() : fail(new Error(`Backend exit ${code}`))
    })
    current.kill('SIGTERM')
  })
}

beforeAll(async () => {
  databaseDir = await mkdtemp(join(tmpdir(), 'pst-integration-'))
  port = await freePort()
  await startBackend()
  frontend = await createServer({
    server: {
      host: '127.0.0.1',
      port: 0,
      proxy: { '/api/v1': { target: `http://127.0.0.1:${port}` } },
    },
  })
  await frontend.listen()
  const address = frontend.httpServer?.address()
  if (!address || typeof address === 'string') throw new Error('No frontend address')
  origin = `http://127.0.0.1:${address.port}`
  client = new ApiClient(`${origin}/api/v1`)
})

afterAll(async () => {
  try {
    await frontend?.close()
    await stopBackend()
  } finally {
    if (databaseDir) await rm(databaseDir, { recursive: true, force: true })
  }
})

// Vitest runs this file's tests serially by default; restart is the final case.
describe('real Go backend through Vite and the actual API client', () => {
  it('returns validated health through the frontend origin', async () => {
    await expect(client.getHealth()).resolves.toEqual({ status: 'ok' })
  })
  it('preserves backend JSON 404 through the proxy', async () => {
    await expect(client.getJSON('/missing')).rejects.toMatchObject({
      kind: 'http',
      status: 404,
      code: 'not_found',
    })
  })
  it('preserves backend method errors through the proxy', async () => {
    const response = await fetch(`${origin}/api/v1/health`, { method: 'POST' })
    expect(response.status).toBe(405)
    expect(response.headers.get('Allow')).toBe('GET')
    expect(await response.json()).toMatchObject({ error: { code: 'method_not_allowed' } })
  })
  it('connects the built frontend preview to the same real backend', async () => {
    const built = await preview({
      preview: {
        host: '127.0.0.1',
        port: 0,
        proxy: { '/api/v1': { target: `http://127.0.0.1:${port}` } },
      },
    })
    try {
      const address = built.httpServer.address()
      if (!address || typeof address === 'string') throw new Error('No preview address')
      await expect(
        new ApiClient(`http://127.0.0.1:${address.port}/api/v1`).getHealth(),
      ).resolves.toEqual({ status: 'ok' })
    } finally {
      await new Promise<void>((done, fail) =>
        built.httpServer.close((error) => (error ? fail(error) : done())),
      )
    }
  })
  it('creates and edits a real Hunt with null/zero/partial date, then completes with an associated result', async () => {
    const records = new TrackerAPI(client),
      refs = await records.catalog()
    const input = {
      species_id: refs.species[0]!.id,
      form_id: refs.forms.find((f) => f.species_id === refs.species[0]!.id)!.id,
      game_id: null,
      method: 'integration',
      start_date: '2024',
      end_date: null,
      encounter_count: 0,
      count_unit: 'eggs',
      duration_seconds: null,
      status: 'in_progress' as const,
      notes: 'T010 real integration',
    }
    const created = await records.createHunt(input)
    expect(created).toMatchObject(input)
    const edited = await records.updateHunt(
      created.id,
      { ...huntInput(created), encounter_count: 12, notes: 'edited' },
      created.updated_at,
    )
    expect(edited).toMatchObject({ ...input, encounter_count: 12, notes: 'edited' })
    await expect(
      records.updateHunt(
        edited.id,
        { ...huntInput(edited), status: 'completed' },
        edited.updated_at,
      ),
    ).rejects.toMatchObject({ status: 409, code: 'conflict' })
    await client.postJSON('/shiny-records', {
      current_species_id: input.species_id,
      hunt_id: edited.id,
    })
    expect((await records.results(edited.id)).meta.total).toBe(1)
    const completed = await records.updateHunt(
      edited.id,
      { ...huntInput(edited), status: 'completed' },
      edited.updated_at,
    )
    expect(completed.status).toBe('completed')
    expect((await records.hunts('q=edited')).data.some((h) => h.id === completed.id)).toBe(true)
  })
  it('preserves real validation errors as safe machine codes', async () => {
    await expect(new TrackerAPI(client).hunts('page=0')).rejects.toMatchObject({
      status: 400,
      code: 'invalid_request',
    })
  })
  it('registers independent and linked individuals, preserves original facts and records real history', async () => {
    const records = new TrackerAPI(client),
      refs = await records.catalog()
    const first = refs.species[0]!.id,
      second = refs.species[1]!.id
    const independent = await records.createShiny({
      hunt_id: null,
      original_species_id: null,
      original_form_id: null,
      current_species_id: first,
      current_form_id: null,
      original_game_id: null,
      current_game_id: null,
      current_location: null,
      obtained_date: '2018',
      source: 'trade',
      holding_status: 'held',
      nickname: 'T011 integration',
      gender: null,
      nature: '胆小',
      ability: null,
      ball: '精灵球',
      level: 50,
      notes: 'keep',
    })
    const changed = await records.updateShiny(
      independent.id,
      {
        ...shinyInput(independent),
        current_species_id: second,
        current_game_id: refs.games[1]!.id,
        current_location: 'Box 2',
        holding_status: 'traded',
        change_type: 'evolution',
      },
      independent.updated_at,
    )
    expect(changed).toMatchObject({
      id: independent.id,
      original_species_id: null,
      original_game_id: null,
      hunt_id: null,
      ball: '精灵球',
      nature: '胆小',
      level: 50,
    })
    expect((await records.shinies('q=T011+integration')).meta.total).toBe(1)
    expect((await records.history(changed.id)).data.map((h) => h.kind)).toEqual(
      expect.arrayContaining(['registered', 'evolution', 'holding_change', 'location_change']),
    )
    const h = (await records.hunts()).data[0]!
    const linked = await records.createShiny({
      ...shinyInput(independent),
      hunt_id: h.id,
      nickname: 'T011 linked',
    })
    expect(linked.hunt_id).toBe(h.id)
    await expect(
      records.updateShiny(linked.id, { ...shinyInput(linked), level: 101 }, linked.updated_at),
    ).rejects.toMatchObject({ status: 400, code: 'invalid_request' })
  })
  it('preserves a real reviewed Random Hunt snapshot through save and idempotent Hunt conversion', async () => {
    const games = await new TrackerAPI(client).games()
    const ids = games.filter((g) => ['scarlet', 'violet'].includes(g.slug)).map((g) => g.id)
    const input = { request_id: 'integration-random-001', rules: { game_ids: ids, count: 2 } }
    const raw = (await client.postJSON('/planner/random', input)) as { data: unknown }
    const draw = parsePlan(raw.data)
    expect(draw.targets).toHaveLength(2)
    expect(draw.coverage.status).toBe('partial')
    expect(new Set(draw.targets.map((t) => t.target_key)).size).toBe(2)
    const saved = parsePlan(
      ((await client.postJSON(`/planner/plans/${draw.id}/save`, {})) as { data: unknown }).data,
    )
    expect(saved.status).toBe('active')
    const route = `/planner/plans/${draw.id}/targets/${draw.targets[0]!.id}/hunt`
    const a = parsePlan(((await client.postJSON(route, {})) as { data: unknown }).data),
      b = parsePlan(((await client.postJSON(route, {})) as { data: unknown }).data)
    expect(a.targets[0]!.hunt_id).toBe(b.targets[0]!.hunt_id)
    expect(a.targets[0]!.snapshot.choice.evidence[0]!.fact.methods[0]!.encounter_rate).toBeNull()
  })
  it('round trips a real persisted Bingo card, reviews and a unique reroll', async () => {
    const gs = await new TrackerAPI(client).games()
    const ids = gs.filter((g) => ['scarlet', 'violet'].includes(g.slug)).map((g) => g.id)
    const created = parsePlan(
      (
        (await client.postJSON('/planner/bingo', {
          request_id: 'integration-bingo-001',
          rules: { game_ids: ids },
          size: 3,
          calendar_zone: 'UTC',
        })) as { data: unknown }
      ).data,
    )
    expect(created.kind).toBe('bingo')
    expect(created.targets).toHaveLength(9)
    expect(created.bingo?.lines).toEqual([])
    const target = created.targets[8]!
    const changed = parsePlan(
      (
        (await client.postJSON(`/planner/plans/${created.id}/targets/${target.id}/reroll`, {
          request_id: 'integration-reroll-001',
          expected_target_key: target.target_key,
        })) as { data: unknown }
      ).data,
    )
    expect(changed.targets[8]!.target_key).not.toBe(target.target_key)
    expect(new Set(changed.targets.map((t) => t.target_key)).size).toBe(9)
    const refreshed = parsePlan(
      ((await client.postJSON(`/planner/plans/${created.id}/refresh`, {})) as { data: unknown })
        .data,
    )
    expect(refreshed.bingo?.events.some((e) => e.kind === 'rerolled')).toBe(true)
  })
  it('reports backend shutdown as an error, then succeeds after restart', async () => {
    await stopBackend()
    await expect(client.getHealth()).rejects.toMatchObject({ kind: 'http', status: 502 })
    await startBackend()
    await expect(client.getHealth()).resolves.toEqual({ status: 'ok' })
  })
})
