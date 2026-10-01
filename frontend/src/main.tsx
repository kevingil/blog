import React from 'react'
import ReactDOM from 'react-dom/client'

import './index.css'

import { RouterProvider, createRouter } from '@tanstack/react-router'
import { routeTree } from './routeTree.gen'
import { trackPageView } from './lib/analytics'
import { AuthProvider, useAuthContext } from './services/auth/auth'

// Create a router instance
const router = createRouter({
  routeTree,
  context: {
    auth: undefined!, // This will be properly set in the RouterWithAuth component
  },
})

// Register the router instance for type safety
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

// Wrap the RouterProvider with AuthProvider to make auth available to routes
function RouterWithAuth() {
  const auth = useAuthContext();
  
  // Create router with auth context
  const routerWithAuth = React.useMemo(() => {
    const nextRouter = createRouter({
      routeTree,
      context: {
        auth,
      },
      defaultErrorComponent: ({ error }) => {
        if (error.message?.includes("Cannot read properties of undefined")) {
          // Handle auth context errors
          console.error("Auth context error:", error);
        }
        console.error("Error:", error);
      },
    });
    nextRouter.subscribe("onResolved", ({ toLocation }) => {
      trackPageView(`${toLocation.pathname}${toLocation.searchStr}`);
    });
    return nextRouter;
  }, [auth]);

  return <RouterProvider router={routerWithAuth} />;
}

const rootElement = document.getElementById('root')!

if (!rootElement.innerHTML) {
  const root = ReactDOM.createRoot(rootElement)
  root.render(
    <React.StrictMode>
      <AuthProvider>
        <RouterWithAuth />
      </AuthProvider>
    </React.StrictMode>
  )
}
