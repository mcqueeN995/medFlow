import { useState } from 'react'

interface GeoState {
  position: { lat: number; lon: number } | null
  loading: boolean
  error: string | null
}

export function useGeolocation() {
  const [state, setState] = useState<GeoState>({ position: null, loading: false, error: null })

  function locate() {
    if (!navigator.geolocation) {
      setState({ position: null, loading: false, error: 'Геолокация не поддерживается браузером' })
      return
    }
    setState((s) => ({ ...s, loading: true, error: null }))
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        setState({
          position: { lat: pos.coords.latitude, lon: pos.coords.longitude },
          loading: false,
          error: null,
        })
      },
      (err) => {
        const message =
          err.code === err.PERMISSION_DENIED
            ? 'Доступ к геолокации запрещён — разрешите в настройках браузера'
            : 'Не удалось определить местоположение'
        setState({ position: null, loading: false, error: message })
      },
      { enableHighAccuracy: true, timeout: 10000 },
    )
  }

  return { ...state, locate }
}
