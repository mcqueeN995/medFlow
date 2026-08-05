import { Link } from 'react-router-dom'
import { Download, ExternalLink } from 'lucide-react'
import { TextbookStorageType } from '@/api/generated'
import type { TextbookListItem } from '@/api/generated'
import { Badge } from '@/components/ui/badge'
import { LICENSE_LABELS } from '@/lib/library'

export function TextbookCard({ textbook }: { textbook: TextbookListItem }) {
  const isStored = textbook.storage_type === TextbookStorageType.A

  return (
    <Link
      to={`/library/${textbook.id}`}
      className="flex flex-col gap-3 rounded-2xl border border-border bg-card p-5 shadow-sm transition-shadow hover:shadow-md"
    >
      <div className="flex items-start justify-between gap-2">
        <Badge
          variant={isStored ? 'default' : 'secondary'}
          className={isStored ? 'bg-accent text-accent-foreground' : ''}
        >
          {isStored ? (
            <>
              <Download className="size-3" /> Скачать PDF
            </>
          ) : (
            <>
              <ExternalLink className="size-3" /> Только ссылка
            </>
          )}
        </Badge>
        {textbook.license_type && (
          <span className="text-right text-xs text-muted-foreground">{LICENSE_LABELS[textbook.license_type]}</span>
        )}
      </div>

      <h3 className="line-clamp-2 font-semibold text-foreground">{textbook.title}</h3>

      {textbook.authors && <p className="line-clamp-1 text-sm text-muted-foreground">{textbook.authors}</p>}

      {(textbook.subject || textbook.course) && (
        <div className="mt-auto flex flex-wrap gap-1.5 pt-1">
          {textbook.subject && (
            <span className="rounded-full bg-secondary px-2.5 py-0.5 text-xs text-secondary-foreground">
              {textbook.subject}
            </span>
          )}
          {textbook.course && (
            <span className="rounded-full bg-secondary px-2.5 py-0.5 text-xs text-secondary-foreground">
              {textbook.course} курс
            </span>
          )}
        </div>
      )}
    </Link>
  )
}
