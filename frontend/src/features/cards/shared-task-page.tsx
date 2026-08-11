import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Logo } from '@/components/shared/logo'
import { getCardsSharedToken } from '@/api/generated/medFlowAPI'
import type { SharedCardTask } from '@/api/generated'
import { Skeleton } from '@/components/ui/skeleton'

export function SharedTaskPage() {
  const { token } = useParams<{ token: string }>()
  const [task, setTask] = useState<SharedCardTask | null>(null)
  const [notFound, setNotFound] = useState(false)

  useEffect(() => {
    if (!token) return
    getCardsSharedToken(token)
      .then(setTask)
      .catch(() => setNotFound(true))
  }, [token])

  return (
    <div className="mx-auto flex min-h-svh max-w-2xl flex-col gap-5 p-6">
      <Link to="/" className="flex items-center gap-2">
        <Logo className="h-8 w-8" />
        <span className="font-bold text-primary">medFlow</span>
      </Link>

      {notFound ? (
        <div className="flex flex-col items-center gap-2 p-16 text-center text-sm text-muted-foreground">
          <p className="font-medium text-foreground">Ссылка недействительна</p>
          <p>Набор карточек не найден, скрыт автором или ещё не готов</p>
        </div>
      ) : !task ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-32 rounded-2xl" />
        </div>
      ) : (
        <>
          <div>
            <h1 className="text-xl font-bold text-foreground">{task.topic ?? 'Набор карточек'}</h1>
            <p className="text-sm text-muted-foreground">Поделились набором из {task.cards?.length ?? 0} карточек</p>
          </div>
          <div className="flex flex-col gap-3">
            {task.cards?.map((card) => (
              <div key={card.id} className="rounded-2xl border border-border bg-card p-4">
                <p className="text-sm font-medium text-foreground">{card.question}</p>
                <p className="mt-1.5 text-sm text-muted-foreground">{card.answer}</p>
              </div>
            ))}
          </div>
          <p className="text-center text-xs text-muted-foreground">
            Хотите генерировать свои карточки?{' '}
            <Link to="/register" className="text-accent underline underline-offset-2">
              Зарегистрируйтесь в medFlow
            </Link>
          </p>
        </>
      )}
    </div>
  )
}
