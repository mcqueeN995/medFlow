import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft, BrainCircuit, Heart } from 'lucide-react'
import { toast } from 'sonner'
import { deleteCardsIdFavorite, getCardsFavorites, postCardsIdStars, deleteCardsIdStars } from '@/api/generated/medFlowAPI'
import type { Card } from '@/api/generated'
import { Skeleton } from '@/components/ui/skeleton'
import { buttonVariants } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { StarRating } from './star-rating'

export function FavoritesPage() {
  const [cards, setCards] = useState<Card[] | null>(null)

  function load() {
    getCardsFavorites({}).then((res) => setCards(res.data ?? []))
  }

  useEffect(load, [])

  async function unfavorite(cardId: string) {
    try {
      await deleteCardsIdFavorite(cardId)
      setCards((prev) => prev?.filter((c) => c.id !== cardId) ?? prev)
    } catch {
      toast.error('Не удалось убрать из избранного')
    }
  }

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
      setCards((prev) => prev?.map((c) => (c.id === cardId ? { ...c, my_stars: undefined } : c)) ?? prev)
    } catch {
      toast.error('Не удалось убрать оценку')
    }
  }

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-5 p-6">
      <Link to="/cards" className="flex w-fit items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" /> К карточкам
      </Link>

      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-primary">Избранное</h1>
          <p className="text-sm text-muted-foreground">Карточки, которые вы отметили для изучения</p>
        </div>
        <Link
          to="/cards/review?scope=favorites"
          className={cn(buttonVariants(), 'h-9 rounded-full bg-linear-to-r from-primary to-accent px-4 text-primary-foreground')}
        >
          <BrainCircuit className="size-4" /> Повторить избранное
        </Link>
      </div>

      {cards === null ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-2xl" />
          ))}
        </div>
      ) : cards.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
          Пока нет избранных карточек — добавляйте их значком <Heart className="inline size-3.5" /> при просмотре набора
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {cards.map((card) => (
            <div key={card.id} className="rounded-2xl border border-border bg-card p-4">
              <p className="text-sm font-medium text-foreground">{card.question}</p>
              <p className="mt-1.5 text-sm text-muted-foreground">{card.answer}</p>
              <div className="mt-3 flex items-center justify-between gap-2">
                <StarRating
                  myStars={card.my_stars}
                  averageStars={card.average_stars}
                  ratingsCount={card.ratings_count}
                  onRate={(stars) => rateStars(card.id!, stars)}
                  onRemove={() => removeStars(card.id!)}
                />
                <button
                  type="button"
                  onClick={() => unfavorite(card.id!)}
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-destructive"
                >
                  <Heart className="size-3.5 fill-destructive text-destructive" /> Убрать
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
