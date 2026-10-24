import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'node', include: ['tests/integration/**/*.test.ts'],
    hookTimeout: 20_000, testTimeout: 15_000, maxWorkers: 1,
  },
})
