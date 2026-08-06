import { Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import type { ReactNode } from 'react'

export function LegalPage({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-5 p-6">
      <Link to="/login" className="flex w-fit items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" /> Назад
      </Link>
      <h1 className="text-2xl font-bold text-primary">{title}</h1>
      <div className="flex flex-col gap-4 text-sm leading-relaxed text-foreground">{children}</div>
    </div>
  )
}
