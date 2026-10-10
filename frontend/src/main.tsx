import React from 'react'
import ReactDOM from 'react-dom/client'

import './index.css'

import { RouterProvider, createRouter } from '@tanstack/react-router'
import { routeTree } from './routeTree.gen'
import { trackPageView } from './lib/analytics'
import { AuthProvider, useAuthContext } from './services/auth/auth'

const router = createRouter({
  routeTree,
  context: {
    auth: undefined!, // RouterWithAuth supplies the live auth context
  },
  defaultErrorComponent: ({ error }) => {
    console.error("Error:", error);
    return null;
  },
})

router.subscribe("onResolved", ({ toLocation }) => {
  trackPageView(`${toLocation.pathname}${toLocation.searchStr}`);
});

// Register the router instance for type safety
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

function RouterWithAuth() {
  const auth = useAuthContext();
  const signedIn = React.useRef(auth.isAuthenticated);

  // Guards only run on navigation. Re-run them when the session starts or ends
  // so a dead session leaves the dashboard at once.
  React.useEffect(() => {
    if (signedIn.current === auth.isAuthenticated) {
      return;
    }
    signedIn.current = auth.isAuthenticated;
    void router.invalidate();
  }, [auth.isAuthenticated]);

  return <RouterProvider router={router} context={{ auth }} />;
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
