import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { AlertTriangle, ArrowLeft, Clock, Flag, Heart, Link2, ShieldAlert } from 'lucide-react'
import { toast } from 'sonner'
import {
  deleteCardsIdFavorite,
  deleteCardsIdStars,
  getCardsTasksId,
  getCardsTasksIdCards,
  postCardsIdFavorite,
  postCardsIdReport,
  postCardsIdStars,
  postCardsTasksIdShare,
} from '@/api/generated/medFlowAPI'
import { CardTaskStatus } from '@/api/generated'
import type { Card, CardTask } from '@/api/generated'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { TASK_STATUS_META } from './task-status-meta'
import { getTaskTopic } from './task-topic-cache'
import { StarRating } from './star-rating'

export function TaskDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [task, setTask] = useState<CardTask | null>(null)
  const [cards, setCards] = useState<Card[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    let cancelled = false

    async function load() {
      const t = await getCardsTasksId(id!)
      if (cancelled) return
      setTask(t)
      if (t.status === CardTaskStatus.done) {
        const res = await getCardsTasksIdCards(id!, {})
        if (!cancelled) setCards(res.data ?? [])
      }
      setLoading(false)
    }

    load()
    const interval = setInterval(() => {
      if (task?.status === CardTaskStatus.pending || task?.status === CardTaskStatus.processing || !task) load()
    }, 2000)
    return () => {
      cancelled = true
      clearInterval(interval)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, task?.status])

  async function shareTask() {
    if (!id) return
    try {
      const res = await postCardsTasksIdShare(id)
      const url = `${window.location.origin}/shared/${res.share_token}`
      await navigator.clipboard.writeText(url)
      toast.success('Ссылка скопирована в буфер обмена')
    } catch {
      toast.error('Не удалось создать ссылку')
    }
  }

  async function toggleFavorite(cardId: string, currentlyFavorite?: boolean | null) {
    try {
      if (currentlyFavorite) {
        await deleteCardsIdFavorite(cardId)
      } else {
        await postCardsIdFavorite(cardId)
      }
      setCards((prev) => prev.map((c) => (c.id === cardId ? { ...c, is_favorite: !currentlyFavorite } : c)))
    } catch {
      toast.error('Не удалось обновить избранное')
    }
  }

  async function rateStars(cardId: string, stars: number) {
    try {
      await postCardsIdStars(cardId, { stars })
      setCards((prev) =>
        prev.map((c) => {
          if (c.id !== cardId) return c
          const prevCount = c.ratings_count ?? 0
          const hadMine = c.my_stars != null
          const prevSum = (c.average_stars ?? 0) * prevCount
          const nextCount = hadMine ? prevCount : prevCount + 1
          const nextSum = hadMine ? prevSum - (c.my_stars ?? 0) + stars : prevSum + stars
          return { ...c, my_stars: stars, ratings_count: nextCount, average_stars: nextCount ? nextSum / nextCount : undefined }
        }),
      )
    } catch {
      toast.error('Не удалось сохранить оценку')
    }
  }

  async function removeStars(cardId: string) {
    try {
      await deleteCardsIdStars(cardId)
      setCards((prev) =>
        prev.map((c) => {
          if (c.id !== cardId || c.my_stars == null) return c
          const prevCount = c.ratings_count ?? 0
          const prevSum = (c.average_stars ?? 0) * prevCount
          const nextCount = Math.max(0, prevCount - 1)
          const nextSum = prevSum - c.my_stars
          return { ...c, my_stars: undefined, ratings_count: nextCount || undefined, average_stars: nextCount ? nextSum / nextCount : undefined }
        }),
      )
    } catch {
      toast.error('Не удалось убрать оценку')
    }
  }

  if (loading && !task) {
    return (
      <div className="mx-auto flex max-w-2xl flex-col gap-3 p-6">
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-32 rounded-2xl" />
      </div>
    )
  }

  if (!task) {
    return (
      <div className="p-16 text-center text-sm text-muted-foreground">
        Задача не найдена. <Link to="/cards" className="text-accent underline">К списку</Link>
      </div>
    )
  }

  const meta = TASK_STATUS_META[task.status ?? CardTaskStatus.pending]
  const topic = getTaskTopic(id ?? '')

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-5 p-6">
      <Link to="/cards" className="flex w-fit items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" /> К карточкам
      </Link>

      <div className="flex items-center justify-between gap-3 rounded-2xl border border-border bg-card p-5">
        <div>
          <h1 className="font-semibold text-foreground">{topic ?? 'Задача генерации карточек'}</h1>
          <p className="text-sm text-muted-foreground">Создана: {new Date(task.created_at!).toLocaleString('ru-RU')}</p>
        </div>
        <div className="flex items-center gap-2">
          {task.status === CardTaskStatus.done && (
            <button
              type="button"
              onClick={shareTask}
              className="flex items-center gap-1.5 rounded-full border border-border px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground"
            >
              <Link2 className="size-3.5" /> Поделиться
            </button>
          )}
          <Badge className={meta.className}>{meta.label}</Badge>
        </div>
      </div>

      {(task.status === CardTaskStatus.pending || task.status === CardTaskStatus.processing) && (
        <div className="flex items-center gap-3 rounded-2xl border border-border bg-card p-5">
          <Clock className="size-5 shrink-0 animate-pulse text-accent" />
          <div>
            <p className="text-sm text-foreground">
              {task.status === CardTaskStatus.pending ? 'Задача в очереди' : 'ИИ обрабатывает материал…'}
            </p>
            {task.position_in_queue != null && (
              <p className="text-xs text-muted-foreground">
                Позиция в очереди: {task.position_in_queue}
                {task.estimated_wait_seconds != null && ` · ~${task.estimated_wait_seconds} сек`}
              </p>
            )}
          </div>
        </div>
      )}

      {task.status === CardTaskStatus.failed && (
        <div className="flex items-center gap-3 rounded-2xl border border-destructive/30 bg-destructive/5 p-5 text-destructive">
          <AlertTriangle className="size-5 shrink-0" />
          <p className="text-sm">{task.error_message ?? 'Не удалось сгенерировать карточки'}</p>
        </div>
      )}

      {task.status === CardTaskStatus.done && (
        <div className="flex flex-col gap-3">
          <p className="text-sm text-muted-foreground">Сгенерировано карточек: {cards.length}</p>
          {cards.map((card) => (
            <CardItem
              key={card.id}
              card={card}
              onToggleFavorite={() => toggleFavorite(card.id!, card.is_favorite)}
              onRate={(stars) => rateStars(card.id!, stars)}
              onRemoveRating={() => removeStars(card.id!)}
            />
          ))}
        </div>
      )}
    </div>
  )
}

