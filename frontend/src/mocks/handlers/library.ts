import { http, HttpResponse } from 'msw'
import { TextbookLicenseType, TextbookStorageType } from '@/api/generated'
import type { Textbook, TextbookListItem } from '@/api/generated'

const API = '*/api/v1'

interface SeedTextbook extends Textbook {
  // Бэкенд не отдаёт source_url в публичном Textbook — редирект собирается
  // сервером внутри /source. Здесь это внутреннее поле мока, не часть контракта.
  sourceUrl?: string
}

const textbooks: SeedTextbook[] = [
  {
    id: 't-0001',
    title: 'Анатомия человека. Том 1: Опорно-двигательный аппарат',
    authors: 'М. Р. Сапин, Д. Б. Никитюк',
    isbn: '978-5-9704-5544-1',
    year: 2021,
    pages: 512,
    description:
      'Классический учебник по анатомии для студентов лечебного и педиатрического факультетов. Подробно рассмотрены строение костей, суставов и мышц.',
    subject: 'Анатомия человека',
    course: 1,
    department: 'Кафедра анатомии человека',
    storage_type: TextbookStorageType.A,
    license_type: TextbookLicenseType.cc_by_nc,
    copyright_holder: 'Издательская группа «ГЭОТАР-Медиа»',
    created_at: '2026-01-12T10:00:00Z',
  },
  {
    id: 't-0002',
    title: 'Биохимия: учебник для медицинских вузов',
    authors: 'Е. С. Северин',
    isbn: '978-5-9704-3956-4',
    year: 2020,
    pages: 768,
    description: 'Базовый курс биохимии: обмен веществ, ферменты, гормональная регуляция, биохимия тканей.',
    subject: 'Биохимия',
    course: 2,
    department: 'Кафедра биохимии',
    storage_type: TextbookStorageType.A,
    license_type: TextbookLicenseType.cc_by,
    copyright_holder: 'Е. С. Северин',
    created_at: '2026-01-10T10:00:00Z',
  },
  {
    id: 't-0003',
    title: 'Гистология, эмбриология, цитология',
    authors: 'Ю. И. Афанасьев, Н. А. Юрина',
    isbn: '978-5-9704-6120-6',
    year: 2022,
    pages: 800,
    description: 'Учебник по общей и частной гистологии с атласом микропрепаратов.',
    subject: 'Гистология',
    course: 2,
    department: 'Кафедра гистологии',
    storage_type: TextbookStorageType.A,
    license_type: TextbookLicenseType.public_domain,
    copyright_holder: 'Общественное достояние',
    created_at: '2026-01-08T10:00:00Z',
  },
  {
    id: 't-0004',
    title: 'Нормальная физиология',
    authors: 'В. М. Смирнов',
    isbn: '978-5-9704-4012-6',
    year: 2019,
    pages: 624,
    description: 'Физиология возбудимых тканей, ЦНС, кровообращения, дыхания, пищеварения, эндокринной системы.',
    subject: 'Нормальная физиология',
    course: 2,
    department: 'Кафедра нормальной физиологии',
    storage_type: TextbookStorageType.A,
    license_type: TextbookLicenseType.cc0,
    copyright_holder: 'В. М. Смирнов',
    created_at: '2026-01-05T10:00:00Z',
  },
  {
    id: 't-0005',
    title: 'Фармакология с общей рецептурой',
    authors: 'Д. А. Харкевич',
    isbn: '978-5-9704-5891-6',
    year: 2023,
    pages: 760,
    description: 'Общая и частная фармакология: механизмы действия, показания, побочные эффекты основных групп препаратов.',
    subject: 'Фармакология',
    course: 3,
    department: 'Кафедра фармакологии',
    storage_type: TextbookStorageType.A,
    license_type: TextbookLicenseType.custom,
    copyright_holder: 'Издательство «ГЭОТАР-Медиа», по договору с правообладателем',
    created_at: '2026-01-03T10:00:00Z',
  },
  {
    id: 't-0006',
    title: 'Патологическая анатомия',
    authors: 'М. А. Пальцев, В. С. Пауков',
    isbn: '978-5-9704-4501-5',
    year: 2021,
    pages: 960,
    description: 'Общая и частная патологическая анатомия с клинико-морфологическими сопоставлениями.',
    subject: 'Патологическая анатомия',
    course: 3,
    department: 'Кафедра патологической анатомии',
    storage_type: TextbookStorageType.A,
    license_type: TextbookLicenseType.cc_by_sa,
    copyright_holder: 'М. А. Пальцев',
    created_at: '2025-12-28T10:00:00Z',
  },
  {
    id: 't-0007',
    title: 'Хирургические болезни',
    storage_type: TextbookStorageType.B,
    license_type: TextbookLicenseType.all_rights_reserved,
    created_at: '2025-12-20T10:00:00Z',
    sourceUrl: 'https://www.studentlibrary.ru/book/surgical-diseases',
  },
  {
    id: 't-0008',
    title: 'Внутренние болезни. Учебник в 2 томах',
    storage_type: TextbookStorageType.B,
    license_type: TextbookLicenseType.all_rights_reserved,
    created_at: '2025-12-18T10:00:00Z',
    sourceUrl: 'https://www.rosmedlib.ru/book/internal-diseases',
  },
  {
    id: 't-0009',
    title: 'Микробиология, вирусология и иммунология',
    authors: 'В. В. Зверев, М. Н. Бойченко',
    isbn: '978-5-9704-3702-7',
    year: 2020,
    pages: 480,
    description: 'Общая и частная микробиология, основы вирусологии и клинической иммунологии.',
    subject: 'Микробиология',
    course: 2,
    department: 'Кафедра микробиологии',
    storage_type: TextbookStorageType.A,
    license_type: TextbookLicenseType.cc_by_nc,
    copyright_holder: 'В. В. Зверев',
    created_at: '2025-12-15T10:00:00Z',
  },
  {
    id: 't-0010',
    title: 'Акушерство и гинекология: национальное руководство',
    storage_type: TextbookStorageType.B,
    license_type: TextbookLicenseType.all_rights_reserved,
    created_at: '2025-12-10T10:00:00Z',
    sourceUrl: 'https://www.rosmedlib.ru/book/obstetrics-gynecology',
  },
  {
    id: 't-0011',
    title: 'Топографическая анатомия и оперативная хирургия',
    authors: 'И. И. Каган',
    isbn: '978-5-9704-4210-6',
    year: 2018,
    pages: 512,
    description: 'Топография областей тела человека и основные оперативные доступы.',
    subject: 'Оперативная хирургия',
    course: 3,
    department: 'Кафедра оперативной хирургии',
    storage_type: TextbookStorageType.A,
    license_type: TextbookLicenseType.public_domain,
    copyright_holder: 'Общественное достояние',
    created_at: '2025-12-05T10:00:00Z',
  },
  {
    id: 't-0012',
    title: 'Педиатрия. Национальное руководство',
    storage_type: TextbookStorageType.B,
    license_type: TextbookLicenseType.all_rights_reserved,
    created_at: '2025-12-01T10:00:00Z',
    sourceUrl: 'https://www.rosmedlib.ru/book/pediatrics',
  },
]

