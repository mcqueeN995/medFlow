import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Bell, GraduationCap, LogOut, Mail, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { deleteUsersMe, getUsersMe, patchPushPreferences, patchUsersMe } from '@/api/generated/medFlowAPI'
import { University, UserRole } from '@/api/generated'
import type { PushPreferences, UserProfile } from '@/api/generated'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { useAuthStore } from '@/stores/auth-store'
import { formatDate } from '@/lib/library'
import { getCurrentPushSubscription, isPushSupported, subscribeToPush, unsubscribeFromPush } from '@/lib/push'

const PUSH_PREFERENCE_LABELS: { key: keyof PushPreferences; label: string }[] = [
  { key: 'thread_reply', label: 'Ответы в темах' },
  { key: 'comment_reply', label: 'Ответы на комментарии' },
  { key: 'reaction', label: 'Реакции' },
  { key: 'card_task_done', label: 'Карточки готовы' },
  { key: 'card_task_failed', label: 'Ошибка генерации карточек' },
  { key: 'moderation_action', label: 'Действия модерации' },
  { key: 'system', label: 'Системные уведомления' },
]

const UNIVERSITY_LABELS: Record<string, string> = {
  [University.sechenov]: 'Сеченовский университет',
  [University.pirogov]: 'РНИМУ им. Пирогова',
  [University.evdokimov]: 'МГМСУ им. Евдокимова',
  [University.other]: 'Другой вуз',
}

const ROLE_LABELS: Record<UserRole, string> = {
  [UserRole.guest]: 'Гость',
  [UserRole.user]: 'Студент',
  [UserRole.moderator]: 'Модератор',
  [UserRole.admin]: 'Администратор',
}

