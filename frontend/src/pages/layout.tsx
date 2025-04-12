import './globals.css';
import { UserProvider } from '../lib/auth';
import { getUser } from '../db/queries';
import { FooterSection } from "../components/footer";
import { Navbar } from "../components/navbar";
import { ThemeProvider } from "../components/home/theme-provider";
import Aurora from "../components/home/aurora";
import 'highlight.js/styles/base16/snazzy.css';
import { CopilotKit } from '@copilotkit/react-core';
import { Outlet } from '@tanstack/react-router';

export const siteData = {
  title: 'Kevin Gil',
  description: 'Software Engineer in San Francisco.',
};

export default function RootLayout() {
  const userPromise = getUser();

  return (
    <div className="min-h-[100dvh] flex flex-col relative">
      <UserProvider userPromise={userPromise}>
        <ThemeProvider
          attribute="class"
          defaultTheme="light"
          enableSystem={false}
          disableTransitionOnChange
        >
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
