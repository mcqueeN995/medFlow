import { useEffect, useState } from 'react'
import { Activity, Ban, Layers, ListTodo, MessageSquare, Users } from 'lucide-react'
import { getAdminStats } from '@/api/generated/medFlowAPI'
import type { AdminStats } from '@/api/generated'
import { Skeleton } from '@/components/ui/skeleton'

const METRICS: Array<{ key: keyof AdminStats; label: string; icon: typeof Users }> = [
  { key: 'users_total', label: 'Всего пользователей', icon: Users },
  { key: 'users_banned', label: 'Забанено', icon: Ban },
  { key: 'threads_total', label: 'Тредов на форуме', icon: MessageSquare },
  { key: 'card_tasks_total', label: 'Задач на карточки', icon: Layers },
  { key: 'card_tasks_pending', label: 'Задач в очереди', icon: ListTodo },
  { key: 'active_sessions', label: 'Активных сессий', icon: Activity },
]

export function AdminStatsPage() {
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    getAdminStats()
      .then(setStats)
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-24 rounded-2xl" />
        ))}
      </div>
    )
  }

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
      {METRICS.map(({ key, label, icon: Icon }) => (
        <div key={key} className="flex flex-col gap-2 rounded-2xl border border-border bg-card p-4">
          <Icon className="size-5 text-accent" />
          <span className="text-2xl font-bold text-foreground">{stats?.[key] ?? 0}</span>
          <span className="text-xs text-muted-foreground">{label}</span>
        </div>
      ))}
    </div>
  )
}
