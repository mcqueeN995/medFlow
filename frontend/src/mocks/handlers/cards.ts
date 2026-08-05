import { http, HttpResponse } from 'msw'
import { CardDifficulty, CardTaskStatus } from '@/api/generated'
import type { Card, CardTask, CardsStats, Report, ReviewCard } from '@/api/generated'
import { applySM2, DEFAULT_SM2_PROGRESS, nextReviewDate, type SM2Progress } from '@/lib/sm2'

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
}

const tasks: MockTask[] = []
const cards: MockCard[] = []
let taskCounter = 0
let cardCounter = 0

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

function toCard(c: MockCard): Card {
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
  }
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
      status: CardTaskStatus.pending,
      position_in_queue: activeCount + 1,
      estimated_wait_seconds: (activeCount + 1) * 15,
      cards_count: 0,
      error_message: null,
      started_at: null,
      finished_at: null,
      created_at: new Date().toISOString(),
    }
    tasks.unshift(task)
    runTaskLifecycle(task, difficulty, cardsCount)

    return HttpResponse.json(task, { status: 201 })
  }),

  http.get(`${API}/cards/tasks`, ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status')
    const filtered = status ? tasks.filter((t) => t.status === status) : tasks
    return HttpResponse.json({
      data: filtered,
      pagination: { page: 1, limit: filtered.length, total: filtered.length, has_next: false },
    })
  }),

  http.get(`${API}/cards/tasks/:id`, ({ params }) => {
    const task = tasks.find((t) => t.id === params.id)
    if (!task) return HttpResponse.json({ error: { code: 'not_found', message: 'Задача не найдена' } }, { status: 404 })
    return HttpResponse.json(task)
  }),

  http.get(`${API}/cards/tasks/:id/cards`, ({ params }) => {
    const task = tasks.find((t) => t.id === params.id)
    if (!task) return HttpResponse.json({ error: { code: 'not_found', message: 'Задача не найдена' } }, { status: 404 })
    const taskCards = cards.filter((c) => c.taskId === params.id)
    return HttpResponse.json({
      data: taskCards.map(toCard),
      pagination: { page: 1, limit: taskCards.length, total: taskCards.length, has_next: false },
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
