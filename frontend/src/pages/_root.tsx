import { UserProvider } from '@/services/auth';
import { getUser } from '@/services/user';
import { FooterSection } from "@/components/footer";
import { Navbar } from "@/components/navbar";
import { ThemeProvider } from "@/components/theme-provider";
import 'highlight.js/styles/base16/snazzy.css';
import { Outlet } from '@tanstack/react-router';
import Aurora from '@/components/home/aurora';

export default function RootLayout() {
  const userPromise = getUser();

  return (
    <div className="min-h-[100dvh] flex flex-col relative">
      <UserProvider userPromise={userPromise}>
        <ThemeProvider>
          <Aurora />
          <Navbar />
            <main className="w-full max-w-6xl mx-auto px-2 sm:px-6 z-[1]" data-vaul-drawer-wrapper="">
              <Outlet />
            </main>
          <FooterSection />
        </ThemeProvider>
      </UserProvider>
    </div>
  );
}
