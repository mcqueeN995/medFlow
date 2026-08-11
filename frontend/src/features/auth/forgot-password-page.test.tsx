import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { ForgotPasswordPage } from './forgot-password-page'
import { LoginPage } from './login-page'

function renderForgotPasswordPage() {
  return render(
    <MemoryRouter initialEntries={['/forgot-password']}>
      <Routes>
        <Route path="/forgot-password" element={<ForgotPasswordPage />} />
        <Route path="/login" element={<LoginPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('ForgotPasswordPage', () => {
  it('requests a code, then confirms it with a new password', async () => {
    const user = userEvent.setup()
    renderForgotPasswordPage()

    await user.type(screen.getByPlaceholderText('Логин или почта'), 'student@sechenov.ru')
    await user.click(screen.getByRole('button', { name: 'Отправить код' }))

    await screen.findByPlaceholderText('Код из письма')

    await user.type(screen.getByPlaceholderText('Код из письма'), '654321')
    await user.type(screen.getByPlaceholderText('Новый пароль (мин. 8 символов)'), 'newpassword123')
    await user.click(screen.getByRole('button', { name: 'Сменить пароль' }))

    await waitFor(() => expect(screen.getByText(/Пароль успешно обновлён/)).toBeInTheDocument())
  })

  it('shows an error for an invalid code', async () => {
    const user = userEvent.setup()
    renderForgotPasswordPage()

    await user.type(screen.getByPlaceholderText('Логин или почта'), 'student@sechenov.ru')
    await user.click(screen.getByRole('button', { name: 'Отправить код' }))

    await screen.findByPlaceholderText('Код из письма')

    await user.type(screen.getByPlaceholderText('Код из письма'), '000000')
    await user.type(screen.getByPlaceholderText('Новый пароль (мин. 8 символов)'), 'newpassword123')
    await user.click(screen.getByRole('button', { name: 'Сменить пароль' }))

    expect(await screen.findByText('Неверный или истёкший код')).toBeInTheDocument()
  })

  it('is reachable from the login page', async () => {
    const user = userEvent.setup()
    render(
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
        </Routes>
      </MemoryRouter>,
    )

    await user.click(screen.getByRole('link', { name: 'Забыли пароль?' }))
    await screen.findByText(/если аккаунт существует/i)
  })
})
