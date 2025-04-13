import { UserProvider } from '@/services/auth';
import { getUser } from '@/services/user';
import { FooterSection } from "@/components/footer";
import { Navbar } from "@/components/navbar";
import { ThemeProvider } from "@/components/theme-provider";
import Aurora from "@/components/home/aurora";
import 'highlight.js/styles/base16/snazzy.css';
import { CopilotKit } from '@copilotkit/react-core';
import { Outlet } from '@tanstack/react-router';
import '@/index.css';

export default function RootLayout() {
  const userPromise = getUser();

  return (
    <div className="min-h-[100dvh] flex flex-col relative">
      <UserProvider userPromise={userPromise}>
        <ThemeProvider>
          <Navbar />
          <Aurora />
          <CopilotKit runtimeUrl="/api/copilotkit">
            <main className="w-full max-w-6xl mx-auto px-2 sm:px-6 z-[1]" data-vaul-drawer-wrapper="">
              <Outlet />
            </main>
          </CopilotKit>
          <FooterSection />
        </ThemeProvider>
      </UserProvider>
    </div>
  );
}
