import { createRootRoute, createRouter } from '@tanstack/react-router'
import RootLayout from './pages/layout'
import NotFound from './pages/not-found'
import { indexRoute } from './pages/index'

// Create a root route
const rootRoute = createRootRoute({
  component: RootLayout,
  notFoundComponent: NotFound,
})

// Create the route tree
const routeTree = rootRoute.addChildren([
  indexRoute,
])

// Create the router
export const router = createRouter({ routeTree })

// Register the router for type safety
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
} 
