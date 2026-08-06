import path from 'node:path'
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: false,
    // По умолчанию vitest подхватывает и *.spec.ts - без этого исключения
    // ловит заодно Playwright-спеки из e2e/ (другой раннер, `test`/`expect`
    // из @playwright/test, а не vitest).
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
  },
})
