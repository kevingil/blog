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
      <html lang="en">
        <body>
          <Navbar />
          <main>
            {children}
          </main>
          <FooterSection />
        </body>
      </html>
    </MantineProvider>
  );
}
