import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft, Layers, Search } from 'lucide-react'
import { getCardsCatalog } from '@/api/generated/medFlowAPI'
import { CardDifficulty } from '@/api/generated'
import type { CardCatalogEntry } from '@/api/generated'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'

const DIFFICULTY_LABELS: Record<CardDifficulty, string> = {
  [CardDifficulty.easy]: 'Лёгкая',
  [CardDifficulty.medium]: 'Средняя',
  [CardDifficulty.hard]: 'Сложная',
}

function formatDate(iso?: string): string {
  if (!iso) return ''
  return new Intl.DateTimeFormat('ru-RU', { day: '2-digit', month: 'short' }).format(new Date(iso))
}

export function CatalogFeedPage() {
  const [entries, setEntries] = useState<CardCatalogEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [q, setQ] = useState('')

  useEffect(() => {
    setLoading(true)
    const handle = setTimeout(() => {
      getCardsCatalog({ q: q || undefined })
        .then((res) => setEntries(res.data ?? []))
        .finally(() => setLoading(false))
    }, 250)
    return () => clearTimeout(handle)
  }, [q])

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-5 p-6">
      <Link to="/cards" className="flex w-fit items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" /> К карточкам
      </Link>

      <div>
        <h1 className="text-xl font-bold text-primary">Каталог карточек</h1>
        <p className="text-sm text-muted-foreground">Уже сгенерированные наборы по учебникам — переиспользуйте вместо генерации заново</p>
      </div>

      <div className="relative">
        <Search className="absolute left-4 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Поиск по теме или учебнику…"
          className="h-11 rounded-full pl-10"
        />
      </div>

      {loading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-xl" />
          ))}
        </div>
      ) : entries.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
          {q ? 'Ничего не найдено' : 'В каталоге пока нет готовых наборов'}
        </div>
      ) : (
        <ul className="flex flex-col gap-2">
          {entries.map((entry) => (
            <li key={entry.task_id}>
              <Link
                to={`/cards/tasks/${entry.task_id}`}
                className="flex items-center justify-between gap-3 rounded-xl border border-border bg-card p-3.5 hover:bg-secondary"
              >
                <div className="flex min-w-0 items-center gap-3">
                  <Layers className="size-4 shrink-0 text-accent" />
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-foreground">{entry.topic ?? entry.textbook_title}</p>
                    <p className="truncate text-xs text-muted-foreground">
                      {entry.textbook_title} · {entry.cards_count} карточек · {formatDate(entry.created_at)}
                    </p>
                  </div>
                </div>
                <span className="shrink-0 rounded-full bg-secondary px-2.5 py-0.5 text-xs text-secondary-foreground">
                  {DIFFICULTY_LABELS[entry.difficulty ?? 'medium']}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
