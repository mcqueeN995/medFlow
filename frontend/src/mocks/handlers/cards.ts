import { http, HttpResponse } from 'msw'
import { CardDifficulty, CardTaskStatus } from '@/api/generated'
import type { Card, CardCatalogEntry, CardTask, CardsStats, Report, ReviewCard, SharedCardTask } from '@/api/generated'
import { applySM2, DEFAULT_SM2_PROGRESS, nextReviewDate, type SM2Progress } from '@/lib/sm2'
import { userByToken } from './auth'

const API = '*/api/v1'
const DISCLAIMER = 'Сгенерировано ИИ. Проверьте по источнику перед использованием.'
const MAX_ACTIVE_TASKS = 3

interface MockCard {
  id: string
  taskId: string
  question: string
  answer: string
  topic: string
  subtopic?: string
  chapter?: string
  page_approx?: number
  difficulty: CardDifficulty
  report_count: number
  created_at: string
  progress: SM2Progress
  nextReviewAt: string
}

interface MockTask extends CardTask {
  topic: string
  authorId: string
  sourceType: 'catalog_textbook' | 'user_upload'
  textbookId?: string
  shareToken?: string | null
}

const tasks: MockTask[] = []
const cards: MockCard[] = []
let taskCounter = 0
let cardCounter = 0

// cardId -> userId -> избранное/звёзды. Отдельные карты, независимые друг от
// друга и от SM-2 прогресса (card.progress выше) - как и в реальном бэкенде
// (card_favorites/card_ratings отдельные таблицы).
const favorites = new Map<string, Set<string>>()
const ratings = new Map<string, Map<string, number>>()

function isFavorited(cardId: string, userId: string): boolean {
  return favorites.get(cardId)?.has(userId) ?? false
}

function ratingSummary(cardId: string, userId: string): { average?: number; count: number; mine?: number } {
  const userStars = ratings.get(cardId)
  if (!userStars || userStars.size === 0) return { count: 0 }
  const values = [...userStars.values()]
  const average = values.reduce((sum, v) => sum + v, 0) / values.length
  return { average, count: values.length, mine: userStars.get(userId) }
}

function requireUser(request: Request) {
  return userByToken(request.headers.get('authorization'))
}

function unauthorized() {
  return HttpResponse.json({ error: { code: 'UNAUTHORIZED', message: 'authentication required' } }, { status: 401 })
}

// Небольшой банк реалистичных вопросов по частым темам — по остальным темам
// генерируем шаблонные карточки (демонстрация потока, не настоящий RAG).
const TOPIC_BANKS: Record<string, Array<[string, string]>> = {
  сердц: [
    ['Сколько камер в сердце человека?', 'Четыре: два предсердия и два желудочка.'],
    ['Какой слой сердца обеспечивает сокращение?', 'Миокард — средний мышечный слой.'],
    ['Что такое синоатриальный узел?', 'Естественный водитель ритма сердца, расположен в правом предсердии.'],
    ['Какой клапан находится между левым предсердием и желудочком?', 'Митральный (двустворчатый) клапан.'],
  ],
  фармак: [
    ['Что такое период полувыведения препарата?', 'Время, за которое концентрация вещества в крови снижается вдвое.'],
    ['Чем агонист отличается от антагониста рецептора?', 'Агонист активирует рецептор, антагонист блокирует его действие.'],
    ['Что такое биодоступность препарата?', 'Доля дозы, достигающая системного кровотока в неизменном виде.'],
  ],
  анатом: [
    ['Сколько позвонков в шейном отделе позвоночника?', 'Семь шейных позвонков (C1–C7).'],
    ['Как называется самая длинная кость тела человека?', 'Бедренная кость (femur).'],
  ],
  биохим: [
    ['Что является конечным продуктом гликолиза?', 'Пируват (в анаэробных условиях — лактат).'],
    ['Где в клетке происходит синтез АТФ окислительным фосфорилированием?', 'В митохондриях, на внутренней мембране.'],
  ],
}

function generateQA(topic: string, index: number): [string, string] {
  const key = Object.keys(TOPIC_BANKS).find((k) => topic.toLowerCase().includes(k))
  const bank = key ? TOPIC_BANKS[key] : null
  if (bank && bank[index % bank.length]) return bank[index % bank.length]
  return [
    `Вопрос №${index + 1} по теме «${topic}»`,
    `Ответ №${index + 1} — карточка сгенерирована демо-ИИ на основе загруженного материала.`,
  ]
}

