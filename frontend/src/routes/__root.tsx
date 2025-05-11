import { FooterSection } from "@/components/footer";
import { Navbar } from "@/components/navbar";
import { ThemeProvider } from "@/components/theme-provider";
import 'highlight.js/styles/base16/snazzy.css';
import { createRootRouteWithContext, Outlet } from '@tanstack/react-router';
import Aurora from '@/components/home/aurora';
import { Suspense } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { AuthContext } from "@/services/auth/auth";
import { AuthProvider } from "@/services/auth/auth";

const queryClient = new QueryClient();

interface MyRouterContext {
  auth: AuthContext;
}

function RootLayout() {

  return (
    <div className="min-h-[100dvh] flex flex-col relative">
      <Suspense fallback={<div>Loading...</div>}>
        <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
          <ThemeProvider>
            <Aurora />
            <Navbar />
            <main className="w-full max-w-7xl mx-auto px-2 sm:px-6 z-[1]" data-vaul-drawer-wrapper="">
              <Outlet />
            </main>
            <FooterSection />
          </ThemeProvider>
        </ThemeProvider>
      </Suspense>
    </div>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <RootLayout />
      </AuthProvider>
    </QueryClientProvider>
  );
}

export const Route = createRootRouteWithContext<MyRouterContext>()({
  component: App,
});
