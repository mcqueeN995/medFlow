import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import axios from 'axios'
import type { UserProfile } from '@/api/generated'

const baseURL = import.meta.env.VITE_API_URL ?? '/api/v1'

interface AuthState {
  user: UserProfile | null
  accessToken: string | null
  refreshToken: string | null
  setSession: (session: { user: UserProfile; accessToken: string; refreshToken: string }) => void
  setUser: (user: UserProfile) => void
  refresh: () => Promise<string | null>
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      accessToken: null,
      refreshToken: null,

      setSession: ({ user, accessToken, refreshToken }) =>
        set({ user, accessToken, refreshToken }),

      setUser: (user) => set({ user }),

      // Отдельный от orval-клиента axios-вызов — исключает циклическую
      // зависимость с axios-instance.ts, который читает этот стор.
      refresh: async () => {
        const refreshToken = get().refreshToken
        if (!refreshToken) return null
        try {
          const { data } = await axios.post(`${baseURL}/auth/refresh`, { refresh_token: refreshToken })
          set({ accessToken: data.access_token, refreshToken: data.refresh_token })
          return data.access_token as string
        } catch {
          set({ user: null, accessToken: null, refreshToken: null })
          return null
        }
      },

      logout: () => set({ user: null, accessToken: null, refreshToken: null }),
    }),
    {
      name: 'medflow-auth',
      partialize: (state) => ({
        user: state.user,
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
      }),
    },
  ),
)
