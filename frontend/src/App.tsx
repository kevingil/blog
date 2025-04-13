import { RouterProvider } from '@tanstack/react-router'
import { router } from './router'
import './App.css'
import { Suspense } from 'react'

function App() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <RouterProvider router={router} />
    </Suspense>
  )
}

export default App
