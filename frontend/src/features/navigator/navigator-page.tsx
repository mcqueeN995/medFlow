import { useEffect, useMemo, useRef, useState } from 'react'
import { YMaps, Map as YMap, Placemark, ZoomControl } from '@pbe/react-yandex-maps'
import type { Map as YMapInstance } from 'yandex-maps'
import { ExternalLink, LocateFixed, Star, X } from 'lucide-react'
import { getMapPoi } from '@/api/generated/medFlowAPI'
import { PoiType } from '@/api/generated'
import type { Poi } from '@/api/generated'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { formatDistance, formatWalkingTime } from '@/lib/geo'
import { AVAILABLE_TAGS, POI_TYPE_META, TAG_LABELS } from './poi-meta'
import { getPoiIcon, getUserLocationIcon } from './poi-marker-icon'
import { useGeolocation } from './use-geolocation'

const ALL = '__all__'
// Сеченовский Университет, Хамовники — центр карты по умолчанию, пока нет
// геолокации пользователя.
const DEFAULT_CENTER: [number, number] = [55.7325, 37.582]

const YANDEX_MAPS_API_KEY = import.meta.env.VITE_YANDEX_MAPS_API_KEY as string | undefined

export function NavigatorPage() {
  const [type, setType] = useState(ALL)
  const [activeTags, setActiveTags] = useState<string[]>([])
  const [poi, setPoi] = useState<Poi[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const { position, loading: locating, error: geoError, locate } = useGeolocation()
  const mapRef = useRef<YMapInstance | null>(null)

  useEffect(() => {
    setLoading(true)
    getMapPoi({
      type: type === ALL ? undefined : (type as PoiType),
      tags: activeTags.length ? activeTags.join(',') : undefined,
      lat: position?.lat,
      lon: position?.lon,
    })
      .then((res) => setPoi(res.data ?? []))
      .finally(() => setLoading(false))
  }, [type, activeTags, position])

  const selected = useMemo(() => poi.find((p) => p.id === selectedId) ?? null, [poi, selectedId])
  const mapCenter: [number, number] = position
    ? [position.lat, position.lon]
    : selected
      ? [selected.latitude!, selected.longitude!]
      : DEFAULT_CENTER

  useEffect(() => {
    mapRef.current?.setCenter(mapCenter, selected ? 17 : 15, { duration: 600 })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mapCenter[0], mapCenter[1], selected])

  function toggleTag(tag: string) {
    setActiveTags((prev) => (prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag]))
  }

  return (
    <div className="flex h-[calc(100dvh-5rem)] flex-col overflow-hidden md:h-dvh md:flex-row">
      <div className="flex h-[45%] min-h-0 w-full flex-col border-b border-border bg-card md:h-full md:w-96 md:border-r md:border-b-0">
        <div className="flex flex-col gap-3 border-b border-border p-4">
          <div className="flex items-center justify-between">
            <h1 className="text-lg font-bold text-primary">Навигатор перерывов</h1>
            <Button
              size="sm"
              variant="outline"
              className="rounded-full"
              disabled={locating}
              onClick={locate}
            >
              <LocateFixed className="size-4" />
              {locating ? 'Ищем…' : 'Рядом со мной'}
            </Button>
          </div>
          {geoError && <p className="text-xs text-destructive">{geoError}</p>}

          <Select value={type} onValueChange={(v) => setType(v ?? ALL)}>
            <SelectTrigger className="h-9 w-full rounded-full">
              <SelectValue placeholder="Тип">
                {(v: string) => (v === ALL ? 'Все типы' : POI_TYPE_META[v as PoiType].label)}
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>Все типы</SelectItem>
              {Object.entries(POI_TYPE_META).map(([value, meta]) => (
                <SelectItem key={value} value={value}>{meta.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>

          <div className="flex flex-wrap gap-1.5">
            {AVAILABLE_TAGS.map((tag) => (
              <button
                key={tag}
                onClick={() => toggleTag(tag)}
                className={cn(
                  'rounded-full border px-2.5 py-1 text-xs transition-colors',
                  activeTags.includes(tag)
                    ? 'border-accent bg-accent text-accent-foreground'
                    : 'border-border bg-transparent text-muted-foreground hover:text-foreground',
                )}
              >
                {TAG_LABELS[tag]}
              </button>
            ))}
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto">
          {loading ? (
            <div className="flex flex-col gap-2 p-4">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-16 rounded-xl" />
              ))}
            </div>
          ) : poi.length === 0 ? (
            <p className="p-4 text-sm text-muted-foreground">Ничего не найдено по выбранным фильтрам</p>
          ) : (
            <ul className="flex flex-col divide-y divide-border">
              {poi.map((p) => {
                const meta = POI_TYPE_META[p.type ?? PoiType.other]
                const Icon = meta.icon
                return (
                  <li key={p.id}>
                    <button
                      onClick={() => setSelectedId(p.id ?? null)}
                      className={cn(
                        'flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-secondary',
                        selectedId === p.id && 'bg-secondary',
                      )}
                    >
                      <span
                        className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-full"
                        style={{ background: meta.color }}
                      >
                        <Icon className="size-4 text-white" />
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="block truncate text-sm font-medium text-foreground">{p.name}</span>
                        <span className="block truncate text-xs text-muted-foreground">{p.address}</span>
                        {p.distance_meters != null && (
                          <span className="mt-0.5 block text-xs text-accent">
                            {formatDistance(p.distance_meters)} · {formatWalkingTime(p.walking_time_seconds)}
                          </span>
                        )}
                      </span>
                    </button>
                  </li>
                )
              })}
            </ul>
          )}
        </div>
      </div>

      <div className="relative min-h-0 flex-1">
        {YANDEX_MAPS_API_KEY ? (
          <YMaps query={{ apikey: YANDEX_MAPS_API_KEY, lang: 'ru_RU' }}>
            <YMap
              defaultState={{ center: DEFAULT_CENTER, zoom: 15 }}
              width="100%"
              height="100%"
              modules={['control.ZoomControl']}
              instanceRef={(ref: YMapInstance | null) => {
                mapRef.current = ref
              }}
              options={{ suppressMapOpenBlock: true }}
            >
              <ZoomControl options={{ position: { right: 10, top: 10 } }} />
              {position && (
                <Placemark geometry={[position.lat, position.lon]} options={getUserLocationIcon()} />
              )}
              {poi.map((p) => (
                <Placemark
                  key={p.id}
                  geometry={[p.latitude!, p.longitude!]}
                  options={{ iconLayout: 'default#image', ...getPoiIcon(p.type ?? PoiType.other, p.id === selectedId) }}
                  onClick={() => setSelectedId(p.id ?? null)}
                />
              ))}
            </YMap>
          </YMaps>
        ) : (
          <div className="flex h-full w-full items-center justify-center bg-secondary p-6 text-center text-sm text-muted-foreground">
            Карта недоступна: не задан VITE_YANDEX_MAPS_API_KEY. Получите бесплатный ключ на
            developer.tech.yandex.ru и добавьте его в frontend/.env.local.
          </div>
        )}

        {selected && (
          <div className="absolute inset-x-3 bottom-3 z-400 rounded-2xl border border-border bg-card p-4 shadow-lg md:left-3 md:right-auto md:w-80">
            <button
              onClick={() => setSelectedId(null)}
              className="absolute top-3 right-3 text-muted-foreground hover:text-foreground"
              aria-label="Закрыть"
            >
              <X className="size-4" />
            </button>
            <p className="pr-6 font-semibold text-foreground">{selected.name}</p>
            <p className="text-xs text-muted-foreground">{selected.address}</p>
            {selected.description && <p className="mt-1.5 text-sm text-muted-foreground">{selected.description}</p>}
            <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
              {selected.rating != null && (
                <span className="flex items-center gap-0.5">
                  <Star className="size-3.5 fill-current text-amber-500 text-amber-500" /> {selected.rating.toFixed(1)}
                </span>
              )}
              {selected.distance_meters != null && (
                <span>{formatDistance(selected.distance_meters)} · {formatWalkingTime(selected.walking_time_seconds)}</span>
              )}
            </div>
            <a
              href={`https://yandex.ru/maps/?rtext=~${selected.latitude},${selected.longitude}&rtt=pd`}
              target="_blank"
              rel="noopener noreferrer"
              className="mt-3 inline-flex h-9 items-center gap-1.5 rounded-full bg-linear-to-r from-primary to-accent px-4 text-sm font-medium text-primary-foreground"
            >
              <ExternalLink className="size-4" /> Маршрут в Яндекс.Картах
            </a>
          </div>
        )}
      </div>
    </div>
  )
}
