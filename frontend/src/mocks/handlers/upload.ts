import { http, HttpResponse } from 'msw'
import type { UploadResponse } from '@/api/generated'

const API = '*/api/v1'

export const uploadHandlers = [
  http.post(`${API}/upload`, async ({ request }) => {
    const url = new URL(request.url)
    const type = url.searchParams.get('type')

    if (!request.headers.get('authorization')) {
      return HttpResponse.json({ error: { code: 'unauthorized', message: 'Требуется вход' } }, { status: 401 })
    }

    const formData = await request.formData()
    const file = formData.get('file')
    if (!(file instanceof File)) {
      return HttpResponse.json({ error: { code: 'bad_request', message: 'Файл не передан' } }, { status: 400 })
    }
    if (type === 'pdf' && file.type !== 'application/pdf' && !file.name.toLowerCase().endsWith('.pdf')) {
      return HttpResponse.json({ error: { code: 'bad_request', message: 'Ожидается PDF-файл' } }, { status: 400 })
    }

    const response: UploadResponse = {
      file_id: crypto.randomUUID(),
      url: `https://mock-s3.medflow.local/temp/${encodeURIComponent(file.name)}`,
      size_bytes: file.size,
      mime_type: file.type || 'application/pdf',
      expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
    }
    return HttpResponse.json(response, { status: 201 })
  }),
]
