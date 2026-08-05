import { BookOpen, Coffee, Laptop, MapPin, Trees, UtensilsCrossed, type LucideIcon } from 'lucide-react'
import { PoiType } from '@/api/generated'

interface PoiTypeMeta {
  label: string
  color: string
  icon: LucideIcon
}

export const POI_TYPE_META: Record<PoiType, PoiTypeMeta> = {
  [PoiType.coworking]: { label: 'Коворкинг', color: '#3E8C9E', icon: Laptop },
  [PoiType.cafe]: { label: 'Кафе', color: '#C98A2C', icon: Coffee },
  [PoiType.library]: { label: 'Библиотека', color: '#1F3A5F', icon: BookOpen },
  [PoiType.canteen]: { label: 'Столовая', color: '#C2703F', icon: UtensilsCrossed },
  [PoiType.park]: { label: 'Парк', color: '#2E9E6D', icon: Trees },
  [PoiType.other]: { label: 'Другое', color: '#6B7A8C', icon: MapPin },
}

export const TAG_LABELS: Record<string, string> = {
  wifi: 'Wi-Fi',
  розетки: 'Розетки',
  тихо: 'Тихо',
  бюджетно: 'Бюджетно',
}

export const AVAILABLE_TAGS = Object.keys(TAG_LABELS)
