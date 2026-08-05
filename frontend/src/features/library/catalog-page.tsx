import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Search, Upload } from 'lucide-react'
import { getLibraryTextbooks } from '@/api/generated/medFlowAPI'
import { GetLibraryTextbooksSort, TextbookStorageType } from '@/api/generated'
import type { TextbookListItem } from '@/api/generated'
import { Input } from '@/components/ui/input'
import { Button, buttonVariants } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { COURSES, SUBJECTS } from '@/lib/library'
import { TextbookCard } from './textbook-card'

const PAGE_SIZE = 12
const ALL = '__all__'

// SelectValue у base-ui по умолчанию показывает сырое value, а не текст
// выбранного SelectItem — подписи для триггера задаём явно.
const STORAGE_TYPE_LABELS: Record<string, string> = {
  [ALL]: 'Категория: все',
  [TextbookStorageType.A]: 'A — скачать PDF',
  [TextbookStorageType.B]: 'B — только ссылка',
}
const SORT_LABELS: Record<string, string> = {
  [GetLibraryTextbooksSort.created_at_desc]: 'Сначала новые',
  [GetLibraryTextbooksSort.title_asc]: 'Название А→Я',
  [GetLibraryTextbooksSort.title_desc]: 'Название Я→А',
}

export function LibraryCatalogPage() {
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [subject, setSubject] = useState(ALL)
  const [course, setCourse] = useState(ALL)
  const [storageType, setStorageType] = useState(ALL)
  const [sort, setSort] = useState<string>(GetLibraryTextbooksSort.created_at_desc)
  const [page, setPage] = useState(1)

  const [items, setItems] = useState<TextbookListItem[]>([])
  const [total, setTotal] = useState(0)
  const [hasNext, setHasNext] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const timeout = setTimeout(() => setDebouncedSearch(search), 300)
    return () => clearTimeout(timeout)
  }, [search])

  useEffect(() => setPage(1), [debouncedSearch, subject, course, storageType, sort])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    getLibraryTextbooks({
      q: debouncedSearch || undefined,
      subject: subject === ALL ? undefined : subject,
      course: course === ALL ? undefined : Number(course),
      storage_type: storageType === ALL ? undefined : (storageType as TextbookStorageType),
      sort: sort as GetLibraryTextbooksSort,
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
  }, [debouncedSearch, subject, course, storageType, sort, page])

  return (
    <div className="flex flex-col gap-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold text-primary">Библиотека учебников</h1>
          <p className="text-sm text-muted-foreground">
            Легальные источники: категория A — скачивание PDF, категория B — переход к первоисточнику
          </p>
        </div>
        <Link to="/library/upload" className={cn(buttonVariants({ variant: 'outline' }), 'h-9 rounded-full px-4')}>
          <Upload className="size-4" /> Мои материалы для ИИ
        </Link>
      </div>

      <div className="flex flex-col gap-3 rounded-2xl border border-border bg-card p-4">
        <div className="relative">
          <Search className="pointer-events-none absolute top-1/2 left-3.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Поиск по названию…"
            className="h-11 rounded-full pl-10"
          />
        </div>

        <div className="flex flex-wrap gap-2">
          <Select value={subject} onValueChange={(v) => setSubject(v ?? ALL)}>
            <SelectTrigger className="h-10 rounded-full">
              <SelectValue placeholder="Предмет">{(v: string) => (v === ALL ? 'Все предметы' : v)}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>Все предметы</SelectItem>
              {SUBJECTS.map((s) => (
                <SelectItem key={s} value={s}>{s}</SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={course} onValueChange={(v) => setCourse(v ?? ALL)}>
            <SelectTrigger className="h-10 w-32 rounded-full">
              <SelectValue placeholder="Курс">{(v: string) => (v === ALL ? 'Все курсы' : `${v} курс`)}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>Все курсы</SelectItem>
              {COURSES.map((c) => (
                <SelectItem key={c} value={String(c)}>{c} курс</SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={storageType} onValueChange={(v) => setStorageType(v ?? ALL)}>
            <SelectTrigger className="h-10 w-44 rounded-full">
              <SelectValue placeholder="Категория">
                {(v: string) => STORAGE_TYPE_LABELS[v] ?? 'Категория: все'}
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>Категория: все</SelectItem>
              <SelectItem value={TextbookStorageType.A}>A — скачать PDF</SelectItem>
              <SelectItem value={TextbookStorageType.B}>B — только ссылка</SelectItem>
            </SelectContent>
          </Select>

          <Select value={sort} onValueChange={(v) => setSort(v ?? GetLibraryTextbooksSort.created_at_desc)}>
            <SelectTrigger className="h-10 w-48 rounded-full ml-auto">
              <SelectValue placeholder="Сортировка">{(v: string) => SORT_LABELS[v]}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={GetLibraryTextbooksSort.created_at_desc}>Сначала новые</SelectItem>
              <SelectItem value={GetLibraryTextbooksSort.title_asc}>Название А→Я</SelectItem>
              <SelectItem value={GetLibraryTextbooksSort.title_desc}>Название Я→А</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {loading ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-44 rounded-2xl" />
          ))}
        </div>
      ) : items.length === 0 ? (
        <div className="flex flex-col items-center gap-2 rounded-2xl border border-dashed border-border py-16 text-center">
          <p className="font-medium text-foreground">Ничего не найдено</p>
          <p className="text-sm text-muted-foreground">Попробуйте изменить поиск или фильтры</p>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {items.map((t) => (
              <TextbookCard key={t.id} textbook={t} />
            ))}
          </div>

          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <span>Всего найдено: {total}</span>
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