export function ProfilePage() {
  const navigate = useNavigate()
  const setUser = useAuthStore((s) => s.setUser)
  const logout = useAuthStore((s) => s.logout)

  const [profile, setProfile] = useState<UserProfile | null>(null)
  const [loading, setLoading] = useState(true)

  const [nickname, setNickname] = useState('')
  const [university, setUniversity] = useState('')
  const [course, setCourse] = useState('')
  const [faculty, setFaculty] = useState('')
  const [saving, setSaving] = useState(false)

  const [deleting, setDeleting] = useState(false)
  const [deletePassword, setDeletePassword] = useState('')
  const [deleteSubmitting, setDeleteSubmitting] = useState(false)

  const [pushSubscribed, setPushSubscribed] = useState(false)
  const [pushBusy, setPushBusy] = useState(false)
  const [pushPrefs, setPushPrefs] = useState<PushPreferences | null>(null)

  useEffect(() => {
    getUsersMe()
      .then((p) => {
        setProfile(p)
        setNickname(p.nickname ?? '')
        setUniversity(p.university ?? '')
        setCourse(p.course ? String(p.course) : '')
        setFaculty(p.faculty ?? '')
      })
      .finally(() => setLoading(false))

    if (isPushSupported()) {
      getCurrentPushSubscription().then((sub) => {
        setPushSubscribed(!!sub)
        // PATCH с пустым телом ничего не меняет (все поля опциональны), но
        // возвращает актуальные настройки - отдельного GET-эндпоинта для
        // чтения preferences в контракте нет.
        if (sub) patchPushPreferences({}).then(setPushPrefs)
      })
    }
  }, [])

  async function onTogglePush() {
    setPushBusy(true)
    try {
      if (pushSubscribed) {
        await unsubscribeFromPush()
        setPushSubscribed(false)
        setPushPrefs(null)
        toast.success('Push-уведомления отключены')
      } else {
        await subscribeToPush()
        setPushSubscribed(true)
        toast.success('Push-уведомления включены')
      }
    } catch {
      toast.error('Не удалось изменить подписку на push-уведомления')
    } finally {
      setPushBusy(false)
    }
  }

  async function onTogglePreference(key: keyof PushPreferences, value: boolean) {
    try {
      const updated = await patchPushPreferences({ [key]: value })
      setPushPrefs(updated)
    } catch {
      toast.error('Не удалось сохранить настройку')
    }
  }

  async function onSave(e: React.FormEvent) {
    e.preventDefault()
    if (nickname.trim().length < 3) {
      toast.error('Никнейм — минимум 3 символа')
      return
    }
    setSaving(true)
    try {
      const updated = await patchUsersMe({
        nickname: nickname.trim(),
        university: (university as University) || undefined,
        course: course ? Number(course) : undefined,
        faculty: faculty.trim() || undefined,
      })
      setProfile(updated)
      setUser(updated)
      toast.success('Профиль обновлён')
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 409) toast.error('Такой никнейм уже занят')
      else toast.error('Не удалось сохранить изменения')
    } finally {
      setSaving(false)
    }
  }

  async function onConfirmDelete() {
    if (!deletePassword) {
      toast.error('Введите пароль для подтверждения')
      return
    }
    setDeleteSubmitting(true)
    try {
      await deleteUsersMe({ password: deletePassword })
      toast.success('Аккаунт удалён')
      logout()
      navigate('/login')
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 401) toast.error('Неверный пароль')
      else toast.error('Не удалось удалить аккаунт')
    } finally {
      setDeleteSubmitting(false)
    }
  }

  function onLogout() {
    logout()
    navigate('/login')
  }

  if (loading) {
    return (
      <div className="mx-auto flex max-w-xl flex-col gap-4 p-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 rounded-2xl" />
      </div>
    )
  }

  if (!profile) {
    return (
      <div className="flex flex-col items-center gap-3 p-16 text-center">
        <p className="font-medium text-foreground">Не удалось загрузить профиль</p>
      </div>
    )
  }

  return (
    <div className="mx-auto flex max-w-xl flex-col gap-5 p-6">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold text-primary">Профиль</h1>
          <p className="flex items-center gap-1.5 text-sm text-muted-foreground">
            <Mail className="size-3.5" /> {profile.email}
          </p>
        </div>
        <Badge variant={profile.role === UserRole.admin ? 'default' : 'secondary'}>{ROLE_LABELS[profile.role!]}</Badge>
      </div>

      <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <GraduationCap className="size-3.5" /> С нами с {formatDate(profile.created_at)}
      </p>

      <form onSubmit={onSave} className="flex flex-col gap-4 rounded-2xl border border-border bg-card p-5">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="nickname">Никнейм</Label>
          <Input
            id="nickname"
            value={nickname}
            onChange={(e) => setNickname(e.target.value)}
            className="h-10 rounded-xl"
            minLength={3}
            maxLength={50}
            required
          />
        </div>

        <div className="flex gap-3">
          <div className="flex flex-1 flex-col gap-1.5">
            <Label>Вуз</Label>
            <Select value={university} onValueChange={(v) => setUniversity(v ?? '')}>
              <SelectTrigger className="h-10 rounded-xl">
                <SelectValue placeholder="Вуз">{(v: string) => UNIVERSITY_LABELS[v] ?? 'Не указан'}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {Object.entries(UNIVERSITY_LABELS).map(([value, label]) => (
                  <SelectItem key={value} value={value}>
                    {label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex w-24 flex-col gap-1.5">
            <Label htmlFor="course">Курс</Label>
            <Input
              id="course"
              type="number"
              min={1}
              max={7}
              value={course}
              onChange={(e) => setCourse(e.target.value)}
              className="h-10 rounded-xl"
            />
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="faculty">Факультет</Label>
          <Input
            id="faculty"
            value={faculty}
            onChange={(e) => setFaculty(e.target.value)}
            placeholder="Например: Лечебное дело"
            className="h-10 rounded-xl"
            maxLength={100}
          />
        </div>

        <Button type="submit" disabled={saving} className="h-11 rounded-full bg-linear-to-r from-primary to-accent text-primary-foreground">
          {saving ? 'Сохраняем…' : 'Сохранить изменения'}
        </Button>
      </form>

      <Button variant="outline" className="h-11 w-fit rounded-full" onClick={onLogout}>
        <LogOut className="size-4" /> Выйти
      </Button>

      <div className="flex flex-col gap-4 rounded-2xl border border-border bg-card p-5">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <Bell className="size-4 text-muted-foreground" />
            <h2 className="font-semibold text-foreground">Push-уведомления</h2>
          </div>
          {isPushSupported() ? (
            <Button variant={pushSubscribed ? 'outline' : 'default'} size="sm" className="h-9 rounded-full" disabled={pushBusy} onClick={onTogglePush}>
              {pushBusy ? 'Подождите…' : pushSubscribed ? 'Отключить' : 'Включить'}
            </Button>
          ) : (
            <span className="text-xs text-muted-foreground">Не поддерживается вашим браузером</span>
          )}
        </div>

        {pushSubscribed && pushPrefs && (
          <div className="flex flex-col gap-2.5">
            {PUSH_PREFERENCE_LABELS.map(({ key, label }) => (
              <label key={key} className="flex items-center gap-2.5 text-sm text-foreground">
                <Checkbox
                  checked={pushPrefs[key] ?? true}
                  onCheckedChange={(v) => {
                    const value = v === true
                    setPushPrefs((prev) => (prev ? { ...prev, [key]: value } : prev))
                    onTogglePreference(key, value)
                  }}
                />
                {label}
              </label>
            ))}
          </div>
        )}
      </div>

      <div className="flex flex-col gap-3 rounded-2xl border border-destructive/30 bg-destructive/5 p-5">
        <div>
          <h2 className="font-semibold text-destructive">Удаление аккаунта</h2>
          <p className="text-sm text-muted-foreground">
            Аккаунт будет удалён безвозвратно, все ваши сессии завершатся. Это действие нельзя отменить.
          </p>
        </div>

        {deleting ? (
          <div className="flex flex-col gap-2">
            <Label htmlFor="delete-password">Введите пароль для подтверждения</Label>
            <Input
              id="delete-password"
              type="password"
              value={deletePassword}
              onChange={(e) => setDeletePassword(e.target.value)}
              className="h-10 rounded-xl"
            />
            <div className="flex gap-2">
              <Button
                variant="destructive"
                size="sm"
                className="h-9 rounded-full"
                disabled={deleteSubmitting}
                onClick={onConfirmDelete}
              >
                {deleteSubmitting ? 'Удаляем…' : 'Удалить безвозвратно'}
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="h-9 rounded-full"
                onClick={() => {
                  setDeleting(false)
                  setDeletePassword('')
                }}
              >
                Отмена
              </Button>
            </div>
          </div>
        ) : (
          <Button variant="destructive" size="sm" className="h-9 w-fit rounded-full" onClick={() => setDeleting(true)}>
            <Trash2 className="size-4" /> Удалить аккаунт
          </Button>
        )}
      </div>
    </div>
  )
}
