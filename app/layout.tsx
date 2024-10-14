import './globals.css';
import type { Metadata, Viewport } from 'next';
import { Manrope } from 'next/font/google';
import { UserProvider } from '@/lib/auth';
import { getUser } from '@/db/queries';
import { FooterSection } from "@/components/layout/sections/footer";
import { Navbar } from "@/components/layout/navbar";
import { ThemeProvider } from "@/components/layout/theme-provider";

export const metadata: Metadata = {
  title: 'Kevin Gil',
  description: 'Software Engineer in San Francisco.',
};

export const viewport: Viewport = {
  maximumScale: 1,
};

const manrope = Manrope({ subsets: ['latin'] });

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {

  let userPromise = getUser();
  

  return (
    <html
      lang="en"
      className={`bg-white dark:bg-gray-950 text-black dark:text-white ${manrope.className}`}
    >
      <body className="min-h-[100dvh]">
        <UserProvider userPromise={userPromise}>


          <ThemeProvider
            attribute="class"
            defaultTheme="system"
            enableSystem
            disableTransitionOnChange
          >
            <Navbar />

            

            {children}
            <FooterSection />

          </ThemeProvider>

        </UserProvider>
      </body>
    </html>
  );
}