function toListItem(t: SeedTextbook): TextbookListItem {
  if (t.storage_type === TextbookStorageType.B) {
    return {
      id: t.id,
      title: t.title,
      storage_type: t.storage_type,
      license_type: t.license_type,
      created_at: t.created_at,
    }
  }
  const { sourceUrl: _sourceUrl, ...rest } = t
  void _sourceUrl
  return rest
}

function toDetails(t: SeedTextbook): Textbook {
  if (t.storage_type === TextbookStorageType.B) {
    return {
      id: t.id,
      title: t.title,
      storage_type: t.storage_type,
      license_type: t.license_type,
      created_at: t.created_at,
    }
  }
  const { sourceUrl: _sourceUrl, ...rest } = t
  void _sourceUrl
  return rest
}

// Минимальный валидный PDF — достаточно, чтобы браузер принял его как файл
// и триггернул сохранение; содержимое не имеет значения для проверки потока.
const MOCK_PDF_BYTES = new TextEncoder().encode(
  '%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]>>endobj\ntrailer<</Root 1 0 R>>',
)

export const libraryHandlers = [
  http.get(`${API}/library/textbooks`, ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.toLowerCase()
    const subject = url.searchParams.get('subject')
    const course = url.searchParams.get('course')
    const storageType = url.searchParams.get('storage_type')
    const sort = url.searchParams.get('sort') ?? 'created_at_desc'
    const page = Number(url.searchParams.get('page') ?? '1')
    const limit = Number(url.searchParams.get('limit') ?? '20')

    let filtered = textbooks.filter((t) => {
      if (q && !t.title?.toLowerCase().includes(q)) return false
      if (subject && t.subject !== subject) return false
      if (course && String(t.course) !== course) return false
      if (storageType && t.storage_type !== storageType) return false
      return true
    })

    filtered = [...filtered].sort((a, b) => {
      if (sort === 'title_asc') return (a.title ?? '').localeCompare(b.title ?? '', 'ru')
      if (sort === 'title_desc') return (b.title ?? '').localeCompare(a.title ?? '', 'ru')
      return (b.created_at ?? '').localeCompare(a.created_at ?? '')
    })

    const total = filtered.length
    const start = (page - 1) * limit
    const pageItems = filtered.slice(start, start + limit)

    return HttpResponse.json({
      data: pageItems.map(toListItem),
      pagination: { page, limit, total, has_next: start + limit < total },
    })
  }),

  http.get(`${API}/library/textbooks/:id`, ({ params }) => {
    const textbook = textbooks.find((t) => t.id === params.id)
    if (!textbook) {
      return HttpResponse.json({ error: { code: 'not_found', message: 'Учебник не найден' } }, { status: 404 })
    }
    return HttpResponse.json(toDetails(textbook))
  }),

  http.get(`${API}/library/textbooks/:id/download`, ({ params, request }) => {
    const textbook = textbooks.find((t) => t.id === params.id)
    if (!textbook) {
      return HttpResponse.json({ error: { code: 'not_found', message: 'Учебник не найден' } }, { status: 404 })
    }
    if (textbook.storage_type !== TextbookStorageType.A) {
      return HttpResponse.json(
        { error: { code: 'forbidden', message: 'Скачивание доступно только для категории A' } },
        { status: 403 },
      )
    }
    if (!request.headers.get('authorization')) {
      return HttpResponse.json({ error: { code: 'unauthorized', message: 'Требуется вход' } }, { status: 401 })
    }
    return new HttpResponse(MOCK_PDF_BYTES, {
      status: 200,
      headers: {
        'Content-Type': 'application/pdf',
        // Заголовки HTTP ограничены ISO-8859-1 — кириллицу сюда класть нельзя.
        // Реальное имя файла для сохранения фронтенд берёт из title и ставит
        // сам через атрибут download (см. use-download-textbook.ts).
        'Content-Disposition': 'attachment; filename="textbook.pdf"',
      },
    })
  }),

  http.get(`${API}/library/textbooks/:id/source`, ({ params }) => {
    const textbook = textbooks.find((t) => t.id === params.id)
    if (!textbook || textbook.storage_type !== TextbookStorageType.B || !textbook.sourceUrl) {
      return HttpResponse.json({ error: { code: 'not_found', message: 'Источник не найден' } }, { status: 404 })
    }
    return HttpResponse.redirect(textbook.sourceUrl, 302)
  }),
]
