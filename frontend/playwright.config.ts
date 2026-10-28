import { defineConfig } from '@playwright/test'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const databaseDir = mkdtempSync(join(tmpdir(), 'pst-browser-'))
process.on('exit', () => rmSync(databaseDir, { recursive: true, force: true }))
function testPort(value: string | undefined, fallback: number) {
  const port = value === undefined ? fallback : Number(value)
  if (!Number.isInteger(port) || port < 1024 || port > 65535) throw new Error('Invalid E2E port')
  return port
}
const apiPort = testPort(process.env.SHINY_E2E_API_PORT, 15080)
const webPort = testPort(process.env.SHINY_E2E_WEB_PORT, 15173)
if (apiPort === webPort) throw new Error('E2E ports must differ')

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  use: { baseURL: `http://127.0.0.1:${webPort}`, browserName: 'chromium' },
  webServer: [
    {
      command: '../bin/shiny-tracker',
      env: { SHINY_HTTP_PORT: String(apiPort), SHINY_DB_PATH: join(databaseDir, 'test.sqlite3') },
      url: `http://127.0.0.1:${apiPort}/api/v1/health`,
      reuseExistingServer: false,
      timeout: 30_000,
      gracefulShutdown: { signal: 'SIGTERM', timeout: 7_000 },
    },
    {
      command: `npm run dev -- --port ${webPort}`,
      env: { SHINY_API_PORT: String(apiPort) },
      url: `http://127.0.0.1:${webPort}`,
      reuseExistingServer: false,
      timeout: 30_000,
    },
  ],
})
