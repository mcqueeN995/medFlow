import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ReviewPage } from './review-page'
import { server } from '@/test/setup'
import * as reviewQueue from '@/lib/review-queue'

const API = '*/api/v1'

function mockReviewBatch(count = 1) {
  const cards = Array.from({ length: count }, (_, i) => ({
    card_id: `card-${i}`,
    question: `Вопрос ${i}`,
    answer: `Ответ ${i}`,
  }))
  server.use(
    http.get(`${API}/cards/review`, () => HttpResponse.json({ data: cards, count: cards.length })),
  )
  return cards
}

function renderReviewPage() {
  return render(
    <MemoryRouter>
      <ReviewPage />
    </MemoryRouter>,
  )
}

describe('ReviewPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows the empty state when nothing is due', async () => {
    mockReviewBatch(0)
    renderReviewPage()
    expect(await screen.findByText('Нечего повторять')).toBeInTheDocument()
  })

  it('reveals the answer, submits a grade, and advances to completion', async () => {
    mockReviewBatch(1)
    server.use(http.post(`${API}/cards/:id/rate`, () => HttpResponse.json({ next_review_at: new Date().toISOString() })))
    const user = userEvent.setup()
    renderReviewPage()

    expect(await screen.findByText('Вопрос 0')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Показать ответ' }))
    expect(await screen.findByText('Ответ 0')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Норм' }))
    expect(await screen.findByText('Сессия завершена')).toBeInTheDocument()
  })

  it('queues the grade offline instead of losing it when the request fails at the network level', async () => {
    mockReviewBatch(1)
    server.use(http.post(`${API}/cards/:id/rate`, () => HttpResponse.error()))
    const enqueueSpy = vi.spyOn(reviewQueue, 'enqueueReview').mockResolvedValue()
    const user = userEvent.setup()
    renderReviewPage()

    await screen.findByText('Вопрос 0')
    await user.click(screen.getByRole('button', { name: 'Показать ответ' }))
    await user.click(screen.getByRole('button', { name: 'Норм' }))

    await waitFor(() => expect(enqueueSpy).toHaveBeenCalledWith('card-0', 2))
    // офлайн-путь всё равно листает карточку дальше, а не блокирует пользователя
    expect(await screen.findByText('Сессия завершена')).toBeInTheDocument()
  })
})
