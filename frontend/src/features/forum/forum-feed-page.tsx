import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus } from 'lucide-react'
import { getThreads } from '@/api/generated/medFlowAPI'
import { GetThreadsSort, ThreadTag } from '@/api/generated'
import type { ThreadListItem } from '@/api/generated'
import { Button, buttonVariants } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { THREAD_TAG_LABELS } from '@/lib/forum'
import { ThreadCard } from './thread-card'

const PAGE_SIZE = 10
const ALL = '__all__'

const SORT_LABELS: Record<string, string> = {
  [GetThreadsSort.created_at_desc]: 'Сначала новые',
  [GetThreadsSort.popular]: 'Популярное',
}

export function ForumFeedPage() {
  const [tag, setTag] = useState<string>(ALL)
  const [sort, setSort] = useState<string>(GetThreadsSort.created_at_desc)
  const [page, setPage] = useState(1)

  const [items, setItems] = useState<ThreadListItem[]>([])
  const [total, setTotal] = useState(0)
  const [hasNext, setHasNext] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => setPage(1), [tag, sort])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    getThreads({
      tag: tag === ALL ? undefined : (tag as ThreadTag),
      sort: sort as GetThreadsSort,
      page,
      limit: PAGE_SIZE,
    })
      .then((res) => {
        if (cancelled) return
        setItems(res.data ?? [])
        setTotal(res.pagination?.total ?? 0)
        setHasNext(res.pagination?.has_next ?? false)
      })
      .finally(() => !cancelled && setLoading(false))
    return () => {
      cancelled = true
    }
  }, [tag, sort, page])

  return (
    <div className="flex flex-col gap-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold text-primary">Треды</h1>
          <p className="text-sm text-muted-foreground">Обсуждения, вопросы и барахолка для студентов-медиков</p>
        </div>
        <Link to="/forum/create" className={cn(buttonVariants(), 'h-10 rounded-full bg-linear-to-r from-primary to-accent px-4 text-primary-foreground')}>
          <Plus className="size-4" /> Новый тред
        </Link>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => setTag(ALL)}
          className={cn(
            'rounded-full px-3 py-1.5 text-sm font-medium transition-colors',
            tag === ALL ? 'bg-primary text-primary-foreground' : 'bg-secondary text-secondary-foreground hover:bg-muted',
          )}
        >
          Все теги
        </button>
        {Object.values(ThreadTag).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTag(t)}
            className={cn(
              'rounded-full px-3 py-1.5 text-sm font-medium transition-colors',
              tag === t ? 'bg-primary text-primary-foreground' : 'bg-secondary text-secondary-foreground hover:bg-muted',
            )}
          >
            {THREAD_TAG_LABELS[t]}
          </button>
        ))}

        <Select value={sort} onValueChange={(v) => setSort(v ?? GetThreadsSort.created_at_desc)}>
          <SelectTrigger className="ml-auto h-9 w-44 rounded-full">
            <SelectValue placeholder="Сортировка">{(v: string) => SORT_LABELS[v]}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={GetThreadsSort.created_at_desc}>Сначала новые</SelectItem>
            <SelectItem value={GetThreadsSort.popular}>Популярное</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {loading ? (
        <div className="flex flex-col gap-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-32 rounded-2xl" />
          ))}
        </div>
      ) : items.length === 0 ? (
        <div className="flex flex-col items-center gap-2 rounded-2xl border border-dashed border-border py-16 text-center">
          <p className="font-medium text-foreground">Пока пусто</p>
          <p className="text-sm text-muted-foreground">Будьте первым, кто создаст тред</p>
        </div>
      ) : (
        <>
          <div className="flex flex-col gap-3">
            {items.map((t) => (
              <ThreadCard key={t.id} thread={t} />
            ))}
          </div>

          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <span>Всего тредов: {total}</span>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                Назад
              </Button>
              <span className="flex items-center px-2">Стр. {page}</span>
              <Button variant="outline" size="sm" disabled={!hasNext} onClick={() => setPage((p) => p + 1)}>
                Вперёд
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
