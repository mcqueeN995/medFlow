/// <reference lib="webworker" />
import { clientsClaim } from 'workbox-core'
import { createHandlerBoundToURL, precacheAndRoute } from 'workbox-precaching'
import { NavigationRoute, registerRoute } from 'workbox-routing'

declare const self: ServiceWorkerGlobalScope

// Без этого новая версия SW после деплоя остаётся "waiting", пока
// пользователь не закроет все вкладки сайта - активная старая версия
// продолжает отдавать закэшированный (устаревший) index.html/бандл сколь
// угодно долго. skipWaiting + clientsClaim - новая версия берёт управление
// сразу на следующей загрузке.
self.skipWaiting()
clientsClaim()

// Прекэш app shell из injectManifest - даёт офлайн-открытие приложения
// (усиливает офлайн-баннер: даже без сети сама PWA загружается).
precacheAndRoute(self.__WB_MANIFEST)

// SPA-роутинг: любой прямой переход/открытие по URL вида /cards/review,
// /forum/threads/:id и т.п. без сети должен отдавать закэшированный
// index.html (дальше маршрутизацию решает React Router на клиенте) - без
// этого фолбэка офлайн работает только та единственная страница, что была
// открыта последней. /api и файлы с расширением (assets) не перехватываем.
registerRoute(
  new NavigationRoute(createHandlerBoundToURL('/index.html'), {
    denylist: [/^\/api\//, /\/[^/?]+\.[^/]+$/],
  }),
)

interface PushPayload {
  title: string
  message: string
  kind: string
}

self.addEventListener('push', (event) => {
  if (!event.data) return
  const payload = event.data.json() as PushPayload
  event.waitUntil(
    self.registration.showNotification(payload.title, {
      body: payload.message,
      icon: '/icons/icon-192.png',
      badge: '/icons/icon-192.png',
      data: { kind: payload.kind },
    }),
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
      const existing = clients.find((c) => 'focus' in c)
      if (existing) return (existing as WindowClient).focus()
      return self.clients.openWindow('/')
    }),
  )
})
