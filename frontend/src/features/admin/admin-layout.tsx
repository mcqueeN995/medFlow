import { NavLink, Outlet } from 'react-router-dom'
import { AreaChart, ScrollText, ShieldCheck, TriangleAlert, Users } from 'lucide-react'
import { useAuthStore } from '@/stores/auth-store'
import { UserRole } from '@/api/generated'
import { cn } from '@/lib/utils'

// Жалобы видны moderator+; Пользователи/Статистика/Аудит-лог - только admin
// (бэкенд эти эндпоинты и так защищает RequireRole, вкладки здесь просто не
// показывают то, что всё равно ответит 403).
const TABS = [
  { to: '/admin/reports', label: 'Жалобы', icon: TriangleAlert, adminOnly: false },
  { to: '/admin/users', label: 'Пользователи', icon: Users, adminOnly: true },
  { to: '/admin/stats', label: 'Статистика', icon: AreaChart, adminOnly: true },
  { to: '/admin/audit-log', label: 'Аудит-лог', icon: ScrollText, adminOnly: true },
] as const

export function AdminLayout() {
  const role = useAuthStore((s) => s.user?.role)
  const isAdmin = role === UserRole.admin
  const tabs = TABS.filter((t) => !t.adminOnly || isAdmin)

  return (
    <div className="mx-auto flex max-w-4xl flex-col gap-5 p-6">
      <div className="flex items-center gap-2">
        <ShieldCheck className="size-6 text-accent" />
        <h1 className="text-2xl font-bold text-primary">Админ-панель</h1>
      </div>

      <nav className="flex flex-wrap gap-1.5 border-b border-border pb-3">
        {tabs.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-1.5 rounded-full px-3.5 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground',
                isActive && 'bg-secondary text-foreground',
              )
            }
          >
            <Icon className="size-4" /> {label}
          </NavLink>
        ))}
      </nav>

      <Outlet />
    </div>
  )
}
