import { Star } from 'lucide-react'
import { cn } from '@/lib/utils'

interface StarRatingProps {
  myStars?: number | null
  averageStars?: number | null
  ratingsCount?: number | null
  onRate: (stars: number) => void
  onRemove: () => void
  disabled?: boolean
}

// StarRating - клик по уже выбранной звезде убирает оценку (toggle), клик по
// другой - меняет её (backend делает ON CONFLICT DO UPDATE одним запросом).
export function StarRating({ myStars, averageStars, ratingsCount, onRate, onRemove, disabled }: StarRatingProps) {
  return (
    <div className="flex items-center gap-2 text-xs text-muted-foreground">
      <div className="flex items-center gap-0.5">
        {[1, 2, 3, 4, 5].map((n) => (
          <button
            key={n}
            type="button"
            disabled={disabled}
            onClick={() => (myStars === n ? onRemove() : onRate(n))}
            aria-label={`Оценить на ${n} ${n === 1 ? 'звезду' : n < 5 ? 'звезды' : 'звёзд'}`}
            className="disabled:opacity-50"
          >
            <Star className={cn('size-4', myStars && n <= myStars ? 'fill-amber-400 text-amber-400' : 'text-muted-foreground')} />
          </button>
        ))}
      </div>
      {averageStars != null && ratingsCount ? (
        <span>
          {averageStars.toFixed(1)} ({ratingsCount})
        </span>
      ) : null}
    </div>
  )
}
