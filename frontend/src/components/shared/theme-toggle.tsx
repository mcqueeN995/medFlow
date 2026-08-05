import { useEffect, useState } from 'react'
import { Contrast, Moon, Sun } from 'lucide-react'
import { useTheme } from 'next-themes'
import { cn } from '@/lib/utils'

const THEMES = [
  { value: 'light', icon: Sun, label: 'Светлая тема' },
  { value: 'dim', icon: Contrast, label: 'Умеренная тема' },
  { value: 'dark', icon: Moon, label: 'Тёмная тема' },
] as const

const STEP = 40 // ширина кнопки (36px, size-9) + gap (4px, gap-1)

export function ThemeToggle({ collapsed }: { collapsed?: boolean }) {
  const { theme, resolvedTheme, setTheme } = useTheme()
  // theme недоступен до монтирования (next-themes читает localStorage на
  // клиенте) - без этой защиты бегунок на миг прыгает не в ту позицию.
  const [mounted, setMounted] = useState(false)
  useEffect(() => setMounted(true), [])

  const active = mounted ? (theme === 'system' ? resolvedTheme : theme) : 'light'
  const activeIndex = Math.max(0, THEMES.findIndex((t) => t.value === active))

  return (
    <div
      className={cn(
        'relative flex gap-1 rounded-2xl bg-sidebar-accent/10 p-1',
        // justify-center вместе с absolute-позиционированным бегунком ломает
        // его "статическую позицию" (CSS считает её как если бы центрирование
        // применялось и к нему) - вместо этого центрируем сам блок целиком.
        collapsed ? 'flex-col' : 'mx-auto w-fit flex-row',
      )}
    >
      <div
        className="absolute size-9 rounded-xl bg-sidebar-accent/25 shadow-sm transition-transform duration-200 ease-out"
        style={{ transform: collapsed ? `translateY(${activeIndex * STEP}px)` : `translateX(${activeIndex * STEP}px)` }}
        aria-hidden="true"
      />
      {THEMES.map(({ value, icon: Icon, label }) => (
        <button
          key={value}
          type="button"
          onClick={() => setTheme(value)}
          title={label}
          aria-label={label}
          className={cn(
            'relative z-10 flex size-9 shrink-0 items-center justify-center rounded-xl text-sidebar-foreground/55 transition-colors hover:text-sidebar-foreground',
            mounted && active === value && 'text-sidebar-primary',
          )}
        >
          <Icon className="size-4" />
        </button>
      ))}
    </div>
  )
}