function toCard(c: MockCard, viewerId?: string): Card {
  const { average, count, mine } = viewerId ? ratingSummary(c.id, viewerId) : { count: 0 }
  return {
    id: c.id,
    chapter: c.chapter,
    topic: c.topic,
    subtopic: c.subtopic,
    question: c.question,
    answer: c.answer,
    page_approx: c.page_approx,
    difficulty: c.difficulty,
    disclaimer: DISCLAIMER,
    created_at: c.created_at,
    is_favorite: viewerId ? isFavorited(c.id, viewerId) : undefined,
    average_stars: average,
    ratings_count: count || undefined,
    my_stars: mine,
  }
}

// authorizeTaskAccess - то же правило ослабления владения, что и на бэкенде
// (CardService.authorizeTaskAccess): catalog_textbook открыт любому
// авторизованному, user_upload - только автору.
function authorizeTaskAccess(task: MockTask, userId: string): boolean {
  return task.sourceType !== 'user_upload' || task.authorId === userId
}

function toReviewCard(c: MockCard): ReviewCard {
  return {
    card_id: c.id,
    question: c.question,
    answer: c.answer,
    disclaimer: DISCLAIMER,
    progress: {
      next_review_at: c.nextReviewAt,
      interval_days: c.progress.intervalDays,
      ease_factor: c.progress.easeFactor,
      repetitions: c.progress.repetitions,
    },
  }
}

function runTaskLifecycle(task: MockTask, difficulty: CardDifficulty, cardsCount: number) {
  setTimeout(() => {
    task.status = CardTaskStatus.processing
    task.position_in_queue = null
    task.started_at = new Date().toISOString()
  }, 1500)

  setTimeout(() => {
    task.status = CardTaskStatus.done
    task.finished_at = new Date().toISOString()
    task.cards_count = cardsCount

    for (let i = 0; i < cardsCount; i++) {
      const [question, answer] = generateQA(task.topic, i)
      cardCounter += 1
      cards.push({
        id: `card-${cardCounter}`,
        taskId: task.id!,
        question,
        answer,
        topic: task.topic,
        page_approx: 12 + i * 6,
        difficulty,
        report_count: 0,
        created_at: new Date().toISOString(),
        progress: { ...DEFAULT_SM2_PROGRESS },
        // Свежесгенерированные карточки сразу доступны к повторению.
        nextReviewAt: new Date().toISOString(),
      })
    }
  }, 5000)
}

