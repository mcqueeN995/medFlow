import { useState } from 'react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'

const API_URL = import.meta.env.VITE_API_URL ?? '/api/v1'

// Скачивание для категории A требует Bearer-токена (см. openapi.yaml — эндпоинт
// наследует глобальный security: BearerAuth), поэтому обычная <a href> не подойдёт:
// браузер не приложит Authorization при простой навигации. Вместо этого делаем
// authenticated fetch, браузер сам проходит 302 → presigned S3 URL, а мы получаем
// готовые байты PDF и триггерим сохранение через временную ссылку.
export function useDownloadTextbook() {
  const [downloadingId, setDownloadingId] = useState<string | null>(null)

  async function download(id: string, title: string) {
    const accessToken = useAuthStore.getState().accessToken
    setDownloadingId(id)
    try {
      const res = await fetch(`${API_URL}/library/textbooks/${id}/download`, {
        headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : {},
      })

      if (!res.ok) {
        if (res.status === 401) toast.error('Войдите, чтобы скачать учебник')
        else if (res.status === 403) toast.error('Скачивание недоступно для этой категории')
        else if (res.status === 404) toast.error('Файл не найден')
        else toast.error('Не удалось скачать файл')
        return
      }

      const blob = await res.blob()
      const objectUrl = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = objectUrl
      link.download = `${title}.pdf`
      document.body.appendChild(link)
      link.click()
      link.remove()
      URL.revokeObjectURL(objectUrl)
    } catch {
      toast.error('Не удалось скачать файл — проверьте соединение')
    } finally {
      setDownloadingId(null)
    }
  }

  return { download, downloadingId }
}
