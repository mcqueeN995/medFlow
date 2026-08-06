import { expect, test } from '@playwright/test'
import { cleanupTestUser } from './fixtures/db'

const emails: string[] = []

test.afterEach(async () => {
  while (emails.length) {
    await cleanupTestUser(emails.pop()!)
  }
})

test('registers a new account through the UI and lands on the library', async ({ page }) => {
  const suffix = `${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
  const email = `e2e_reg_${suffix}@sechenov.ru`
  emails.push(email)

  await page.goto('/register')
  await page.getByPlaceholder('Почта').fill(email)
  await page.getByPlaceholder('Никнейм').fill(`e2e_reg_${suffix}`)
  await page.getByPlaceholder('Пароль (мин. 8 символов)').fill('password123')
  await page.locator('[data-slot="checkbox"]').click()
  await page.getByRole('button', { name: 'Зарегистрироваться' }).click()

  await page.waitForURL('**/library')
  await expect(page.getByText('Библиотека учебников')).toBeVisible()
})

test('logs in, sees profile, logs out, and browses as a guest afterwards', async ({ page }) => {
  const suffix = `${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
  const email = `e2e_login_${suffix}@sechenov.ru`
  emails.push(email)

  await page.goto('/register')
  await page.getByPlaceholder('Почта').fill(email)
  await page.getByPlaceholder('Никнейм').fill(`e2e_login_${suffix}`)
  await page.getByPlaceholder('Пароль (мин. 8 символов)').fill('password123')
  await page.locator('[data-slot="checkbox"]').click()
  await page.getByRole('button', { name: 'Зарегистрироваться' }).click()
  await page.waitForURL('**/library')

  await page.goto('/profile')
  await expect(page.getByText(email)).toBeVisible()

  await page.getByRole('button', { name: 'Выйти' }).click()
  await page.waitForURL('**/login')

  // гость свободно листает библиотеку без входа
  await page.goto('/library')
  await expect(page.getByText('Библиотека учебников')).toBeVisible()
  await expect(page.getByRole('link', { name: 'Войти' })).toBeVisible()

  // но форум для гостя закрыт
  await page.goto('/forum')
  await expect(page.getByText(/только для авторизованных/i)).toBeVisible()
})

test('rejects login with a wrong password', async ({ page }) => {
  const suffix = `${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
  const email = `e2e_wrongpw_${suffix}@sechenov.ru`
  emails.push(email)

  await page.goto('/register')
  await page.getByPlaceholder('Почта').fill(email)
  await page.getByPlaceholder('Никнейм').fill(`e2e_wrongpw_${suffix}`)
  await page.getByPlaceholder('Пароль (мин. 8 символов)').fill('password123')
  await page.locator('[data-slot="checkbox"]').click()
  await page.getByRole('button', { name: 'Зарегистрироваться' }).click()
  await page.waitForURL('**/library')

  await page.goto('/profile')
  await page.getByRole('button', { name: 'Выйти' }).click()
  await page.waitForURL('**/login')

  await page.getByPlaceholder('Логин/почта').fill(email)
  await page.getByPlaceholder('Пароль').fill('totally-wrong-password')
  await page.locator('[data-slot="checkbox"]').click()
  await page.getByRole('button', { name: 'Войти' }).click()

  await expect(page.getByText('Неверный логин или пароль')).toBeVisible()
})
