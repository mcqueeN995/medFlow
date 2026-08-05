const EARTH_RADIUS_M = 6371000

// Backend считает расстояние по формуле Haversine (см. Полная спецификация
// проекта, раздел "Архитектурные решения") — дублируем здесь для клиентского
// мока и на случай, если lat/lon пользователя ещё не отправлены на сервер.
export function haversineMeters(a: { lat: number; lon: number }, b: { lat: number; lon: number }): number {
  const toRad = (deg: number) => (deg * Math.PI) / 180
  const dLat = toRad(b.lat - a.lat)
  const dLon = toRad(b.lon - a.lon)
  const sinLat = Math.sin(dLat / 2)
  const sinLon = Math.sin(dLon / 2)
  const h = sinLat * sinLat + Math.cos(toRad(a.lat)) * Math.cos(toRad(b.lat)) * sinLon * sinLon
  return 2 * EARTH_RADIUS_M * Math.asin(Math.sqrt(h))
}

const AVERAGE_WALKING_SPEED_MPS = 1.35 // ~4.9 км/ч

export function estimateWalkingSeconds(meters: number): number {
  return Math.round(meters / AVERAGE_WALKING_SPEED_MPS)
}

export function formatDistance(meters?: number | null): string {
  if (meters == null) return ''
  if (meters < 1000) return `${Math.round(meters)} м`
  return `${(meters / 1000).toFixed(1)} км`
}

export function formatWalkingTime(seconds?: number | null): string {
  if (seconds == null) return ''
  const minutes = Math.round(seconds / 60)
  if (minutes < 1) return '<1 мин пешком'
  return `${minutes} мин пешком`
}
