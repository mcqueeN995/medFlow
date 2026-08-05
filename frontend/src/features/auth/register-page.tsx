import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { AuthLayout } from './auth-layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { postAuthRegister } from '@/api/generated/medFlowAPI'
import { University } from '@/api/generated'
import { useAuthStore } from '@/stores/auth-store'

const universityLabels: Record<string, string> = {
  [University.sechenov]: 'Сеченовский университет',
  [University.pirogov]: 'РНИМУ им. Пирогова',
  [University.evdokimov]: 'МГМСУ им. Евдокимова',
  [University.other]: 'Другой вуз',
}

export function RegisterPage() {
  const navigate = useNavigate()
  const setSession = useAuthStore((s) => s.setSession)

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [nickname, setNickname] = useState('')
  const [university, setUniversity] = useState<string>('')
  const [course, setCourse] = useState('')
  const [agreed, setAgreed] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!agreed) {
      setError('Нужно принять условия использования')
      return
    }
    setError(null)
    setSubmitting(true)
    try {
      const res = await postAuthRegister({
        email,
        password,
        nickname,
        university: (university as University) || undefined,
        course: course ? Number(course) : undefined,
        agree_to_terms: agreed,
      })
      if (!res.user || !res.access_token || !res.refresh_token) {
        throw new Error('incomplete auth response')
      }
      setSession({ user: res.user, accessToken: res.access_token, refreshToken: res.refresh_token })
      navigate('/library')
    } catch {
      setError('Не удалось зарегистрироваться — проверьте данные')
      toast.error('Ошибка регистрации')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthLayout>
      <form onSubmit={onSubmit} className="flex flex-col gap-3">
        <Input
          type="email"
          placeholder="Почта"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          className="h-12 rounded-full px-5"
        />
        <Input
          type="text"
          placeholder="Никнейм"
          value={nickname}
          onChange={(e) => setNickname(e.target.value)}
          required
          minLength={3}
          className="h-12 rounded-full px-5"
        />
        <Input
          type="password"
          placeholder="Пароль (мин. 8 символов)"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          minLength={8}
          className="h-12 rounded-full px-5"
        />

        <div className="flex gap-3">
          <Select value={university} onValueChange={(v) => setUniversity(v ?? '')}>
            <SelectTrigger className="h-12 flex-1 rounded-full px-5">
              <SelectValue placeholder="Вуз">{(v: string) => universityLabels[v] ?? 'Вуз'}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {Object.entries(universityLabels).map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Input
            type="number"
            placeholder="Курс"
            min={1}
            max={7}
            value={course}
            onChange={(e) => setCourse(e.target.value)}
            className="h-12 w-24 rounded-full px-5"
          />
        </div>

        <label className="flex items-start gap-2 text-sm text-muted-foreground">
          <Checkbox
            checked={agreed}
            onCheckedChange={(v) => setAgreed(v === true)}
            className="mt-0.5 size-5 rounded-md"
          />
          <span>
            Я принимаю{' '}
            <Link to="/terms" className="text-accent underline underline-offset-2">
              Условия использования
            </Link>{' '}
            и{' '}
            <Link to="/privacy" className="text-accent underline underline-offset-2">
              Политику конфиденциальности
            </Link>
          </span>
        </label>

        {error && <p className="text-sm text-destructive">{error}</p>}

        <Button
          type="submit"
          disabled={submitting}
          className="h-12 rounded-full bg-linear-to-r from-primary to-accent text-base font-medium text-primary-foreground shadow-md hover:opacity-95"
        >
          {submitting ? 'Регистрируем…' : 'Зарегистрироваться'}
        </Button>

        <p className="text-center text-sm text-muted-foreground">
          Уже есть аккаунт?{' '}
          <Link to="/login" className="font-medium text-accent underline underline-offset-2">
            войти
          </Link>
        </p>
      </form>
    </AuthLayout>
  )
}
