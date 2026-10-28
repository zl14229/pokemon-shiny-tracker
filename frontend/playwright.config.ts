import { defineConfig } from '@playwright/test'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const databaseDir = mkdtempSync(join(tmpdir(), 'pst-browser-'))
process.on('exit', () => rmSync(databaseDir, { recursive: true, force: true }))

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  use: { baseURL: 'http://127.0.0.1:15173', browserName: 'chromium' },
  webServer: [
    {
      command: '../bin/shiny-tracker',
      env: { SHINY_HTTP_PORT: '15080', SHINY_DB_PATH: join(databaseDir, 'test.sqlite3') },
      url: 'http://127.0.0.1:15080/api/v1/health',
      reuseExistingServer: false,
      timeout: 30_000,
      gracefulShutdown: { signal: 'SIGTERM', timeout: 7_000 },
    },
    {
      command: 'npm run dev -- --port 15173',
      env: { SHINY_API_PORT: '15080' },
      url: 'http://127.0.0.1:15173',
      reuseExistingServer: false,
      timeout: 30_000,
    },
  ],
})
