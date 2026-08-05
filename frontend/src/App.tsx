import { RouterProvider } from 'react-router-dom'
import { ThemeProvider } from 'next-themes'
import { Toaster } from '@/components/ui/sonner'
import { router } from '@/app/router'

function App() {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem themes={['light', 'dim', 'dark']}>
      <RouterProvider router={router} />
      <Toaster position="top-center" richColors />
    </ThemeProvider>
  )
}

export default App
