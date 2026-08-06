import path from 'node:path'
import { expect, test } from '@playwright/test'
import { cleanupTestUser, registerTestUser } from './fixtures/db'

let email: string

test.afterEach(async () => {
  if (email) await cleanupTestUser(email)
})

// Без настроенного LLM-провайдера (LLM_API_KEY/LLM_PROVIDER=ollama) конвейер
// детерминированно уходит в status=failed с error_message (см. README) - это
// такой же валидный terminal-исход, как done. Тест проверяет саму асинхронную
// механику очереди (задача создаётся → воркер её забирает → доходит до
// терминального статуса), а не качество генерации ИИ.
test('creates a card generation task and it reaches a terminal status', async ({ page, baseURL }) => {
  const user = await registerTestUser(baseURL!, 'e2e_cards')
  email = user.email

  await page.goto('/login')
  await page.getByPlaceholder('Логин/почта').fill(user.email)
  await page.getByPlaceholder('Пароль').fill(user.password)
  await page.locator('[data-slot="checkbox"]').click()
  await page.getByRole('button', { name: 'Войти' }).click()
  await page.waitForURL('**/library')

  await page.goto('/cards/create')
  await page.locator('input[type="file"]').setInputFiles(path.join(import.meta.dirname, 'fixtures', 'sample.pdf'))
  await expect(page.getByText('Файл загружен')).toBeVisible({ timeout: 15_000 })

  await page.locator('#topic').fill('Строение сердца (e2e)')
  await page.getByRole('button', { name: 'Сгенерировать карточки' }).click()

  await page.waitForURL('**/cards/tasks/*')

  // терминальный статус - Готово (сгенерированы карточки) или Ошибка (нет
  // рабочего LLM-провайдера в этом окружении) - оба варианта допустимы.
  await expect(page.getByText(/^(Готово|Ошибка)$/)).toBeVisible({ timeout: 45_000 })
})
