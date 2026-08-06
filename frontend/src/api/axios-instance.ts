import axios, { type AxiosRequestConfig } from 'axios'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'

export const axiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_URL ?? '/api/v1',
})

axiosInstance.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

let refreshing: Promise<string | null> | null = null

axiosInstance.interceptors.response.use(
  (response) => response,
  async (error) => {
    const original = error.config
    if (error.response?.status === 401 && !original._retry) {
      original._retry = true
      refreshing ??= useAuthStore.getState().refresh()
      const newToken = await refreshing
      refreshing = null
      if (newToken) {
        original.headers.Authorization = `Bearer ${newToken}`
        return axiosInstance(original)
      }
      useAuthStore.getState().logout()
    } else if (error.response?.status === 429) {
      const retryAfter = error.response.headers?.['retry-after']
      const suffix = retryAfter ? ` (повторите через ${retryAfter} сек)` : ''
      toast.error((error.response.data?.error?.message ?? 'Слишком много запросов, попробуйте позже') + suffix)
    }
    return Promise.reject(error)
  },
)

// orval mutator: оборачивает сгенерированные вызовы в axios с отменой запроса
export const customInstance = <T>(config: AxiosRequestConfig): Promise<T> => {
  const controller = new AbortController()
  const promise = axiosInstance({ ...config, signal: controller.signal }).then((response) => response.data) as Promise<T> & {
    cancel?: () => void
  }
  promise.cancel = () => controller.abort()
  return promise
}

export default customInstance
