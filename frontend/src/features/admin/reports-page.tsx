import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Check, EyeOff, ExternalLink, X } from 'lucide-react'
import { toast } from 'sonner'
import {
  getAdminReports,
  patchAdminReportsId,
  postAdminCommentsIdHide,
  postAdminThreadsIdHide,
} from '@/api/generated/medFlowAPI'
import { PatchAdminReportsIdBodyStatus, ReportStatus } from '@/api/generated'
import type { Report } from '@/api/generated'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'

const ALL = '__all__'

const STATUS_LABELS: Record<ReportStatus, string> = {
  [ReportStatus.pending]: 'На рассмотрении',
  [ReportStatus.reviewed]: 'Рассмотрено',
  [ReportStatus.dismissed]: 'Отклонено',
}

const TARGET_LABELS: Record<string, string> = {
  thread: 'Тред',
  comment: 'Комментарий',
  card: 'Карточка',
}

function formatDate(iso?: string) {
  if (!iso) return ''
  return new Date(iso).toLocaleString('ru-RU')
}

export function AdminReportsPage() {
  const [status, setStatus] = useState<string>(ReportStatus.pending)
  const [reports, setReports] = useState<Report[]>([])
  const [loading, setLoading] = useState(true)
  const [busyId, setBusyId] = useState<string | null>(null)

  function load() {
    setLoading(true)
    getAdminReports({ status: status === ALL ? undefined : (status as ReportStatus), limit: 50 })
      .then((res) => setReports(res.data ?? []))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status])

  async function review(id: string, newStatus: typeof PatchAdminReportsIdBodyStatus.reviewed | typeof PatchAdminReportsIdBodyStatus.dismissed) {
    setBusyId(id)
    try {
      await patchAdminReportsId(id, { status: newStatus })
      toast.success(newStatus === PatchAdminReportsIdBodyStatus.reviewed ? 'Жалоба отмечена рассмотренной' : 'Жалоба отклонена')
      setReports((prev) => prev.filter((r) => r.id !== id))
    } catch {
      toast.error('Не удалось обновить жалобу')
    } finally {
      setBusyId(null)
    }
  }

  function targetLink(r: Report): string | null {
    if (r.target_thread_id) {
      return r.target_type === 'comment' ? `/forum/${r.target_thread_id}?highlight=${r.target_id}` : `/forum/${r.target_thread_id}`
    }
    if (r.target_task_id) return `/cards/tasks/${r.target_task_id}`
    return null
  }

  // Скрыть саму цель жалобы (тред/комментарий) прямо из списка жалоб, не
  // переходя на страницу треда, и сразу отметить жалобу рассмотренной - так
  // модератору не нужно делать два отдельных действия на двух разных страницах.
  async function hideTarget(r: Report) {
    if (r.target_type !== 'thread' && r.target_type !== 'comment') return
    const reason = window.prompt('Причина скрытия:')
    if (!reason?.trim()) return
    setBusyId(r.id!)
    try {
      if (r.target_type === 'thread') {
        await postAdminThreadsIdHide(r.target_thread_id!, { reason: reason.trim() })
      } else {
        await postAdminCommentsIdHide(r.target_id!, { reason: reason.trim() })
      }
      await patchAdminReportsId(r.id!, { status: PatchAdminReportsIdBodyStatus.reviewed })
      toast.success('Контент скрыт, жалоба отмечена рассмотренной')
      setReports((prev) => prev.filter((x) => x.id !== r.id))
    } catch {
      toast.error('Не удалось скрыть контент')
    } finally {
      setBusyId(null)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <Select value={status} onValueChange={(v) => setStatus(v ?? ALL)}>
        <SelectTrigger className="h-9 w-56 rounded-full">
          <SelectValue placeholder="Статус">{(v: string) => (v === ALL ? 'Все статусы' : STATUS_LABELS[v as ReportStatus])}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>Все статусы</SelectItem>
          {Object.values(ReportStatus).map((s) => (
            <SelectItem key={s} value={s}>
              {STATUS_LABELS[s]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {loading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-xl" />
          ))}
        </div>
      ) : reports.length === 0 ? (
        <p className="p-6 text-center text-sm text-muted-foreground">Жалоб по выбранному фильтру нет</p>
      ) : (
        <div className="flex flex-col gap-2">
          {reports.map((r) => (
            <div key={r.id} className="flex flex-col gap-2 rounded-2xl border border-border bg-card p-4">
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant="outline">{TARGET_LABELS[r.target_type ?? ''] ?? r.target_type}</Badge>
                <Badge variant={r.status === ReportStatus.pending ? 'default' : r.status === ReportStatus.reviewed ? 'secondary' : 'outline'}>
                  {STATUS_LABELS[r.status ?? ReportStatus.pending]}
                </Badge>
                {r.target_removed && (
                  <Badge variant="outline" className="text-muted-foreground">
                    Уже скрыто/удалено
                  </Badge>
                )}
                <span className="ml-auto text-xs text-muted-foreground">{formatDate(r.created_at)}</span>
              </div>

              {(r.target_title || r.target_snippet) && (
                <div className="rounded-xl bg-secondary/50 p-3 text-sm">
                  {r.target_title && <p className="font-medium text-foreground">{r.target_title}</p>}
                  {r.target_snippet && <p className="mt-0.5 line-clamp-3 text-muted-foreground">«{r.target_snippet}»</p>}
                  {targetLink(r) && (
                    <Link to={targetLink(r)!} className="mt-1.5 flex w-fit items-center gap-1 text-xs text-accent hover:underline">
                      <ExternalLink className="size-3.5" /> Перейти к {r.target_type === 'card' ? 'карточке' : 'обсуждению'}
                    </Link>
                  )}
                </div>
              )}

              <p className="text-sm text-foreground">
                <span className="text-muted-foreground">Причина жалобы:</span> {r.reason}
              </p>
              {r.resolution_note && <p className="text-xs text-muted-foreground">Заметка: {r.resolution_note}</p>}
              {r.status === ReportStatus.pending && (
                <div className="flex flex-wrap gap-2">
                  <Button
                    size="sm"
                    className="h-8 rounded-full"
                    disabled={busyId === r.id}
                    onClick={() => review(r.id!, PatchAdminReportsIdBodyStatus.reviewed)}
                  >
                    <Check className="size-3.5" /> Рассмотрено
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-8 rounded-full"
                    disabled={busyId === r.id}
                    onClick={() => review(r.id!, PatchAdminReportsIdBodyStatus.dismissed)}
                  >
                    <X className="size-3.5" /> Отклонить
                  </Button>
                  {!r.target_removed && (r.target_type === 'thread' || r.target_type === 'comment') && (
                    <Button
                      size="sm"
                      variant="outline"
                      className="h-8 rounded-full text-destructive hover:text-destructive"
                      disabled={busyId === r.id}
                      onClick={() => hideTarget(r)}
                    >
                      <EyeOff className="size-3.5" /> Скрыть контент
                    </Button>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
