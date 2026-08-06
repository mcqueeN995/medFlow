import path from 'node:path'
import { config } from 'dotenv'
import { Pool } from 'pg'

// e2e читает те же креды/порты, что docker-compose (корневой .env) - не
// дублирует их отдельным набором переменных для тестов.
config({ path: path.resolve(import.meta.dirname, '../../../.env') })

export const pool = new Pool({
  host: 'localhost',
  port: Number(process.env.POSTGRES_PORT ?? 5432),
  user: process.env.POSTGRES_USER,
  password: process.env.POSTGRES_PASSWORD,
  database: process.env.POSTGRES_DB,
})

export interface TestUser {
  id: string
  email: string
  password: string
  accessToken: string
}

// registerTestUser - идёт через настоящий POST /auth/register (не прямой
// INSERT), т.к. пароль в БД хранится Argon2-хэшем - проще и надёжнее
// зарегистрироваться реальным запросом, чем повторять хэширование бэкенда в JS.
export async function registerTestUser(baseURL: string, prefix: string): Promise<TestUser> {
  const suffix = `${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
  const email = `${prefix}_${suffix}@sechenov.ru`
  const password = 'password123'
  const res = await fetch(`${baseURL}/api/v1/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email,
      password,
      nickname: `${prefix}_${suffix}`,
      university: 'sechenov',
      agree_to_terms: true,
    }),
  })
  if (!res.ok) throw new Error(`registerTestUser failed: ${res.status} ${await res.text()}`)
  const body = (await res.json()) as { user: { id: string }; access_token: string }
  return { id: body.user.id, email, password, accessToken: body.access_token }
}

// seedDueCard - обходит реальный (флейкующий без настоящего LLM) конвейер
// генерации: пишет card_task/card/card_progress напрямую, с next_review_at
// в прошлом - карточка гарантированно попадает в батч /cards/review.
export async function seedDueCard(userId: string): Promise<{ taskId: string; cardId: string }> {
  const task = await pool.query(
    `INSERT INTO card_tasks (user_id, source_type, status, cards_count) VALUES ($1, 'user_upload', 'done', 1) RETURNING id`,
    [userId],
  )
  const taskId = task.rows[0].id as string

  const card = await pool.query(
    `INSERT INTO cards (task_id, question, answer, topic) VALUES ($1, $2, $3, 'e2e') RETURNING id`,
    [taskId, 'e2e тестовый вопрос', 'e2e тестовый ответ'],
  )
  const cardId = card.rows[0].id as string

  await pool.query(
    `INSERT INTO card_progress (user_id, card_id, next_review_at) VALUES ($1, $2, now() - interval '1 day')`,
    [userId, cardId],
  )

  return { taskId, cardId }
}

export async function cleanupTestUser(email: string): Promise<void> {
  await pool.query(
    `DELETE FROM card_progress WHERE user_id IN (SELECT id FROM users WHERE email = $1)`,
    [email],
  )
  await pool.query(
    `DELETE FROM cards WHERE task_id IN (SELECT id FROM card_tasks WHERE user_id IN (SELECT id FROM users WHERE email = $1))`,
    [email],
  )
  await pool.query(`DELETE FROM card_tasks WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
  await pool.query(`DELETE FROM uploads WHERE uploader_id IN (SELECT id FROM users WHERE email = $1)`, [email])
  await pool.query(`DELETE FROM push_subscriptions WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
  await pool.query(`DELETE FROM push_preferences WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
  await pool.query(`DELETE FROM refresh_tokens WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
  await pool.query(`DELETE FROM threads WHERE author_id IN (SELECT id FROM users WHERE email = $1)`, [email])
  await pool.query(`DELETE FROM users WHERE email = $1`, [email])
}
