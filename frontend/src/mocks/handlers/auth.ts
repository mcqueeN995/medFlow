import { http, HttpResponse } from 'msw'
import { University, UserRole } from '@/api/generated'
import type { AuthResponse, LoginRequest, RegisterRequest, UpdateProfileRequest, UserProfile } from '@/api/generated'

const API = '*/api/v1'

interface MockUser extends UserProfile {
  password: string
}

// Посевные пользователи для входа "с нуля" + всё зарегистрированное в рамках сессии вкладки.
const users: MockUser[] = [
  {
    id: '11111111-1111-1111-1111-111111111111',
    email: 'student@sechenov.ru',
    nickname: 'anatomy_enjoyer',
    role: UserRole.user,
    university: University.sechenov,
    course: 3,
    faculty: 'Лечебное дело',
    email_verified_at: new Date().toISOString(),
    created_at: new Date('2026-01-15').toISOString(),
    password: 'password123',
  },
  {
    id: '22222222-2222-2222-2222-222222222222',
    email: 'admin@medflow.local',
    nickname: 'admin',
    role: UserRole.admin,
    email_verified_at: new Date().toISOString(),
    created_at: new Date('2026-01-01').toISOString(),
    password: 'admin12345',
  },
]

// Токен → id владельца. Реальный backend кладёт user_id в JWT-claims (см.
// internal/pkg/jwt); здесь достаточно плоской карты, чтобы /users/me и
// /auth/refresh отвечали за того, кто реально залогинен, а не всегда за
// users[0] — иначе с двумя ролями в моках сессии будут путаться.
const tokenOwners = new Map<string, string>()

function issueTokens(user: MockUser): AuthResponse {
  const { password: _password, ...profile } = user
  void _password
  const accessToken = `mock-access-${user.id}-${Date.now()}`
  const refreshToken = `mock-refresh-${user.id}-${Date.now()}`
  tokenOwners.set(accessToken, user.id!)
  tokenOwners.set(refreshToken, user.id!)
  return { user: profile, access_token: accessToken, refresh_token: refreshToken, expires_in: 900 }
}

function userByToken(token: string | null): MockUser | undefined {
  if (!token) return undefined
  const id = tokenOwners.get(token.replace(/^Bearer\s+/i, ''))
  return users.find((u) => u.id === id)
}

// Экспортируется для других модулей моков (forum.ts и т.п.) - форум полностью
// закрыт для гостя (см. openapi.yaml: security нигде не переопределён для
// /threads и /comments), поэтому его хендлерам тоже нужно резолвить автора
// по Bearer-токену запроса.
export { users, userByToken }
export type { MockUser }

export const authHandlers = [
  http.post(`${API}/auth/login`, async ({ request }) => {
    const body = (await request.json()) as LoginRequest
    const user = users.find((u) => u.email === body.email && u.password === body.password)
    if (!user) {
      return HttpResponse.json(
        { error: { code: 'invalid_credentials', message: 'Неверный email или пароль' } },
        { status: 401 },
      )
    }
    return HttpResponse.json(issueTokens(user))
  }),

  http.post(`${API}/auth/register`, async ({ request }) => {
    const body = (await request.json()) as RegisterRequest
    if (users.some((u) => u.email === body.email)) {
      return HttpResponse.json(
        { error: { code: 'email_taken', message: 'Пользователь с таким email уже существует' } },
        { status: 409 },
      )
    }
    const newUser: MockUser = {
      id: crypto.randomUUID(),
      email: body.email,
      nickname: body.nickname,
      role: UserRole.user,
      university: body.university,
      course: body.course,
      faculty: body.faculty,
      created_at: new Date().toISOString(),
      password: body.password,
    }
    users.push(newUser)
    return HttpResponse.json(issueTokens(newUser), { status: 201 })
  }),

  http.post(`${API}/auth/refresh`, async ({ request }) => {
    const { refresh_token } = (await request.json()) as { refresh_token: string }
    const user = userByToken(refresh_token) ?? users[0]
    const { access_token, refresh_token: newRefresh, expires_in } = issueTokens(user)
    return HttpResponse.json({ access_token, refresh_token: newRefresh, expires_in })
  }),

  http.get(`${API}/users/me`, ({ request }) => {
    const user = userByToken(request.headers.get('authorization'))
    if (!user) return HttpResponse.json({ error: { code: 'unauthorized', message: 'Не авторизован' } }, { status: 401 })
    const { password: _password, ...profile } = user
    void _password
    return HttpResponse.json(profile)
  }),

  http.patch(`${API}/users/me`, async ({ request }) => {
    const user = userByToken(request.headers.get('authorization'))
    if (!user) return HttpResponse.json({ error: { code: 'unauthorized', message: 'Не авторизован' } }, { status: 401 })

    const body = (await request.json()) as UpdateProfileRequest
    if (body.nickname && users.some((u) => u.id !== user.id && u.nickname === body.nickname)) {
      return HttpResponse.json(
        { error: { code: 'NICKNAME_EXISTS', message: 'Никнейм уже занят' } },
        { status: 409 },
      )
    }

    if (body.nickname !== undefined) user.nickname = body.nickname
    if (body.university !== undefined) user.university = body.university
    if (body.course !== undefined) user.course = body.course
    if (body.faculty !== undefined) user.faculty = body.faculty

    const { password: _password, ...profile } = user
    void _password
    return HttpResponse.json(profile)
  }),

  http.delete(`${API}/users/me`, ({ request }) => {
    const user = userByToken(request.headers.get('authorization'))
    if (!user) return HttpResponse.json({ error: { code: 'unauthorized', message: 'Не авторизован' } }, { status: 401 })

    const url = new URL(request.url)
    const password = url.searchParams.get('password')
    if (password !== user.password) {
      return HttpResponse.json(
        { error: { code: 'INVALID_CREDENTIALS', message: 'Неверный пароль' } },
        { status: 401 },
      )
    }

    const index = users.findIndex((u) => u.id === user.id)
    if (index !== -1) users.splice(index, 1)
    for (const [token, ownerId] of tokenOwners) {
      if (ownerId === user.id) tokenOwners.delete(token)
    }
    return new HttpResponse(null, { status: 204 })
  }),
]
