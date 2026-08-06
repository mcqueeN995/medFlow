import { useEffect } from 'react'
import { toast } from 'sonner'
import { postCardsIdRate } from '@/api/generated/medFlowAPI'
import { listPendingReviews, removePendingReview } from '@/lib/review-queue'

let flushing = false

// flushReviewQueue - последовательно дожимает очередь офлайн-оценок карточек.
// Останавливается на первой сетевой ошибке (значит связь всё ещё не
// восстановилась) - оставшиеся записи дождутся следующего вызова.
export async function flushReviewQueue(): Promise<void> {
  if (flushing) return
  flushing = true
  try {
    const pending = await listPendingReviews()
    if (pending.length === 0) return

    let synced = 0
    for (const entry of pending) {
      try {
        await postCardsIdRate(entry.cardId, { grade: entry.grade })
        await removePendingReview(entry.id)
        synced += 1
      } catch (err) {
        if (!(err as { response?: unknown }).response) break // всё ещё офлайн - остальное досинкаем позже
        await removePendingReview(entry.id) // сервер ответил (не сетевая ошибка) - запись невалидна, не зацикливаемся на ней
      }
    }
    if (synced > 0) {
      toast.success(`Прогресс синхронизирован (карточек: ${synced})`)
    }
  } finally {
    flushing = false
  }
}

// useReviewQueueSync - подключается один раз в App.tsx: досылает очередь при
// старте приложения и при каждом восстановлении соединения.
export function useReviewQueueSync(): void {
  useEffect(() => {
    void flushReviewQueue()
    window.addEventListener('online', flushReviewQueue)
    return () => window.removeEventListener('online', flushReviewQueue)
  }, [])
}
