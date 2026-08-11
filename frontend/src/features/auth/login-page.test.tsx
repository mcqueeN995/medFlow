import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it } from 'vitest'
import { LoginPage } from './login-page'
import { useAuthStore } from '@/stores/auth-store'

function renderLoginPage() {
  return render(
    <MemoryRouter initialEntries={['/login']}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/library" element={<div>Библиотека (мок-цель редиректа)</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('LoginPage', () => {
  afterEach(() => {
    useAuthStore.getState().logout()
  })

  it('requires accepting terms before submitting', async () => {
    const user = userEvent.setup()
    renderLoginPage()

    await user.type(screen.getByPlaceholderText('Логин или почта'), 'student@sechenov.ru')
    await user.type(screen.getByPlaceholderText('Пароль'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Войти' }))

    expect(await screen.findByText('Нужно принять условия использования')).toBeInTheDocument()
    expect(screen.queryByText(/Библиотека/)).not.toBeInTheDocument()
  })

  it('logs in with valid credentials and redirects to the library, populating the auth store', async () => {
    const user = userEvent.setup()
    renderLoginPage()

    await user.type(screen.getByPlaceholderText('Логин или почта'), 'student@sechenov.ru')
    await user.type(screen.getByPlaceholderText('Пароль'), 'password123')
    await user.click(screen.getByRole('checkbox'))
    await user.click(screen.getByRole('button', { name: 'Войти' }))

    await waitFor(() => expect(screen.getByText(/Библиотека/)).toBeInTheDocument())
    expect(useAuthStore.getState().accessToken).toBeTruthy()
    expect(useAuthStore.getState().user?.email).toBe('student@sechenov.ru')
  })

  it('logs in with a login instead of email', async () => {
    const user = userEvent.setup()
    renderLoginPage()

    await user.type(screen.getByPlaceholderText('Логин или почта'), 'anatomy_enjoyer')
    await user.type(screen.getByPlaceholderText('Пароль'), 'password123')
    await user.click(screen.getByRole('checkbox'))
    await user.click(screen.getByRole('button', { name: 'Войти' }))

    await waitFor(() => expect(screen.getByText(/Библиотека/)).toBeInTheDocument())
    expect(useAuthStore.getState().user?.login).toBe('anatomy_enjoyer')
  })

  it('shows an error toast-friendly message on wrong credentials', async () => {
    const user = userEvent.setup()
    renderLoginPage()

    await user.type(screen.getByPlaceholderText('Логин или почта'), 'student@sechenov.ru')
    await user.type(screen.getByPlaceholderText('Пароль'), 'wrong-password')
    await user.click(screen.getByRole('checkbox'))
    await user.click(screen.getByRole('button', { name: 'Войти' }))

    expect(await screen.findByText('Неверный логин или пароль')).toBeInTheDocument()
    expect(useAuthStore.getState().accessToken).toBeNull()
  })
})
