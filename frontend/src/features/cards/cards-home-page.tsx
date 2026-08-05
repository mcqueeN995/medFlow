import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { BrainCircuit, Flame, Layers, Plus, Target } from 'lucide-react'
import { getCardsStats, getCardsTasks } from '@/api/generated/medFlowAPI'
import { CardTaskStatus } from '@/api/generated'
import type { CardTask, CardsStats } from '@/api/generated'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { buttonVariants } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { TASK_STATUS_META } from './task-status-meta'
import { getTaskTopic } from './task-topic-cache'

function formatDateTime(iso?: string): string {
  if (!iso) return ''
  return new Intl.DateTimeFormat('ru-RU', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' }).format(
    new Date(iso),
  )
}

export function CardsHomePage() {
  const [stats, setStats] = useState<CardsStats | null>(null)
  const [tasks, setTasks] = useState<CardTask[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false

    async function load() {
      const [statsRes, tasksRes] = await Promise.all([getCardsStats(), getCardsTasks({})])
      if (cancelled) return
      setStats(statsRes)
      setTasks(tasksRes.data ?? [])
      setLoading(false)
    }

    load()
    // Опрашиваем, пока страница открыта, чтобы видеть переход
    // pending → processing → done без ручного обновления.
    const interval = setInterval(load, 3000)
    return () => {
      cancelled = true
      clearInterval(interval)
    }
  }, [])

  return (
    <div className="flex flex-col gap-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold text-primary">ИИ-карточки</h1>
          <p className="text-sm text-muted-foreground">Генерация из ваших материалов + интервальное повторение (SM-2)</p>
        </div>
        <div className="flex gap-2">
          <Link to="/cards/review" className={cn(buttonVariants({ variant: 'outline' }), 'h-9 rounded-full px-4')}>
            <BrainCircuit className="size-4" /> Повторить
          </Link>
          <Link
            to="/cards/create"
            className={cn(buttonVariants(), 'h-9 rounded-full bg-linear-to-r from-primary to-accent px-4 text-primary-foreground')}
          >
            <Plus className="size-4" /> Создать карточки
          </Link>
        </div>
      </div>

      {loading ? (
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-20 rounded-2xl" />)}
        </div>
      ) : (
        stats && (
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
            <StatTile icon={Layers} label="Изучено карточек" value={stats.total_cards_learned ?? 0} />
            <StatTile icon={Target} label="К повторению сегодня" value={stats.due_today ?? 0} accent />
            <StatTile icon={Flame} label="Серия дней" value={stats.streak_days ?? 0} />
            <StatTile icon={BrainCircuit} label="Средний ease factor" value={(stats.avg_ease_factor ?? 2.5).toFixed(2)} />
          </div>
        )
      )}

      <div className="flex flex-col gap-2">
        <h2 className="text-sm font-semibold text-foreground">Мои задачи</h2>
        {loading ? (
          <div className="flex flex-col gap-2">
            {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-16 rounded-xl" />)}
          </div>
        ) : tasks.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
            Пока нет задач — загрузите PDF и создайте первую подборку карточек
          </div>
        ) : (
          <ul className="flex flex-col gap-2">
            {tasks.map((task) => {
              const meta = TASK_STATUS_META[task.status ?? CardTaskStatus.pending]
              return (
                <li key={task.id}>
                  <Link
                    to={`/cards/tasks/${task.id}`}
                    className="flex items-center justify-between gap-3 rounded-xl border border-border bg-card p-3.5 hover:bg-secondary"
                  >
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-foreground">
                        {getTaskTopic(task.id ?? '') ?? `Задача от ${formatDateTime(task.created_at)}`}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {task.status === CardTaskStatus.pending && task.position_in_queue != null
                          ? `Позиция в очереди: ${task.position_in_queue}`
                          : task.status === CardTaskStatus.done
                            ? `${task.cards_count} карточек`
                            : formatDateTime(task.created_at)}
                      </p>
                    </div>
                    <Badge className={meta.className}>{meta.label}</Badge>
                  </Link>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}

function StatTile({
  icon: Icon,
  label,
  value,
  accent,
}: {
  icon: typeof Layers
  label: string
  value: number | string
  accent?: boolean
}) {
  return (
    <div className="flex flex-col gap-1.5 rounded-2xl border border-border bg-card p-4">
      <Icon className={cn('size-4', accent ? 'text-accent' : 'text-muted-foreground')} />
      <span className="text-xl font-bold text-foreground">{value}</span>
      <span className="text-xs text-muted-foreground">{label}</span>
    </div>
  )
}
