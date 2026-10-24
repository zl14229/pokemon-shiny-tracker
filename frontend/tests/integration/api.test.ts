import { spawn, type ChildProcess } from 'node:child_process'
import { createServer as createTCPServer } from 'node:net'
import { resolve, join } from 'node:path'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { createServer, preview, type ViteDevServer } from 'vite'
import { ApiClient } from '../../src/api/client'

let databaseDir: string
let backend: ChildProcess | undefined
let frontend: ViteDevServer | undefined
let port: number
let origin: string
let client: ApiClient

async function freePort(): Promise<number> {
  const server = createTCPServer()
  await new Promise<void>((done, fail) => { server.once('error', fail); server.listen(0, '127.0.0.1', done) })
  const address = server.address()
  if (!address || typeof address === 'string') throw new Error('No TCP address')
  await new Promise<void>((done, fail) => server.close((error) => error ? fail(error) : done()))
  return address.port
}

async function startBackend() {
  backend = spawn(resolve('../bin/shiny-tracker'), [], {
    env: { ...process.env, SHINY_HTTP_PORT: String(port), SHINY_DB_PATH: join(databaseDir, "test.sqlite3") }, stdio: 'ignore',
  })
  let launchError: Error | undefined
  backend.once('error', (error) => { launchError = error })
  for (let attempt = 0; attempt < 100; attempt++) {
    if (launchError) throw launchError
    if (backend.exitCode !== null) throw new Error('Backend exited before becoming ready')
    try {
      const response = await fetch(`http://127.0.0.1:${port}/api/v1/health`, { signal: AbortSignal.timeout(200) })
      if (response.status === 200 && (await response.json()).data.status === 'ok') return
    } catch { /* Retry only while this freshly spawned process is starting. */ }
    await new Promise((done) => setTimeout(done, 30))
  }
  throw new Error('Backend did not become ready')
}

async function stopBackend() {
  if (!backend || backend.exitCode !== null) return
  const current = backend
  await new Promise<void>((done, fail) => {
    const timer = setTimeout(() => { current.kill('SIGKILL'); fail(new Error('Backend failed to stop')) }, 7_000)
    current.once('exit', (code) => { clearTimeout(timer); code === 0 ? done() : fail(new Error(`Backend exit ${code}`)) })
    current.kill('SIGTERM')
  })
}

beforeAll(async () => {
  databaseDir = await mkdtemp(join(tmpdir(), "pst-integration-"))
  port = await freePort()
  await startBackend()
  frontend = await createServer({
    server: { host: '127.0.0.1', port: 0, proxy: { '/api/v1': { target: `http://127.0.0.1:${port}` } } },
  })
  await frontend.listen()
  const address = frontend.httpServer?.address()
  if (!address || typeof address === 'string') throw new Error('No frontend address')
  origin = `http://127.0.0.1:${address.port}`
  client = new ApiClient(`${origin}/api/v1`)
})

afterAll(async () => {
  try { await frontend?.close(); await stopBackend() }
  finally { if (databaseDir) await rm(databaseDir, { recursive: true, force: true }) }
})

// Vitest runs this file's tests serially by default; restart is the final case.
describe('real Go backend through Vite and the actual API client', () => {
  it('returns validated health through the frontend origin', async () => {
    await expect(client.getHealth()).resolves.toEqual({ status: 'ok' })
  })
  it('preserves backend JSON 404 through the proxy', async () => {
    await expect(client.getJSON('/missing')).rejects.toMatchObject({ kind: 'http', status: 404, code: 'not_found' })
  })
  it('preserves backend method errors through the proxy', async () => {
    const response = await fetch(`${origin}/api/v1/health`, { method: 'POST' })
    expect(response.status).toBe(405)
    expect(response.headers.get('Allow')).toBe('GET')
    expect(await response.json()).toMatchObject({ error: { code: 'method_not_allowed' } })
  })
  it('connects the built frontend preview to the same real backend', async () => {
    const built = await preview({
      preview: { host: '127.0.0.1', port: 0, proxy: { '/api/v1': { target: `http://127.0.0.1:${port}` } } },
    })
    try {
      const address = built.httpServer.address()
      if (!address || typeof address === 'string') throw new Error('No preview address')
      await expect(new ApiClient(`http://127.0.0.1:${address.port}/api/v1`).getHealth()).resolves.toEqual({ status: 'ok' })
    } finally {
      await new Promise<void>((done, fail) => built.httpServer.close((error) => error ? fail(error) : done()))
    }
  })
  it('reports backend shutdown as an error, then succeeds after restart', async () => {
    await stopBackend()
    await expect(client.getHealth()).rejects.toMatchObject({ kind: 'http', status: 502 })
    await startBackend()
    await expect(client.getHealth()).resolves.toEqual({ status: 'ok' })
  })
})
