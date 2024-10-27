import { FooterSection } from "./Footer";
import { Navbar } from "./navbar";
import { MantineProvider } from '@mantine/core';
import { theme } from './theme';

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {

  return (
    <MantineProvider theme={theme}>
      <html
        lang="en"
        className={`bg-white dark:bg-gray-950 text-black dark:text-white `}
      >
        <body className="min-h-[100dvh]">

          <Navbar />

          <main className="max-w-6xl mx-auto px-2 sm:px-6">

            {children}

          </main>

          <FooterSection />
        </body>
      </html>
    </MantineProvider>
  );
}
