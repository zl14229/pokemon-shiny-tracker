import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  use: { baseURL: 'http://127.0.0.1:45173', browserName: 'chromium' },
  webServer: [
    {
      command: '../bin/shiny-tracker',
      env: { SHINY_HTTP_PORT: '45080' },
      url: 'http://127.0.0.1:45080/api/v1/health',
      reuseExistingServer: false,
      timeout: 30_000,
      gracefulShutdown: { signal: 'SIGTERM', timeout: 7_000 },
    },
    {
      command: 'npm run dev -- --port 45173',
      env: { SHINY_API_PORT: '45080' },
      url: 'http://127.0.0.1:45173',
      reuseExistingServer: false,
      timeout: 30_000,
    },
  ],
})
