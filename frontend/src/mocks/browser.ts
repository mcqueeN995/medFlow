import { setupWorker } from 'msw/browser'
import { getMedFlowAPIMock } from '@/api/generated/medFlowAPI.msw'
import { authHandlers } from './handlers/auth'
import { libraryHandlers } from './handlers/library'
import { uploadHandlers } from './handlers/upload'
import { navigatorHandlers } from './handlers/navigator'
import { cardsHandlers } from './handlers/cards'
import { forumHandlers } from './handlers/forum'
import { pushHandlers } from './handlers/push'

// Хендлеры с реалистичным поведением идут первыми — переопределяют
// автогенерированные заглушки там, где нужен настоящий сценарий (стейт,
// фильтрация, редиректы), а не случайные faker-данные.
export const worker = setupWorker(
  ...authHandlers,
  ...libraryHandlers,
  ...uploadHandlers,
  ...navigatorHandlers,
  ...cardsHandlers,
  ...forumHandlers,
  ...pushHandlers,
  ...getMedFlowAPIMock(),
)
