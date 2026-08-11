import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { toast } from 'sonner'
import { getCardsRated, postCardsIdStars, deleteCardsIdStars } from '@/api/generated/medFlowAPI'
import type { Card } from '@/api/generated'
import { Skeleton } from '@/components/ui/skeleton'
import { StarRating } from '@/features/cards/star-rating'

export function RatedCardsPage() {
  const [cards, setCards] = useState<Card[] | null>(null)

  useEffect(() => {
    getCardsRated({}).then((res) => setCards(res.data ?? []))
  }, [])

  async function rateStars(cardId: string, stars: number) {
    try {
      await postCardsIdStars(cardId, { stars })
      setCards((prev) => prev?.map((c) => (c.id === cardId ? { ...c, my_stars: stars } : c)) ?? prev)
    } catch {
      toast.error('Не удалось сохранить оценку')
    }
  }

  async function removeStars(cardId: string) {
    try {
      await deleteCardsIdStars(cardId)
      setCards((prev) => prev?.filter((c) => c.id !== cardId) ?? prev)
    } catch {
      toast.error('Не удалось убрать оценку')
    }
  }

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-5 p-6">
      <Link to="/profile" className="flex w-fit items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" /> К профилю
      </Link>

      <div>
        <h1 className="text-xl font-bold text-primary">Оценённые карточки</h1>
        <p className="text-sm text-muted-foreground">Карточки, которым вы поставили звёздный рейтинг</p>
      </div>

      {cards === null ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-2xl" />
          ))}
        </div>
      ) : cards.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
          Вы пока не оценили ни одной карточки
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {cards.map((card) => (
            <div key={card.id} className="rounded-2xl border border-border bg-card p-4">
              <p className="text-sm font-medium text-foreground">{card.question}</p>
              <p className="mt-1.5 text-sm text-muted-foreground">{card.answer}</p>
              <div className="mt-3">
                <StarRating
                  myStars={card.my_stars}
                  averageStars={card.average_stars}
                  ratingsCount={card.ratings_count}
                  onRate={(stars) => rateStars(card.id!, stars)}
                  onRemove={() => removeStars(card.id!)}
                />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
