import { http, HttpResponse } from 'msw'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { toast } from 'sonner'
import { axiosInstance } from './axios-instance'
import { server } from '@/test/setup'

vi.mock('sonner', () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

const API = '*/api/v1'

describe('axios-instance 429 handling', () => {
  beforeEach(() => {
    vi.mocked(toast.error).mockClear()
  })

  it('shows a toast with the server message and Retry-After hint on 429', async () => {
    server.use(
      http.get(`${API}/rate-limited-probe`, () =>
        HttpResponse.json(
          { error: { code: 'RATE_LIMITED', message: 'слишком много запросов, попробуйте позже' } },
          { status: 429, headers: { 'Retry-After': '7' } },
        ),
      ),
    )

    await expect(axiosInstance.get('/rate-limited-probe')).rejects.toBeTruthy()

    expect(toast.error).toHaveBeenCalledWith('слишком много запросов, попробуйте позже (повторите через 7 сек)')
  })

  it('falls back to a generic Russian message when the server sends none', async () => {
    server.use(http.get(`${API}/rate-limited-probe-2`, () => new HttpResponse(null, { status: 429 })))

    await expect(axiosInstance.get('/rate-limited-probe-2')).rejects.toBeTruthy()

    expect(toast.error).toHaveBeenCalledWith('Слишком много запросов, попробуйте позже')
  })
})
