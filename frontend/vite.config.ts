import path from 'node:path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    // injectManifest (не generateSW) - нужен кастомный src/sw.ts с
    // обработчиками push/notificationclick, а не только автогенерируемый
    // прекэш. devOptions.enabled: false - SW собирается и регистрируется
    // только в прод-сборке, чтобы не конфликтовать по scope с MSW-воркером
    // в dev-режиме (см. VITE_USE_MOCKS). Имена файлов (sw.js,
    // manifest.webmanifest) фиксированы под уже существующие правила
    // кэширования в nginx.conf - менять нельзя.
    VitePWA({
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'sw.ts',
      injectRegister: false,
      devOptions: { enabled: false },
      manifest: {
        name: 'medFlow',
        short_name: 'medFlow',
        description: 'Библиотека учебников, ИИ-карточки, карта кампуса и форум для студентов-медиков',
        lang: 'ru',
        start_url: '/',
        display: 'standalone',
        background_color: '#eaf2ff',
        theme_color: '#7e14ff',
        icons: [
          { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png' },
          { src: '/icons/icon-512-maskable.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
      injectManifest: {
        globPatterns: ['**/*.{js,css,html,svg,png,woff2}'],
      },
    }),
  ],
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
})
