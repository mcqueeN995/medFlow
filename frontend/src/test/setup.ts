import '@testing-library/jest-dom/vitest'
import 'fake-indexeddb/auto'
import { afterAll, afterEach, beforeAll } from 'vitest'
import { cleanup } from '@testing-library/react'
import { setupServer } from 'msw/node'
import { authHandlers } from '@/mocks/handlers/auth'
import { libraryHandlers } from '@/mocks/handlers/library'
import { uploadHandlers } from '@/mocks/handlers/upload'
import { navigatorHandlers } from '@/mocks/handlers/navigator'
import { cardsHandlers } from '@/mocks/handlers/cards'
import { forumHandlers } from '@/mocks/handlers/forum'
import { pushHandlers } from '@/mocks/handlers/push'

// Те же стейтфул-хендлеры, что src/mocks/browser.ts подключает в dev-режиме
// (VITE_USE_MOCKS=true) - переиспользуем их и в тестах через msw/node вместо
// дублирования логики моков под тесты отдельно.
export const server = setupServer(
  ...authHandlers,
  ...libraryHandlers,
  ...uploadHandlers,
  ...navigatorHandlers,
  ...cardsHandlers,
  ...forumHandlers,
  ...pushHandlers,
)

// 'error', не 'bypass' - неожиданный незамоканный запрос должен явно ронять
// тест, а не молча падать в сеть/зависать (в jsdom нет реальной сети).
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => {
  server.resetHandlers()
  cleanup()
})
afterAll(() => server.close())
