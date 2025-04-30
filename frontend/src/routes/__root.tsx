import { FooterSection } from "@/components/footer";
import { Navbar } from "@/components/navbar";
import { ThemeProvider } from "@/components/theme-provider";
import 'highlight.js/styles/base16/snazzy.css';
import { Outlet } from '@tanstack/react-router';
import Aurora from '@/components/home/aurora';
import { createRootRoute } from '@tanstack/react-router';
import { Suspense } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const queryClient = new QueryClient();

export const Route = createRootRoute({
  component: RootLayout,
});

function RootLayout() {

  return (
    <QueryClientProvider client={queryClient}>
      <div className="min-h-[100dvh] flex flex-col relative">
        <Suspense fallback={<div>Loading...</div>}>
          <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
              <ThemeProvider>
                <Aurora />
                <Navbar />
                <main className="w-full max-w-6xl mx-auto px-2 sm:px-6 z-[1]" data-vaul-drawer-wrapper="">
                  <Outlet />
                </main>
                <FooterSection />
              </ThemeProvider>
          </ThemeProvider>
        </Suspense>
      </div>
    </QueryClientProvider>
  );
}
