import { Outlet } from 'react-router-dom'
import { ShieldAlert } from 'lucide-react'
import { useAuthStore } from '@/stores/auth-store'
import type { UserRole } from '@/api/generated'
import { AuthGate } from './auth-gate'

// Тот же паттерн, что и RequireAuthOutlet (см. соседний файл): не редиректит
// на /login, а показывает объяснение внутри AppShell. Гостя (нет accessToken)
// пускаем через AuthGate (там уместны кнопки "Войти"/"Зарегистрироваться");
// для уже вошедшего, но не модератора/админа, эти кнопки только сбивают с
// толку - вход тут не поможет, поэтому свой, более простой блок.
export function RequireRoleOutlet({ roles }: { roles: UserRole[] }) {
  const accessToken = useAuthStore((s) => s.accessToken)
  const role = useAuthStore((s) => s.user?.role)

  if (!accessToken) {
    return <AuthGate title="Админ-панель" description="Войдите под учётной записью модератора или администратора." />
  }
  if (!role || !roles.includes(role)) {
    return (
      <div className="flex flex-col items-center gap-3 p-16 text-center">
        <span className="flex size-12 items-center justify-center rounded-full bg-secondary text-destructive">
          <ShieldAlert className="size-5" />
        </span>
        <h2 className="font-semibold text-foreground">Недостаточно прав</h2>
        <p className="max-w-sm text-sm text-muted-foreground">
          Этот раздел доступен только модераторам и администраторам.
        </p>
      </div>
    )
  }
  return <Outlet />
}
