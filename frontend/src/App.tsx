import { RouterProvider } from 'react-router-dom'
import { ThemeProvider } from 'next-themes'
import { Toaster } from '@/components/ui/sonner'
import { OfflineBanner } from '@/components/shared/offline-banner'
import { router } from '@/app/router'
import { useReviewQueueSync } from '@/lib/sync-review-queue'

function App() {
  useReviewQueueSync()

  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem themes={['light', 'dim', 'dark']}>
      <OfflineBanner />
      <RouterProvider router={router} />
      <Toaster position="top-center" richColors />
    </ThemeProvider>
  )
}

export default App
