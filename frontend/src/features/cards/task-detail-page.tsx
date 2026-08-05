import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { AlertTriangle, ArrowLeft, Clock, Flag, ShieldAlert } from 'lucide-react'
import { toast } from 'sonner'
import { getCardsTasksId, getCardsTasksIdCards, postCardsIdReport } from '@/api/generated/medFlowAPI'
import { CardTaskStatus } from '@/api/generated'
import type { Card, CardTask } from '@/api/generated'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { TASK_STATUS_META } from './task-status-meta'
import { getTaskTopic } from './task-topic-cache'

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
        <Badge className={meta.className}>{meta.label}</Badge>
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
            <CardItem key={card.id} card={card} />
          ))}
        </div>
      )}
    </div>
  )
}

function CardItem({ card }: { card: Card }) {
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
      <p className="text-sm font-medium text-foreground">{card.question}</p>
      <p className="mt-1.5 text-sm text-muted-foreground">{card.answer}</p>
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
