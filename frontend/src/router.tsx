import { createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import App from './App'
import AboutPage from './pages/AboutPage'
import Contact from './pages/Contact'
import Articles from './pages/blog/Articles'
import NotFound from './pages/not-found'

const rootRoute = createRootRoute({
  component: App,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: () => <div>Home Page</div>,
})

const aboutRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/about',
  component: AboutPage,
})

const contactRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/contact',
  component: Contact,
})

const blogRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/blog',
  component: Articles,
})

const notFoundRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '*',
  component: NotFound,
})

const routeTree = rootRoute.addChildren([
  indexRoute,
  aboutRoute,
  contactRoute,
  blogRoute,
  notFoundRoute,
])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
} 
