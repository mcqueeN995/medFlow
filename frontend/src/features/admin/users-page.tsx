import { useEffect, useState } from 'react'
import { Ban, ShieldCheck } from 'lucide-react'
import { toast } from 'sonner'
import { deleteAdminUsersIdBan, getAdminUsers, patchAdminUsersIdRole, postAdminUsersIdBan } from '@/api/generated/medFlowAPI'
import { UserRole } from '@/api/generated'
import type { AdminUser } from '@/api/generated'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'

const ALL = '__all__'

const ROLE_LABELS: Record<UserRole, string> = {
  [UserRole.guest]: 'Гость',
  [UserRole.user]: 'Пользователь',
  [UserRole.moderator]: 'Модератор',
  [UserRole.admin]: 'Админ',
}

export function AdminUsersPage() {
  const [q, setQ] = useState('')
  const [role, setRole] = useState<string>(ALL)
  const [banned, setBanned] = useState<string>(ALL)
  const [users, setUsers] = useState<AdminUser[]>([])
  const [loading, setLoading] = useState(true)
  const [busyId, setBusyId] = useState<string | null>(null)

  function load() {
    setLoading(true)
    getAdminUsers({
      q: q || undefined,
      role: role === ALL ? undefined : (role as UserRole),
      banned: banned === ALL ? undefined : banned === 'true',
      limit: 50,
    })
      .then((res) => setUsers(res.data ?? []))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    const t = setTimeout(load, 300)
    return () => clearTimeout(t)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q, role, banned])

  async function changeRole(id: string, newRole: UserRole) {
    setBusyId(id)
    try {
      const updated = await patchAdminUsersIdRole(id, { role: newRole })
      setUsers((prev) => prev.map((u) => (u.id === id ? updated : u)))
      toast.success('Роль изменена')
    } catch {
      toast.error('Не удалось изменить роль')
    } finally {
      setBusyId(null)
    }
  }

  async function ban(id: string) {
    const reason = window.prompt('Причина бана:')
    if (!reason?.trim()) return
    setBusyId(id)
    try {
      const updated = await postAdminUsersIdBan(id, { reason: reason.trim() })
      setUsers((prev) => prev.map((u) => (u.id === id ? updated : u)))
      toast.success('Пользователь забанен')
    } catch {
      toast.error('Не удалось забанить пользователя')
    } finally {
      setBusyId(null)
    }
  }

  async function unban(id: string) {
    setBusyId(id)
    try {
      const updated = await deleteAdminUsersIdBan(id)
      setUsers((prev) => prev.map((u) => (u.id === id ? updated : u)))
      toast.success('Пользователь разбанен')
    } catch {
      toast.error('Не удалось разбанить пользователя')
    } finally {
      setBusyId(null)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap gap-2">
        <Input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Поиск по нику или e-mail…"
          className="h-9 max-w-64 rounded-full"
        />
        <Select value={role} onValueChange={(v) => setRole(v ?? ALL)}>
          <SelectTrigger className="h-9 rounded-full">
            <SelectValue placeholder="Роль">{(v: string) => (v === ALL ? 'Все роли' : ROLE_LABELS[v as UserRole])}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>Все роли</SelectItem>
            {Object.values(UserRole).map((r) => (
              <SelectItem key={r} value={r}>
                {ROLE_LABELS[r]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={banned} onValueChange={(v) => setBanned(v ?? ALL)}>
          <SelectTrigger className="h-9 rounded-full">
            <SelectValue placeholder="Статус">
              {(v: string) => (v === ALL ? 'Все' : v === 'true' ? 'Забанены' : 'Активны')}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>Все</SelectItem>
            <SelectItem value="false">Активны</SelectItem>
            <SelectItem value="true">Забанены</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {loading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-xl" />
          ))}
        </div>
      ) : users.length === 0 ? (
        <p className="p-6 text-center text-sm text-muted-foreground">Никого не найдено</p>
      ) : (
        <div className="flex flex-col gap-2">
          {users.map((u) => (
            <div key={u.id} className="flex flex-wrap items-center gap-3 rounded-2xl border border-border bg-card p-3.5">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-foreground">{u.nickname}</span>
                  {u.banned_at && (
                    <Badge variant="destructive">
                      <Ban className="size-3" /> забанен
                    </Badge>
                  )}
                </div>
                <p className="truncate text-xs text-muted-foreground">{u.email}</p>
                {u.banned_at && u.ban_reason && (
                  <p className="text-xs text-destructive">Причина: {u.ban_reason}</p>
                )}
              </div>

              <Select value={u.role} onValueChange={(v) => v && changeRole(u.id!, v as UserRole)}>
                <SelectTrigger className="h-8 rounded-full" size="sm">
                  <SelectValue>{(v: UserRole) => ROLE_LABELS[v]}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {Object.values(UserRole).map((r) => (
                    <SelectItem key={r} value={r}>
                      {ROLE_LABELS[r]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              {u.banned_at ? (
                <Button size="sm" variant="outline" className="h-8 rounded-full" disabled={busyId === u.id} onClick={() => unban(u.id!)}>
                  <ShieldCheck className="size-3.5" /> Разбанить
                </Button>
              ) : (
                <Button size="sm" variant="outline" className="h-8 rounded-full text-destructive" disabled={busyId === u.id} onClick={() => ban(u.id!)}>
                  <Ban className="size-3.5" /> Забанить
                </Button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
