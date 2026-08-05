import type { ReactNode } from 'react'
import { Logo } from '@/components/shared/logo'

export function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-svh items-center justify-center bg-linear-to-b from-secondary to-background px-4 py-10">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center gap-3 text-center">
          <Logo className="h-16 w-16" />
          <h1 className="text-4xl font-bold tracking-tight text-primary">medFlow</h1>
          <p className="text-sm text-muted-foreground">
            твой интеллектуальный помощник в мире медицины
          </p>
        </div>
        <div className="rounded-3xl border border-border bg-card/80 p-6 shadow-lg backdrop-blur-sm">
          {children}
        </div>
      </div>
    </div>
  )
}
