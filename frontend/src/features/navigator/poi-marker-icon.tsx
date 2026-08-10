import { renderToStaticMarkup } from 'react-dom/server'
import { PoiType } from '@/api/generated'
import { POI_TYPE_META } from './poi-meta'

// Яндекс.Карты (iconLayout: 'default#image') принимают маркер как обычную
// <img>-картинку — SVG data-URI позволяет переиспользовать те же lucide-иконки,
// что и в списке, без обращения к глобальному объекту ymaps.templateLayoutFactory.
function extractSvgInner(svg: string): string {
  return svg.replace(/^<svg[^>]*>/, '').replace(/<\/svg>$/, '')
}

function buildMarkerSvg(color: string, Icon: PoiTypeIcon, size: number, selected: boolean): string {
  const iconSize = size * 0.55
  const offset = (size - iconSize) / 2
  const iconMarkup = extractSvgInner(
    renderToStaticMarkup(<Icon color="white" size={iconSize} strokeWidth={2.25} />),
  )
  const ring = selected ? `<circle cx="${size / 2}" cy="${size / 2}" r="${size / 2}" fill="white" opacity="0.9" />` : ''
  const shadowId = 'ds'

  return `<svg xmlns="http://www.w3.org/2000/svg" width="${size + 4}" height="${size + 4}" viewBox="0 0 ${size + 4} ${size + 4}">
    <defs>
      <filter id="${shadowId}" x="-50%" y="-50%" width="200%" height="200%">
        <feDropShadow dx="0" dy="1" stdDeviation="1.5" flood-color="#000" flood-opacity="0.35" />
      </filter>
    </defs>
    <g transform="translate(2,2)" filter="url(#${shadowId})">
      ${ring}
      <circle cx="${size / 2}" cy="${size / 2}" r="${size / 2 - 2}" fill="${color}" stroke="white" stroke-width="2" />
      <g transform="translate(${offset},${offset})">${iconMarkup}</g>
    </g>
  </svg>`
}

function toDataUri(svg: string): string {
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

type PoiTypeIcon = (typeof POI_TYPE_META)[PoiType]['icon']

export interface MarkerIcon {
  iconImageHref: string
  iconImageSize: [number, number]
  iconImageOffset: [number, number]
}

const cache = new Map<string, MarkerIcon>()

export function getPoiIcon(type: PoiType, selected = false): MarkerIcon {
  const key = `${type}-${selected}`
  if (!cache.has(key)) {
    const meta = POI_TYPE_META[type]
    const size = selected ? 36 : 30
    const full = size + 4
    cache.set(key, {
      iconImageHref: toDataUri(buildMarkerSvg(meta.color, meta.icon, size, selected)),
      iconImageSize: [full, full],
      iconImageOffset: [-full / 2, -full / 2],
    })
  }
  return cache.get(key)!
}

let userLocationIcon: MarkerIcon | null = null

export function getUserLocationIcon(): MarkerIcon {
  if (!userLocationIcon) {
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 20 20">
      <circle cx="10" cy="10" r="10" fill="rgba(62,140,158,0.35)" />
      <circle cx="10" cy="10" r="5" fill="#3E8C9E" stroke="white" stroke-width="2" />
    </svg>`
    userLocationIcon = { iconImageHref: toDataUri(svg), iconImageSize: [20, 20], iconImageOffset: [-10, -10] }
  }
  return userLocationIcon
}
