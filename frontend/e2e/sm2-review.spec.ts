import { expect, test } from '@playwright/test'
import { cleanupTestUser, pool, registerTestUser, seedDueCard } from './fixtures/db'

let email: string

test.afterEach(async () => {
  if (email) await cleanupTestUser(email)
})

test('rates a due card and it disappears from the review batch (SM-2 progress persisted)', async ({ page, baseURL }) => {
  const user = await registerTestUser(baseURL!, 'e2e_sm2')
  email = user.email
  const { cardId } = await seedDueCard(user.id)

  await page.goto('/login')
  await page.getByPlaceholder('Логин/почта').fill(user.email)
  await page.getByPlaceholder('Пароль').fill(user.password)
  await page.locator('[data-slot="checkbox"]').click()
  await page.getByRole('button', { name: 'Войти' }).click()
  await page.waitForURL('**/library')

  await page.goto('/cards/review')
  await expect(page.getByText('e2e тестовый вопрос')).toBeVisible()

  await page.getByRole('button', { name: 'Показать ответ' }).click()
  await expect(page.getByText('e2e тестовый ответ')).toBeVisible()
  await page.getByRole('button', { name: 'Норм' }).click()

  await expect(page.getByText('Сессия завершена')).toBeVisible()

  const progress = await pool.query('SELECT last_grade, repetitions FROM card_progress WHERE card_id = $1', [cardId])
  expect(progress.rows[0].last_grade).toBe(2)
  expect(progress.rows[0].repetitions).toBe(1)
})

test('shows the empty state once nothing is due anymore', async ({ page, baseURL }) => {
  const user = await registerTestUser(baseURL!, 'e2e_sm2_empty')
  email = user.email

  await page.goto('/login')
  await page.getByPlaceholder('Логин/почта').fill(user.email)
  await page.getByPlaceholder('Пароль').fill(user.password)
  await page.locator('[data-slot="checkbox"]').click()
  await page.getByRole('button', { name: 'Войти' }).click()
  await page.waitForURL('**/library')

  await page.goto('/cards/review')
  await expect(page.getByText('Нечего повторять')).toBeVisible()
})
