import { RouterProvider } from '@tanstack/react-router'
import { router } from './router'
import { Suspense } from 'react'
import { ThemeProvider } from "@/components/theme-provider"
import '@/index.css'


function App() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
        <RouterProvider router={router} />
      </ThemeProvider>
    </Suspense>
  )
}

export default App
