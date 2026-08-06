import { defineConfig } from '@playwright/test'

// e2e гоняются против уже поднятого docker-compose стека (`make up`), не
// против dev-сервера - см. README. baseURL по умолчанию совпадает с портом
// Caddy (HTTP_PORT в .env), переопределяемым через E2E_BASE_URL.
export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  retries: 0,
  reporter: [
    ['list'],
    ['html', { outputFolder: 'playwright-report', open: 'never' }],
    ['json', { outputFile: 'e2e-results.json' }],
  ],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:8081',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  timeout: 60_000,
})
