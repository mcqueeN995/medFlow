import { NavLink, Outlet } from 'react-router-dom'
import { ChevronLeft, ChevronRight, LogIn } from 'lucide-react'
import { cn } from '@/lib/utils'
import { navItems } from '@/lib/nav-items'
import { Logo } from '@/components/shared/logo'
import { ThemeToggle } from '@/components/shared/theme-toggle'
import { useAuthStore } from '@/stores/auth-store'
import { useUIStore } from '@/stores/ui-store'

export function AppShell() {
  const isGuest = useAuthStore((s) => !s.accessToken)
  const collapsed = useUIStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useUIStore((s) => s.toggleSidebar)

  const items = navItems.map((item) =>
    item.to === '/profile' && isGuest ? { ...item, label: 'Войти', to: '/login', icon: LogIn } : item,
  )

  return (
    <div className="flex min-h-svh flex-col bg-background md:flex-row">
      <aside
        className={cn(
          'sticky top-3 my-3 ml-3 hidden h-[calc(100svh-1.5rem)] shrink-0 flex-col rounded-3xl border border-sidebar-border/60 bg-sidebar/70 p-3 text-sidebar-foreground shadow-lg backdrop-blur-2xl transition-[width] duration-200 ease-out md:flex',
          collapsed ? 'w-[76px]' : 'w-64',
        )}
      >
        <div className={cn('mb-4 flex items-center gap-2 px-2 py-2', collapsed && 'justify-center px-0')}>
          <Logo className="size-8 shrink-0" />
          {!collapsed && <span className="text-xl font-bold text-sidebar-primary">medFlow</span>}
        </div>

        <nav className="flex flex-1 flex-col gap-1">
          {items.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              title={collapsed ? label : undefined}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 rounded-2xl px-4 py-3 text-sm font-medium text-sidebar-foreground/70 transition-colors hover:bg-sidebar-accent/15 hover:text-sidebar-foreground',
                  isActive && 'bg-sidebar-accent/15 text-sidebar-primary',
                  collapsed && 'justify-center px-0',
                )
              }
            >
              <Icon className="size-5 shrink-0" />
              {!collapsed && label}
            </NavLink>
          ))}
        </nav>

        <div className="flex flex-col gap-1 border-t border-sidebar-border/60 pt-2">
          <ThemeToggle collapsed={collapsed} />
          <button
            type="button"
            onClick={toggleSidebar}
            title={collapsed ? 'Развернуть панель' : 'Свернуть панель'}
            className={cn(
              'flex items-center gap-3 rounded-2xl px-4 py-3 text-sm font-medium text-sidebar-foreground/70 transition-colors hover:bg-sidebar-accent/15 hover:text-sidebar-foreground',
              collapsed && 'justify-center px-0',
            )}
          >
            {collapsed ? <ChevronRight className="size-5 shrink-0" /> : <ChevronLeft className="size-5 shrink-0" />}
            {!collapsed && 'Свернуть'}
          </button>
        </div>
      </aside>

      <main className="flex-1 pb-20 md:pb-0">
        <Outlet />
      </main>

      <nav className="fixed inset-x-0 bottom-0 z-10 flex border-t border-border bg-card/95 backdrop-blur-sm md:hidden">
        {items.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              cn(
                'flex flex-1 flex-col items-center gap-1 py-2.5 text-[11px] font-medium text-muted-foreground',
                isActive && 'text-primary',
              )
            }
          >
            <Icon className="size-6" />
            {label}
          </NavLink>
        ))}
      </nav>
    </div>
  )
}
