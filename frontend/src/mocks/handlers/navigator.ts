import { http, HttpResponse } from 'msw'
import { PoiType } from '@/api/generated'
import type { Poi } from '@/api/generated'
import { estimateWalkingSeconds, haversineMeters } from '@/lib/geo'

const API = '*/api/v1'

// Точки вокруг Сеченовского Университета (Трубецкая ул., Хамовники) —
// реальный ориентир для правдоподобной демонстрации навигатора.
const poiList: Poi[] = [
  {
    id: 'poi-01',
    name: 'Coffee 8',
    type: PoiType.cafe,
    latitude: 55.7332,
    longitude: 37.5842,
    address: 'Большая Пироговская ул., 6',
    description: 'Кофейня рядом с главным корпусом, есть розетки у окна.',
    rating: 4.5,
    tags: ['wifi', 'розетки'],
    created_at: '2026-01-01T10:00:00Z',
  },
  {
    id: 'poi-02',
    name: 'Библиотека им.Engelhardt',
    type: PoiType.library,
    latitude: 55.7325,
    longitude: 37.5814,
    address: 'Трубецкая ул., 8, стр. 2',
    description: 'Читальный зал, тихо, работает до 22:00.',
    rating: 4.7,
    tags: ['тихо', 'wifi'],
    created_at: '2026-01-01T10:00:00Z',
  },
  {
    id: 'poi-03',
    name: 'Столовая №1',
    type: PoiType.canteen,
    latitude: 55.7336,
    longitude: 37.5825,
    address: 'Большая Пироговская ул., 2/6',
    description: 'Студенческая столовая, комплексный обед от 250 ₽.',
    rating: 3.9,
    tags: ['бюджетно'],
    created_at: '2026-01-01T10:00:00Z',
  },
  {
    id: 'poi-04',
    name: 'Coworking Hub Хамовники',
    type: PoiType.coworking,
    latitude: 55.7312,
    longitude: 37.5867,
    address: 'Усачёва ул., 33',
    description: 'Открытые места и переговорные, почасовая оплата.',
    rating: 4.3,
    tags: ['wifi', 'розетки', 'тихо'],
    created_at: '2026-01-01T10:00:00Z',
  },
  {
    id: 'poi-05',
    name: 'Парк Мандельштама',
    type: PoiType.park,
    latitude: 55.7300,
    longitude: 37.5795,
    address: 'Усачёва ул., 39',
    description: 'Небольшой сквер — удобно позаниматься на воздухе между парами.',
    rating: 4.2,
    tags: ['бюджетно'],
    created_at: '2026-01-01T10:00:00Z',
  },
  {
    id: 'poi-06',
    name: 'Андеграунд Кофе',
    type: PoiType.cafe,
    latitude: 55.7346,
    longitude: 37.5801,
    address: 'Абрикосовский пер., 2',
    description: 'Недорого и быстро, всегда очередь в обед.',
    rating: 4.0,
    tags: ['бюджетно', 'wifi'],
    created_at: '2026-01-01T10:00:00Z',
  },
  {
    id: 'poi-07',
    name: 'Научная библиотека МГМУ',
    type: PoiType.library,
    latitude: 55.7318,
    longitude: 37.5788,
    address: 'Большая Пироговская ул., 2, стр. 4',
    description: 'Абонемент учебной литературы и электронный каталог.',
    rating: 4.4,
    tags: ['тихо'],
    created_at: '2026-01-01T10:00:00Z',
  },
  {
    id: 'poi-08',
    name: 'Т-Кафе',
    type: PoiType.cafe,
    latitude: 55.7290,
    longitude: 37.5850,
    address: 'Погодинская ул., 10',
    description: 'Тихое место, можно долго сидеть с ноутбуком.',
    rating: 4.6,
    tags: ['wifi', 'розетки', 'тихо'],
    created_at: '2026-01-01T10:00:00Z',
  },
  {
    id: 'poi-09',
    name: 'Столовая «Клиника»',
    type: PoiType.canteen,
    latitude: 55.7350,
    longitude: 37.5845,
    address: 'Большая Пироговская ул., 6, стр. 1',
    description: 'При клинической базе, доступна студентам на практике.',
    rating: 3.7,
    tags: ['бюджетно'],
    created_at: '2026-01-01T10:00:00Z',
  },
  {
    id: 'poi-10',
    name: 'Новодевичьи пруды',
    type: PoiType.park,
    latitude: 55.7275,
    longitude: 37.5670,
    address: 'Лужнецкий проезд',
    description: 'Долгая прогулка, но красиво — вариант на большой перерыв.',
    rating: 4.8,
    tags: [],
    created_at: '2026-01-01T10:00:00Z',
  },
]

export const navigatorHandlers = [
  http.get(`${API}/map/poi`, ({ request }) => {
    const url = new URL(request.url)
    const type = url.searchParams.get('type')
    const tagsParam = url.searchParams.get('tags')
    const tags = tagsParam ? tagsParam.split(',').filter(Boolean) : []
    const lat = url.searchParams.get('lat')
    const lon = url.searchParams.get('lon')
    const radius = url.searchParams.get('radius')

    let filtered = poiList.filter((poi) => {
      if (type && poi.type !== type) return false
      if (tags.length > 0 && !tags.every((tag) => poi.tags?.includes(tag))) return false
      return true
    })

    if (lat && lon) {
      const origin = { lat: Number(lat), lon: Number(lon) }
      filtered = filtered
        .map((poi) => {
          const meters = haversineMeters(origin, { lat: poi.latitude!, lon: poi.longitude! })
          return {
            ...poi,
            distance_meters: Math.round(meters),
            walking_time_seconds: estimateWalkingSeconds(meters),
          }
        })
        .filter((poi) => (radius ? (poi.distance_meters ?? 0) <= Number(radius) : true))
        .sort((a, b) => (a.distance_meters ?? 0) - (b.distance_meters ?? 0))
    }

    return HttpResponse.json({ data: filtered })
  }),
]
