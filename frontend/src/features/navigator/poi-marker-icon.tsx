import { renderToStaticMarkup } from 'react-dom/server'
import L from 'leaflet'
import { PoiType } from '@/api/generated'
import { POI_TYPE_META } from './poi-meta'

// L.divIcon рендерит произвольный HTML — переиспользуем те же lucide-иконки,
// что и в списке, вместо набора вручную нарисованных SVG под каждый тип POI.
function buildIcon(type: PoiType, selected: boolean): L.DivIcon {
  const meta = POI_TYPE_META[type]
  const Icon = meta.icon
  const size = selected ? 36 : 30
  const html = renderToStaticMarkup(
    <div
      style={{
        width: size,
        height: size,
        borderRadius: '9999px',
        background: meta.color,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        boxShadow: selected ? '0 0 0 4px rgba(255,255,255,0.9), 0 2px 8px rgba(0,0,0,0.35)' : '0 1px 4px rgba(0,0,0,0.35)',
        border: '2px solid white',
      }}
    >
      <Icon color="white" size={size * 0.55} strokeWidth={2.25} />
    </div>,
  )
  return L.divIcon({
    html,
    className: '',
    iconSize: [size, size],
    iconAnchor: [size / 2, size / 2],
  })
}

const cache = new Map<string, L.DivIcon>()

export function getPoiIcon(type: PoiType, selected = false): L.DivIcon {
  const key = `${type}-${selected}`
  if (!cache.has(key)) cache.set(key, buildIcon(type, selected))
  return cache.get(key)!
}

export function getUserLocationIcon(): L.DivIcon {
  const html = renderToStaticMarkup(
    <div style={{ position: 'relative', width: 20, height: 20 }}>
      <div style={{ position: 'absolute', inset: 0, borderRadius: '9999px', background: 'rgba(62,140,158,0.35)' }} />
      <div
        style={{
          position: 'absolute',
          inset: 5,
          borderRadius: '9999px',
          background: '#3E8C9E',
          border: '2px solid white',
          boxShadow: '0 1px 4px rgba(0,0,0,0.4)',
        }}
      />
    </div>,
  )
  return L.divIcon({ html, className: '', iconSize: [20, 20], iconAnchor: [10, 10] })
}