export const cardsHandlers = [
  http.post(`${API}/cards/tasks`, async ({ request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()

    const body = (await request.json()) as {
      file_id: string
      textbook_id?: string
      topic: string
      difficulty?: CardDifficulty
      cards_count?: number
    }

    const activeCount = tasks.filter(
      (t) => t.status === CardTaskStatus.pending || t.status === CardTaskStatus.processing,
    ).length
    if (activeCount >= MAX_ACTIVE_TASKS) {
      return HttpResponse.json(
        { error: { code: 'rate_limited', message: `Не более ${MAX_ACTIVE_TASKS} активных задач одновременно` } },
        { status: 429 },
      )
    }

    taskCounter += 1
    const difficulty = body.difficulty ?? CardDifficulty.medium
    const cardsCount = body.cards_count ?? 10
    const task: MockTask = {
      id: `task-${taskCounter}`,
      topic: body.topic,
      authorId: user.id!,
      sourceType: body.textbook_id ? 'catalog_textbook' : 'user_upload',
      textbookId: body.textbook_id,
      status: CardTaskStatus.pending,
      position_in_queue: activeCount + 1,
      estimated_wait_seconds: (activeCount + 1) * 15,
      cards_count: 0,
      error_message: null,
      started_at: null,
      finished_at: null,
      created_at: new Date().toISOString(),
      share_token: null,
    }
    tasks.unshift(task)
    runTaskLifecycle(task, difficulty, cardsCount)

    return HttpResponse.json(task, { status: 201 })
  }),

  http.get(`${API}/cards/tasks`, ({ request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()
    const url = new URL(request.url)
    const status = url.searchParams.get('status')
    let filtered = tasks.filter((t) => t.authorId === user.id)
    if (status) filtered = filtered.filter((t) => t.status === status)
    return HttpResponse.json({
      data: filtered,
      pagination: { page: 1, limit: filtered.length, total: filtered.length, has_next: false },
    })
  }),

  http.get(`${API}/cards/catalog`, ({ request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.toLowerCase()

    const entries: CardCatalogEntry[] = tasks
      .filter((t) => t.sourceType === 'catalog_textbook' && t.status === CardTaskStatus.done)
      .filter((t) => !q || t.topic.toLowerCase().includes(q))
      .map((t) => ({
        task_id: t.id!,
        textbook_id: t.textbookId!,
        textbook_title: t.topic, // мок не хранит настоящий каталог учебников отдельно от задач
        topic: t.topic,
        difficulty: CardDifficulty.medium,
        cards_count: t.cards_count ?? 0,
        created_at: t.created_at!,
      }))

    return HttpResponse.json({
      data: entries,
      pagination: { page: 1, limit: entries.length, total: entries.length, has_next: false },
    })
  }),

  http.get(`${API}/cards/tasks/:id`, ({ params, request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()
    const task = tasks.find((t) => t.id === params.id)
    if (!task) return HttpResponse.json({ error: { code: 'not_found', message: 'Задача не найдена' } }, { status: 404 })
    if (!authorizeTaskAccess(task, user.id!)) {
      return HttpResponse.json({ error: { code: 'FORBIDDEN', message: 'you are not the owner of this task' } }, { status: 403 })
    }
    return HttpResponse.json(task)
  }),

  http.get(`${API}/cards/tasks/:id/cards`, ({ params, request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()
    const task = tasks.find((t) => t.id === params.id)
    if (!task) return HttpResponse.json({ error: { code: 'not_found', message: 'Задача не найдена' } }, { status: 404 })
    if (!authorizeTaskAccess(task, user.id!)) {
      return HttpResponse.json({ error: { code: 'FORBIDDEN', message: 'you are not the owner of this task' } }, { status: 403 })
    }
    const taskCards = cards.filter((c) => c.taskId === params.id)
    return HttpResponse.json({
      data: taskCards.map((c) => toCard(c, user.id)),
      pagination: { page: 1, limit: taskCards.length, total: taskCards.length, has_next: false },
    })
  }),

  http.post(`${API}/cards/tasks/:id/share`, ({ params, request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()
    const task = tasks.find((t) => t.id === params.id)
    if (!task) return HttpResponse.json({ error: { code: 'not_found', message: 'Задача не найдена' } }, { status: 404 })
    if (task.authorId !== user.id) {
      return HttpResponse.json({ error: { code: 'FORBIDDEN', message: 'you are not the owner of this task' } }, { status: 403 })
    }
    if (!task.share_token) task.share_token = `share-${task.id}-${crypto.randomUUID().slice(0, 8)}`
    return HttpResponse.json({ share_token: task.share_token })
  }),

  http.delete(`${API}/cards/tasks/:id/share`, ({ params, request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()
    const task = tasks.find((t) => t.id === params.id)
    if (!task) return HttpResponse.json({ error: { code: 'not_found', message: 'Задача не найдена' } }, { status: 404 })
    if (task.authorId !== user.id) {
      return HttpResponse.json({ error: { code: 'FORBIDDEN', message: 'you are not the owner of this task' } }, { status: 403 })
    }
    task.share_token = null
    return new HttpResponse(null, { status: 204 })
  }),

  http.get(`${API}/cards/shared/:token`, ({ params }) => {
    const task = tasks.find((t) => t.share_token === params.token)
    if (!task || task.status !== CardTaskStatus.done) {
      return HttpResponse.json({ error: { code: 'not_found', message: 'Ссылка недействительна' } }, { status: 404 })
    }
    const taskCards = cards.filter((c) => c.taskId === task.id)
    const shared: SharedCardTask = {
      topic: task.topic,
      difficulty: CardDifficulty.medium,
      cards: taskCards.map((c) => toCard(c)),
    }
    return HttpResponse.json(shared)
  }),

  http.post(`${API}/cards/:id/favorite`, ({ params, request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()
    const card = cards.find((c) => c.id === params.id)
    if (!card) return HttpResponse.json({ error: { code: 'not_found', message: 'Карточка не найдена' } }, { status: 404 })
    const task = tasks.find((t) => t.id === card.taskId)
    if (task && !authorizeTaskAccess(task, user.id!)) {
      return HttpResponse.json({ error: { code: 'FORBIDDEN', message: 'you are not the owner of this task' } }, { status: 403 })
    }
    if (!favorites.has(card.id)) favorites.set(card.id, new Set())
    favorites.get(card.id)!.add(user.id!)
    return new HttpResponse(null, { status: 204 })
  }),

  http.delete(`${API}/cards/:id/favorite`, ({ params, request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()
    const set = favorites.get(params.id as string)
    if (!set?.has(user.id!)) {
      return HttpResponse.json({ error: { code: 'not_found', message: 'Не в избранном' } }, { status: 404 })
    }
    set.delete(user.id!)
    return new HttpResponse(null, { status: 204 })
  }),

  http.get(`${API}/cards/favorites`, ({ request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()
    const favCards = cards.filter((c) => favorites.get(c.id)?.has(user.id!))
    return HttpResponse.json({
      data: favCards.map((c) => toCard(c, user.id)),
      pagination: { page: 1, limit: favCards.length, total: favCards.length, has_next: false },
    })
  }),

  http.get(`${API}/cards/favorites/review`, ({ request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()
    const url = new URL(request.url)
    const limit = Number(url.searchParams.get('limit') ?? '20')
    const now = new Date().toISOString()
    const due = cards
      .filter((c) => favorites.get(c.id)?.has(user.id!) && c.nextReviewAt <= now)
      .sort((a, b) => a.nextReviewAt.localeCompare(b.nextReviewAt))
    return HttpResponse.json({ data: due.slice(0, limit).map(toReviewCard), count: due.length })
  }),

  http.post(`${API}/cards/:id/stars`, async ({ params, request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()
    const card = cards.find((c) => c.id === params.id)
    if (!card) return HttpResponse.json({ error: { code: 'not_found', message: 'Карточка не найдена' } }, { status: 404 })
    const task = tasks.find((t) => t.id === card.taskId)
    if (task && !authorizeTaskAccess(task, user.id!)) {
      return HttpResponse.json({ error: { code: 'FORBIDDEN', message: 'you are not the owner of this task' } }, { status: 403 })
    }
    const { stars } = (await request.json()) as { stars: number }
    if (!ratings.has(card.id)) ratings.set(card.id, new Map())
    ratings.get(card.id)!.set(user.id!, stars)
    return new HttpResponse(null, { status: 204 })
  }),

  http.delete(`${API}/cards/:id/stars`, ({ params, request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()
    const userStars = ratings.get(params.id as string)
    if (!userStars?.has(user.id!)) {
      return HttpResponse.json({ error: { code: 'not_found', message: 'Оценка не найдена' } }, { status: 404 })
    }
    userStars.delete(user.id!)
    return new HttpResponse(null, { status: 204 })
  }),

  http.get(`${API}/cards/rated`, ({ request }) => {
    const user = requireUser(request)
    if (!user) return unauthorized()
    const ratedCards = cards.filter((c) => ratings.get(c.id)?.has(user.id!))
    return HttpResponse.json({
      data: ratedCards.map((c) => toCard(c, user.id)),
      pagination: { page: 1, limit: ratedCards.length, total: ratedCards.length, has_next: false },
    })
  }),

  http.get(`${API}/cards/review`, ({ request }) => {
    const url = new URL(request.url)
    const limit = Number(url.searchParams.get('limit') ?? '20')
    const now = new Date().toISOString()
    const due = cards.filter((c) => c.nextReviewAt <= now).sort((a, b) => a.nextReviewAt.localeCompare(b.nextReviewAt))
    return HttpResponse.json({ data: due.slice(0, limit).map(toReviewCard), count: due.length })
  }),

  http.post(`${API}/cards/:id/rate`, async ({ params, request }) => {
    const card = cards.find((c) => c.id === params.id)
    if (!card) return HttpResponse.json({ error: { code: 'not_found', message: 'Карточка не найдена' } }, { status: 404 })
    const { grade } = (await request.json()) as { grade: number }
    card.progress = applySM2(card.progress, grade)
    card.nextReviewAt = nextReviewDate(card.progress.intervalDays)
    return HttpResponse.json({
      next_review_at: card.nextReviewAt,
      interval_days: card.progress.intervalDays,
      ease_factor: card.progress.easeFactor,
      repetitions: card.progress.repetitions,
    })
  }),

  http.post(`${API}/cards/:id/report`, async ({ params, request }) => {
    const card = cards.find((c) => c.id === params.id)
    if (!card) return HttpResponse.json({ error: { code: 'not_found', message: 'Карточка не найдена' } }, { status: 404 })
    const { reason } = (await request.json()) as { reason: string }
    card.report_count += 1
    const report: Report = {
      id: crypto.randomUUID(),
      target_type: 'card',
      target_id: card.id,
      reason,
      status: 'pending',
      created_at: new Date().toISOString(),
    }
    return HttpResponse.json(report, { status: 201 })
  }),

  http.get(`${API}/cards/stats`, () => {
    const learned = cards.filter((c) => c.progress.repetitions > 0)
    const endOfToday = new Date()
    endOfToday.setHours(23, 59, 59, 999)
    const dueToday = cards.filter((c) => c.nextReviewAt <= endOfToday.toISOString())
    const avgEase = cards.length ? cards.reduce((sum, c) => sum + c.progress.easeFactor, 0) / cards.length : 2.5
    const byDifficulty = { easy: 0, medium: 0, hard: 0 }
    for (const c of cards) byDifficulty[c.difficulty] += 1

    const stats: CardsStats = {
      total_cards_learned: learned.length,
      due_today: dueToday.length,
      streak_days: cards.length > 0 ? 3 : 0,
      avg_ease_factor: Math.round(avgEase * 100) / 100,
      by_difficulty: byDifficulty,
    }
    return HttpResponse.json(stats)
  }),
]
