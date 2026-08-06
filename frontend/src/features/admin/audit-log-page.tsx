import { useEffect, useState } from 'react'
import { getAdminAuditLogs } from '@/api/generated/medFlowAPI'
import { AuditAction } from '@/api/generated'
import type { AuditLog } from '@/api/generated'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'

const ALL = '__all__'

const ACTION_LABELS: Record<AuditAction, string> = {
  [AuditAction.user_ban]: 'Бан пользователя',
  [AuditAction.user_unban]: 'Разбан пользователя',
  [AuditAction.user_role_change]: 'Смена роли',
  [AuditAction.thread_hide]: 'Скрытие треда',
  [AuditAction.thread_unhide]: 'Возврат треда',
  [AuditAction.thread_delete]: 'Удаление треда',
  [AuditAction.comment_hide]: 'Скрытие комментария',
  [AuditAction.comment_unhide]: 'Возврат комментария',
  [AuditAction.comment_delete]: 'Удаление комментария',
  [AuditAction.poi_create]: 'Создание точки на карте',
  [AuditAction.poi_update]: 'Изменение точки на карте',
  [AuditAction.poi_delete]: 'Удаление точки на карте',
  [AuditAction.textbook_create]: 'Добавление учебника',
  [AuditAction.textbook_update]: 'Изменение учебника',
  [AuditAction.textbook_delete]: 'Удаление учебника',
}

function formatDate(iso?: string) {
  if (!iso) return ''
  return new Date(iso).toLocaleString('ru-RU')
}

export function AdminAuditLogPage() {
  const [action, setAction] = useState<string>(ALL)
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    getAdminAuditLogs({ action: action === ALL ? undefined : (action as AuditAction), limit: 50 })
      .then((res) => setLogs(res.data ?? []))
      .finally(() => setLoading(false))
  }, [action])

  return (
    <div className="flex flex-col gap-4">
      <Select value={action} onValueChange={(v) => setAction(v ?? ALL)}>
        <SelectTrigger className="h-9 w-64 rounded-full">
          <SelectValue placeholder="Действие">{(v: string) => (v === ALL ? 'Все действия' : ACTION_LABELS[v as AuditAction])}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>Все действия</SelectItem>
          {Object.values(AuditAction).map((a) => (
            <SelectItem key={a} value={a}>
              {ACTION_LABELS[a]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {loading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-14 rounded-xl" />
          ))}
        </div>
      ) : logs.length === 0 ? (
        <p className="p-6 text-center text-sm text-muted-foreground">Записей пока нет</p>
      ) : (
        <div className="flex flex-col gap-1.5">
          {logs.map((l) => (
            <div key={l.id} className="flex flex-wrap items-center gap-2 rounded-xl border border-border bg-card px-3.5 py-2.5 text-sm">
              <Badge variant="outline">{ACTION_LABELS[l.action ?? AuditAction.user_ban]}</Badge>
              <span className="font-medium text-foreground">{l.actor_nickname}</span>
              {l.reason && <span className="text-muted-foreground">— {l.reason}</span>}
              <span className="ml-auto text-xs text-muted-foreground">{formatDate(l.created_at)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
