import { FooterSection } from "@/components/footer";
import { Navbar } from "@/components/navbar";
import Aurora from '@/components/home/aurora';
import { Suspense } from 'react';
import { AuthProvider } from "@/services/auth/auth";
import { createFileRoute, Outlet } from '@tanstack/react-router';
import { SpiralGalaxyAnimation } from "@/components/home/galaxy";

function PublicLayout() {
  return (
    <div className="min-h-[100dvh] flex flex-col relative">
      <Suspense fallback={<div>Loading...</div>}>
        <AuthProvider>
          {/* <Aurora /> */}
          <SpiralGalaxyAnimation />
          <Navbar />
          <main className="w-full max-w-7xl mx-auto px-2 sm:px-6 z-[1]" data-vaul-drawer-wrapper="">
            <Outlet />
          </main>
          <FooterSection />
        </AuthProvider>
      </Suspense>
    </div>
  );
}

export const Route = createFileRoute('/_publicLayout')({
  component: PublicLayout,
}); 
