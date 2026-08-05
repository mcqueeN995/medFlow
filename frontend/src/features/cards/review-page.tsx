import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft, CheckCircle2, PartyPopper, ShieldAlert } from 'lucide-react'
import { getCardsReview, postCardsIdRate } from '@/api/generated/medFlowAPI'
import type { ReviewCard } from '@/api/generated'
import { Button, buttonVariants } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { GRADE_LABELS } from '@/lib/sm2'

const GRADE_STYLES: Record<number, string> = {
  0: 'bg-destructive/10 text-destructive hover:bg-destructive/20',
  1: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
  2: 'bg-[color-mix(in_oklch,var(--accent)_18%,transparent)] text-accent hover:bg-[color-mix(in_oklch,var(--accent)_28%,transparent)]',
  3: 'bg-accent text-accent-foreground hover:opacity-90',
}

export function ReviewPage() {
  const [batch, setBatch] = useState<ReviewCard[] | null>(null)
  const [index, setIndex] = useState(0)
  const [revealed, setRevealed] = useState(false)
  const [rating, setRating] = useState(false)
  const [reviewedCount, setReviewedCount] = useState(0)

  useEffect(() => {
    getCardsReview({ limit: 20 }).then((res) => setBatch(res.data ?? []))
  }, [])

  async function grade(value: number) {
    const card = batch?.[index]
    if (!card?.card_id || rating) return
    setRating(true)
    try {
      await postCardsIdRate(card.card_id, { grade: value })
      setReviewedCount((c) => c + 1)
      setIndex((i) => i + 1)
      setRevealed(false)
    } finally {
      setRating(false)
    }
  }

  if (batch === null) {
    return (
      <div className="mx-auto flex max-w-lg flex-col gap-3 p-6">
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-56 rounded-2xl" />
      </div>
    )
  }

  const current = batch[index]

  return (
    <div className="mx-auto flex max-w-lg flex-col gap-5 p-6">
      <Link to="/cards" className="flex w-fit items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" /> К карточкам
      </Link>

      {!current ? (
        <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-border p-12 text-center">
          {reviewedCount > 0 ? (
            <>
              <PartyPopper className="size-8 text-accent" />
              <p className="font-medium text-foreground">Сессия завершена</p>
              <p className="text-sm text-muted-foreground">Проверено карточек: {reviewedCount}</p>
            </>
          ) : (
            <>
              <CheckCircle2 className="size-8 text-accent" />
              <p className="font-medium text-foreground">Нечего повторять</p>
              <p className="text-sm text-muted-foreground">Все карточки повторены — загляните позже</p>
            </>
          )}
          <Link to="/cards" className={cn(buttonVariants({ variant: 'outline' }), 'mt-2 rounded-full')}>
            К списку карточек
          </Link>
        </div>
      ) : (
        <>
          <p className="text-center text-xs text-muted-foreground">
            Карточка {index + 1} из {batch.length}
          </p>

          <div className="flex min-h-64 flex-col justify-between gap-4 rounded-2xl border border-border bg-card p-6">
            <div>
              <p className="text-lg font-medium text-foreground">{current.question}</p>
              {revealed && <p className="mt-4 text-muted-foreground">{current.answer}</p>}
            </div>
            {revealed && (
              <div className="flex items-start gap-1.5 rounded-lg bg-secondary p-2 text-xs text-secondary-foreground">
                <ShieldAlert className="mt-0.5 size-3.5 shrink-0" />
                <span>{current.disclaimer ?? 'Сгенерировано ИИ. Проверьте по источнику.'}</span>
              </div>
            )}
          </div>

          {!revealed ? (
            <Button onClick={() => setRevealed(true)} className="h-11 rounded-full">
              Показать ответ
            </Button>
          ) : (
            <div className="grid grid-cols-4 gap-2">
              {[0, 1, 2, 3].map((g) => (
                <button
                  key={g}
                  disabled={rating}
                  onClick={() => grade(g)}
                  className={cn('rounded-xl py-2.5 text-sm font-medium transition-colors disabled:opacity-50', GRADE_STYLES[g])}
                >
                  {GRADE_LABELS[g]}
                </button>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}