interface CardItemProps {
  card: Card
  onToggleFavorite: () => void
  onRate: (stars: number) => void
  onRemoveRating: () => void
}

function CardItem({ card, onToggleFavorite, onRate, onRemoveRating }: CardItemProps) {
  const [reporting, setReporting] = useState(false)
  const [reason, setReason] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [reported, setReported] = useState(false)

  async function submitReport() {
    if (!card.id || !reason.trim()) return
    setSubmitting(true)
    try {
      await postCardsIdReport(card.id, { reason: reason.trim() })
      setReported(true)
      setReporting(false)
      toast.success('Жалоба отправлена')
    } catch {
      toast.error('Не удалось отправить жалобу')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="rounded-2xl border border-border bg-card p-4">
      <div className="flex items-start justify-between gap-2">
        <p className="text-sm font-medium text-foreground">{card.question}</p>
        <button
          type="button"
          onClick={onToggleFavorite}
          aria-label={card.is_favorite ? 'Убрать из избранного' : 'Добавить в избранное'}
          className="shrink-0"
        >
          <Heart className={cn('size-4', card.is_favorite ? 'fill-destructive text-destructive' : 'text-muted-foreground')} />
        </button>
      </div>
      <p className="mt-1.5 text-sm text-muted-foreground">{card.answer}</p>

      <div className="mt-3">
        <StarRating
          myStars={card.my_stars}
          averageStars={card.average_stars}
          ratingsCount={card.ratings_count}
          onRate={onRate}
          onRemove={onRemoveRating}
        />
      </div>

      <div className="mt-3 flex items-start gap-1.5 rounded-lg bg-secondary p-2 text-xs text-secondary-foreground">
        <ShieldAlert className="mt-0.5 size-3.5 shrink-0" />
        <span>{card.disclaimer ?? 'Сгенерировано ИИ. Проверьте по источнику.'}</span>
      </div>

      {reported ? (
        <p className="mt-2 text-xs text-muted-foreground">Жалоба отправлена на рассмотрение</p>
      ) : reporting ? (
        <div className="mt-2 flex flex-col gap-2">
          <textarea
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="В чём ошибка?"
            maxLength={2000}
            rows={2}
            className="w-full rounded-lg border border-border bg-transparent p-2 text-sm outline-none focus-visible:border-ring"
          />
          <div className="flex gap-2">
            <Button size="sm" variant="outline" onClick={() => setReporting(false)}>Отмена</Button>
            <Button size="sm" disabled={!reason.trim() || submitting} onClick={submitReport}>
              Отправить
            </Button>
          </div>
        </div>
      ) : (
        <button
          onClick={() => setReporting(true)}
          className="mt-2 flex items-center gap-1 text-xs text-muted-foreground hover:text-destructive"
        >
          <Flag className="size-3" /> Пожаловаться на ошибку
        </button>
      )}
    </div>
  )
}
