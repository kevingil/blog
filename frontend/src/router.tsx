import { createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import AboutPage from './pages/AboutPage'
import Contact from './pages/Contact'
import Articles from './pages/blog/Articles'
import NotFound from './pages/not-found'
import BlogDashboard from './pages/dashboard/blog/BlogDashboard'
import DashboardGeneral from './pages/dashboard/general/DashboardGeneral'
import DashboardSecurity from './pages/dashboard/security/DashboardSecurity'
import ArticleEditor from './components/blog/Editor'
import HomePage from './pages/HomePage'
import RootLayout from './pages/layout'

const rootRoute = createRootRoute({
  component: RootLayout,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: HomePage,
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
  validateSearch: (search: Record<string, unknown>) => ({
    page: search.page as string | undefined,
    tag: search.tag as string | undefined,
    search: search.search as string | undefined,
  }),
} as const)

const dashboardRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/dashboard',
  component: () => <div>Dashboard</div>,
})

const dashboardBlogRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/blog',
  component: BlogDashboard,
})

const dashboardBlogNewRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/blog/new',
  component: () => <ArticleEditor isNew={true} />,
})

const dashboardBlogEditRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/blog/edit/$slug',
  component: () => <ArticleEditor />,
})

const dashboardGeneralRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/general',
  component: DashboardGeneral,
})

const dashboardSecurityRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/security',
  component: DashboardSecurity,
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
  dashboardRoute.addChildren([
    dashboardBlogRoute,
    dashboardBlogNewRoute,
    dashboardBlogEditRoute,
    dashboardGeneralRoute,
    dashboardSecurityRoute,
  ]),
  notFoundRoute,
])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
} 
