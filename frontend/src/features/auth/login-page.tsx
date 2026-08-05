import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Eye, EyeOff } from 'lucide-react'
import { toast } from 'sonner'
import { AuthLayout } from './auth-layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { postAuthLogin } from '@/api/generated/medFlowAPI'
import { useAuthStore } from '@/stores/auth-store'

export function LoginPage() {
  const navigate = useNavigate()
  const setSession = useAuthStore((s) => s.setSession)

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
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
      const res = await postAuthLogin({ email, password })
      if (!res.user || !res.access_token || !res.refresh_token) {
        throw new Error('incomplete auth response')
      }
      setSession({ user: res.user, accessToken: res.access_token, refreshToken: res.refresh_token })
      navigate('/library')
    } catch {
      setError('Неверный логин или пароль')
      toast.error('Не удалось войти')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthLayout>
      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        <Input
          type="email"
          placeholder="Логин/почта"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          autoComplete="email"
          className="h-12 rounded-full px-5"
        />

        <div className="relative">
          <Input
            type={showPassword ? 'text' : 'password'}
            placeholder="Пароль"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="current-password"
            className="h-12 rounded-full px-5 pr-12"
          />
          <button
            type="button"
            onClick={() => setShowPassword((v) => !v)}
            className="absolute inset-y-0 right-4 flex items-center text-muted-foreground"
            aria-label={showPassword ? 'Скрыть пароль' : 'Показать пароль'}
          >
            {showPassword ? <EyeOff className="size-5" /> : <Eye className="size-5" />}
          </button>
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
          {submitting ? 'Входим…' : 'Войти'}
        </Button>

        <p className="text-center text-sm text-muted-foreground">
          У вас нет аккаунта?{' '}
          <Link to="/register" className="font-medium text-accent underline underline-offset-2">
            зарегистрировать
          </Link>
        </p>
      </form>
    </AuthLayout>
  )
}
