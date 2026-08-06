import { expect, test } from '@playwright/test'
import { cleanupTestUser, registerTestUser, seedDueCard } from './fixtures/db'

let email: string

test.afterEach(async () => {
  if (email) await cleanupTestUser(email)
})

test('shows the offline banner while disconnected and hides it once back online', async ({ page, context }) => {
  await page.goto('/login')

  await context.setOffline(true)
  await expect(page.getByText('Нет соединения')).toBeVisible()

  await context.setOffline(false)
  await expect(page.getByText('Нет соединения')).not.toBeVisible()
})

test('queues an SM-2 grade in IndexedDB while offline and syncs it back once reconnected', async ({ page, context, baseURL }) => {
  const user = await registerTestUser(baseURL!, 'e2e_offline_sync')
  email = user.email
  await seedDueCard(user.id)

  await page.goto('/login')
  await page.getByPlaceholder('Логин/почта').fill(user.email)
  await page.getByPlaceholder('Пароль').fill(user.password)
  await page.locator('[data-slot="checkbox"]').click()
  await page.getByRole('button', { name: 'Войти' }).click()
  await page.waitForURL('**/library')

  // страница уже открыта и подгрузила карточку, ПОТОМ рвём связь - имитирует
  // "приложение уже открыто, соединение пропало", а не холодный офлайн-старт
  await page.goto('/cards/review')
  await expect(page.getByText('e2e тестовый вопрос')).toBeVisible()

  await context.setOffline(true)
  await expect(page.getByText('Нет соединения')).toBeVisible()

  await page.getByRole('button', { name: 'Показать ответ' }).click()
  await page.getByRole('button', { name: 'Норм' }).click()
  await expect(page.getByText('Сессия завершена')).toBeVisible()

  const pendingWhileOffline = await page.evaluate(async () => {
    const db = await new Promise<IDBDatabase>((resolve, reject) => {
      const req = indexedDB.open('medflow-review-queue', 1)
      req.onsuccess = () => resolve(req.result)
      req.onerror = () => reject(req.error)
    })
    return new Promise<number>((resolve) => {
      const tx = db.transaction('pending', 'readonly')
      tx.objectStore('pending').count().onsuccess = (e) => resolve((e.target as IDBRequest<number>).result)
    })
  })
  expect(pendingWhileOffline).toBe(1)

  await context.setOffline(false)
  await expect(page.getByText('Нет соединения')).not.toBeVisible()
  await expect(page.getByText(/Прогресс синхронизирован/)).toBeVisible({ timeout: 10_000 })

  const pendingAfterSync = await page.evaluate(async () => {
    const db = await new Promise<IDBDatabase>((resolve, reject) => {
      const req = indexedDB.open('medflow-review-queue', 1)
      req.onsuccess = () => resolve(req.result)
      req.onerror = () => reject(req.error)
    })
    return new Promise<number>((resolve) => {
      const tx = db.transaction('pending', 'readonly')
      tx.objectStore('pending').count().onsuccess = (e) => resolve((e.target as IDBRequest<number>).result)
    })
  })
  expect(pendingAfterSync).toBe(0)
})
