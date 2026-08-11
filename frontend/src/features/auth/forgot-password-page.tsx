import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { AuthLayout } from './auth-layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { postAuthPasswordReset, postAuthPasswordResetConfirm } from '@/api/generated/medFlowAPI'

export function ForgotPasswordPage() {
  const [login, setLogin] = useState('')
  const [codeSent, setCodeSent] = useState(false)
  const [code, setCode] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState(false)

  async function onRequestCode(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await postAuthPasswordReset({ login })
      setCodeSent(true)
    } catch {
      setError('Не удалось отправить код — попробуйте ещё раз')
      toast.error('Ошибка запроса кода')
    } finally {
      setSubmitting(false)
    }
  }

  async function onConfirm(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await postAuthPasswordResetConfirm({ code, new_password: newPassword })
      setDone(true)
      toast.success('Пароль обновлён')
    } catch {
      setError('Неверный или истёкший код')
      toast.error('Не удалось сменить пароль')
    } finally {
      setSubmitting(false)
    }
  }

  if (done) {
    return (
      <AuthLayout>
        <div className="flex flex-col items-center gap-4 text-center">
          <p className="text-sm text-muted-foreground">
            Пароль успешно обновлён. Теперь можно войти с новым паролем.
          </p>
          <Link to="/login" className="font-medium text-accent underline underline-offset-2">
            Перейти ко входу
          </Link>
        </div>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout>
      {!codeSent ? (
        <form onSubmit={onRequestCode} className="flex flex-col gap-4">
          <p className="text-sm text-muted-foreground">
            Укажите логин или почту — если аккаунт существует, на его email придёт код для восстановления пароля.
          </p>
          <Input
            type="text"
            placeholder="Логин или почта"
            value={login}
            onChange={(e) => setLogin(e.target.value)}
            required
            autoComplete="username"
            className="h-12 rounded-full px-5"
          />

          {error && <p className="text-sm text-destructive">{error}</p>}

          <Button
            type="submit"
            disabled={submitting}
            className="h-12 rounded-full bg-linear-to-r from-primary to-accent text-base font-medium text-primary-foreground shadow-md hover:opacity-95"
          >
            {submitting ? 'Отправляем…' : 'Отправить код'}
          </Button>

          <p className="text-center text-sm text-muted-foreground">
            Вспомнили пароль?{' '}
            <Link to="/login" className="font-medium text-accent underline underline-offset-2">
              войти
            </Link>
          </p>
        </form>
      ) : (
        <form onSubmit={onConfirm} className="flex flex-col gap-4">
          <p className="text-sm text-muted-foreground">
            Если аккаунт с таким логином или почтой существует, мы отправили на его email код подтверждения. Введите код и новый пароль.
          </p>
          <Input
            type="text"
            placeholder="Код из письма"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            required
            inputMode="numeric"
            className="h-12 rounded-full px-5"
          />
          <Input
            type="password"
            placeholder="Новый пароль (мин. 8 символов)"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            required
            minLength={8}
            autoComplete="new-password"
            className="h-12 rounded-full px-5"
          />

          {error && <p className="text-sm text-destructive">{error}</p>}

          <Button
            type="submit"
            disabled={submitting}
            className="h-12 rounded-full bg-linear-to-r from-primary to-accent text-base font-medium text-primary-foreground shadow-md hover:opacity-95"
          >
            {submitting ? 'Сохраняем…' : 'Сменить пароль'}
          </Button>

          <button
            type="button"
            onClick={() => setCodeSent(false)}
            className="text-center text-sm text-muted-foreground underline underline-offset-2"
          >
            Отправить код повторно
          </button>
        </form>
      )}
    </AuthLayout>
  )
}
