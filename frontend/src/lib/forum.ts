import { ThreadTag } from '@/api/generated'

export const THREAD_TAG_LABELS: Record<ThreadTag, string> = {
  [ThreadTag.study]: 'Учёба',
  [ThreadTag.department]: 'Кафедра',
  [ThreadTag.humor]: 'Юмор',
  [ThreadTag.marketplace]: 'Барахолка',
  [ThreadTag.clinical_base]: 'Клиническая база',
  [ThreadTag.news]: 'Новости',
  [ThreadTag.help]: 'Помощь',
  [ThreadTag.other]: 'Другое',
}

const RTF = new Intl.RelativeTimeFormat('ru-RU', { numeric: 'auto' })

export function timeAgo(iso?: string): string {
  if (!iso) return ''
  const diffMs = new Date(iso).getTime() - Date.now()
  const diffMin = Math.round(diffMs / 60000)
  if (Math.abs(diffMin) < 60) return RTF.format(diffMin, 'minute')
  const diffHour = Math.round(diffMin / 60)
  if (Math.abs(diffHour) < 24) return RTF.format(diffHour, 'hour')
  const diffDay = Math.round(diffHour / 24)
  if (Math.abs(diffDay) < 30) return RTF.format(diffDay, 'day')
  const diffMonth = Math.round(diffDay / 30)
  return RTF.format(diffMonth, 'month')
}
