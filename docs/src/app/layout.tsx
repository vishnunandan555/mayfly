import { RootProvider } from 'fumadocs-ui/provider';
import { Inter, JetBrains_Mono, Instrument_Serif } from 'next/font/google';
import type { ReactNode } from 'react';
import type { Metadata } from 'next';
import './global.css';

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-inter',
  display: 'swap',
});

const jetbrains = JetBrains_Mono({
  subsets: ['latin'],
  variable: '--font-jetbrains',
  display: 'swap',
});

const instrumentSerif = Instrument_Serif({
  subsets: ['latin'],
  weight: '400',
  variable: '--font-serif',
  display: 'swap',
});

export const metadata: Metadata = {
  title: {
    template: '%s | MayFly Documentation',
    default: 'MayFly — Zero-Dependency Secrets Manager',
  },
  description:
    'A zero-dependency, Go-only local secrets manager that keeps secrets out of .env files by storing them in an encrypted vault and injecting them directly into launched process memory.',
  metadataBase: new URL('https://mayfly.dev'),
};

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html
      lang="en"
      className={`${inter.variable} ${jetbrains.variable} ${instrumentSerif.variable}`}
      suppressHydrationWarning
    >
      <body className="flex min-h-screen flex-col bg-background font-sans antialiased">
        <RootProvider>{children}</RootProvider>
      </body>
    </html>
  );
}
