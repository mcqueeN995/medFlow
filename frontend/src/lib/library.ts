import { TextbookLicenseType } from '@/api/generated'

export const LICENSE_LABELS: Record<TextbookLicenseType, string> = {
  [TextbookLicenseType.cc_by]: 'CC BY',
  [TextbookLicenseType.cc_by_nc]: 'CC BY-NC',
  [TextbookLicenseType.cc_by_sa]: 'CC BY-SA',
  [TextbookLicenseType.cc0]: 'CC0 / общественное достояние',
  [TextbookLicenseType.public_domain]: 'Общественное достояние',
  [TextbookLicenseType.all_rights_reserved]: 'Все права защищены',
  [TextbookLicenseType.custom]: 'По договору с правообладателем',
}

// Каталог сейчас фильтруется по свободному текстовому полю subject — отдельного
// эндпоинта фасетов в API нет, поэтому список курируется вручную под учебный план.
export const SUBJECTS = [
  'Анатомия человека',
  'Биохимия',
  'Гистология',
  'Микробиология',
  'Нормальная физиология',
  'Оперативная хирургия',
  'Патологическая анатомия',
  'Фармакология',
]

export const COURSES = [1, 2, 3, 4, 5, 6]

export function formatDate(iso?: string): string {
  if (!iso) return ''
  return new Intl.DateTimeFormat('ru-RU', { day: '2-digit', month: 'long', year: 'numeric' }).format(new Date(iso))
}
