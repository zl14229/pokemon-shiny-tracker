import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  use: { baseURL: 'http://127.0.0.1:45173', browserName: 'chromium' },
  webServer: {
    command: 'npm run dev -- --port 45173',
    url: 'http://127.0.0.1:45173',
    reuseExistingServer: false,
    timeout: 30_000,
  },
})
