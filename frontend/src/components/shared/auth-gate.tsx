import { Link } from 'react-router-dom'
import { LockKeyhole } from 'lucide-react'
import { buttonVariants } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export function AuthGate({ title, description }: { title: string; description: string }) {
  return (
    <div className="flex flex-col items-center gap-3 p-16 text-center">
      <span className="flex size-12 items-center justify-center rounded-full bg-secondary text-accent">
        <LockKeyhole className="size-5" />
      </span>
      <h2 className="font-semibold text-foreground">{title}</h2>
      <p className="max-w-sm text-sm text-muted-foreground">{description}</p>
      <div className="mt-2 flex gap-2">
        <Link to="/login" className={cn(buttonVariants(), 'rounded-full bg-linear-to-r from-primary to-accent px-5 text-primary-foreground')}>
          Войти
        </Link>
        <Link to="/register" className={cn(buttonVariants({ variant: 'outline' }), 'rounded-full px-5')}>
          Зарегистрироваться
        </Link>
      </div>
    </div>
  )
}
