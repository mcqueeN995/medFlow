import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, Download, ExternalLink, ShieldCheck } from 'lucide-react'
import { getLibraryTextbooksId } from '@/api/generated/medFlowAPI'
import { TextbookStorageType } from '@/api/generated'
import type { Textbook } from '@/api/generated'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { LICENSE_LABELS, formatDate } from '@/lib/library'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'
import { useDownloadTextbook } from './use-download-textbook'

const API_URL = import.meta.env.VITE_API_URL ?? '/api/v1'

export function TextbookDetailsPage() {
  const { id } = useParams<{ id: string }>()
  const [textbook, setTextbook] = useState<Textbook | null>(null)
  const [loading, setLoading] = useState(true)
  const [notFound, setNotFound] = useState(false)
  const { download, downloadingId } = useDownloadTextbook()
  const isGuest = useAuthStore((s) => !s.accessToken)

  useEffect(() => {
    if (!id) return
    setLoading(true)
    setNotFound(false)
    getLibraryTextbooksId(id)
      .then(setTextbook)
      .catch(() => setNotFound(true))
      .finally(() => setLoading(false))
  }, [id])

  if (loading) {
    return (
      <div className="flex flex-col gap-4 p-6">
        <Skeleton className="h-6 w-24" />
        <Skeleton className="h-64 rounded-2xl" />
      </div>
    )
  }

  if (notFound || !textbook) {
    return (
      <div className="flex flex-col items-center gap-3 p-16 text-center">
        <p className="font-medium text-foreground">Учебник не найден</p>
        <Link to="/library" className="text-sm text-accent underline underline-offset-2">
          Вернуться в каталог
        </Link>
      </div>
    )
  }

  const isStored = textbook.storage_type === TextbookStorageType.A
  const metaRows: [string, string | number | undefined][] = isStored
    ? [
        ['Авторы', textbook.authors],
        ['ISBN', textbook.isbn],
        ['Год издания', textbook.year],
        ['Страниц', textbook.pages],
        ['Предмет', textbook.subject],
        ['Курс', textbook.course ? `${textbook.course} курс` : undefined],
        ['Кафедра', textbook.department],
      ]
    : []

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-5 p-6">
      <Link to="/library" className="flex w-fit items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" /> К каталогу
      </Link>

      <div className="flex flex-col gap-4 rounded-2xl border border-border bg-card p-6">
        <Badge
          variant={isStored ? 'default' : 'secondary'}
          className={cn('w-fit', isStored && 'bg-accent text-accent-foreground')}
        >
          {isStored ? 'Категория A — хранится в библиотеке' : 'Категория B — только ссылка на источник'}
        </Badge>

        <h1 className="text-xl font-bold text-foreground">{textbook.title}</h1>

        {isStored && textbook.description && <p className="text-sm text-muted-foreground">{textbook.description}</p>}

        {metaRows.some(([, v]) => v) && (
          <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
            {metaRows
              .filter(([, v]) => v)
              .map(([label, value]) => (
                <div key={label} className="contents">
                  <dt className="text-muted-foreground">{label}</dt>
                  <dd className="text-foreground">{value}</dd>
                </div>
              ))}
          </dl>
        )}

        {textbook.license_type && (
          <div className="flex items-start gap-2 rounded-xl bg-secondary p-3 text-xs text-secondary-foreground">
            <ShieldCheck className="mt-0.5 size-4 shrink-0" />
            <span>
              Лицензия: <strong>{LICENSE_LABELS[textbook.license_type]}</strong>
              {textbook.copyright_holder && <> · Правообладатель: {textbook.copyright_holder}</>}
            </span>
          </div>
        )}

        <p className="text-xs text-muted-foreground">Добавлено в каталог: {formatDate(textbook.created_at)}</p>

        {isStored ? (
          isGuest ? (
            <Link
              to="/login"
              className="inline-flex h-11 items-center justify-center gap-1.5 rounded-full border border-border text-sm font-medium text-muted-foreground hover:text-foreground"
            >
              <Download className="size-4" /> Войдите, чтобы скачать
            </Link>
          ) : (
            <Button
              className="h-11 rounded-full bg-linear-to-r from-primary to-accent text-primary-foreground"
              disabled={downloadingId === id}
              onClick={() => id && download(id, textbook.title ?? 'textbook')}
            >
              <Download className="size-4" />
              {downloadingId === id ? 'Скачивание…' : 'Скачать PDF'}
            </Button>
          )
        ) : (
          <a
            href={`${API_URL}/library/textbooks/${id}/source`}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex h-11 items-center justify-center gap-1.5 rounded-full bg-linear-to-r from-primary to-accent px-4 text-sm font-medium text-primary-foreground shadow-md hover:opacity-95"
          >
            <ExternalLink className="size-4" /> Перейти к источнику
          </a>
        )}
      </div>
    </div>
  )
}
