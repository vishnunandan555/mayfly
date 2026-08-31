import { RootProvider } from 'fumadocs-ui/provider';
import { Inter, JetBrains_Mono, Instrument_Serif, Space_Grotesk } from 'next/font/google';
import type { ReactNode } from 'react';
import type { Metadata } from 'next';
import './global.css';

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-inter',
  display: 'swap',
});

const spaceGrotesk = Space_Grotesk({
  subsets: ['latin'],
  variable: '--font-space-grotesk',
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
    default: 'MayFly — Zero-Disk Secrets Manager & In-Memory Process Injector',
  },
  description:
    'Never save API keys in plaintext .env files again. MayFly encrypts your secrets locally with AES-256-GCM and injects them directly into your application memory in pure Go standard library with zero external dependencies.',
  keywords: [
    'secrets manager',
    'zero dependency',
    'go stdlib',
    'process injection',
    'developer security',
    'prevent env leak',
    'aes-256-gcm',
    'pbkdf2',
    'supply chain security',
    'terminal ui',
    'reproducible build',
  ],
  authors: [{ name: 'MayFly Team', url: 'https://github.com/vishnunandan555/mayfly' }],
  creator: 'MayFly Team',
  publisher: 'MayFly',
  metadataBase: new URL('https://mayfly.dev'),
  alternates: {
    canonical: '/',
  },
  openGraph: {
    title: 'MayFly — Zero-Disk Secrets Manager & In-Memory Process Injector',
    description:
      'Stop storing API keys in plaintext .env files. MayFly encrypts credentials locally and injects them directly into application RAM with zero third-party dependencies.',
    url: 'https://mayfly.dev',
    siteName: 'MayFly',
    images: [
      {
        url: '/icon.png',
        width: 512,
        height: 512,
        alt: 'MayFly Logo',
      },
    ],
    locale: 'en_US',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MayFly — Zero-Disk Secrets Manager',
    description:
      'Never write .env files to disk again. Encrypted local vault with in-memory process injection in pure Go stdlib (0 dependencies).',
    images: ['/icon.png'],
    creator: '@mayfly_sec',
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      'max-video-preview': -1,
      'max-image-preview': 'large',
      'max-snippet': -1,
    },
  },
};

const jsonLd = {
  '@context': 'https://schema.org',
  '@type': 'SoftwareApplication',
  name: 'MayFly',
  operatingSystem: 'Linux, macOS, Windows',
  applicationCategory: 'DeveloperApplication, SecurityApplication',
  description:
    'A zero-dependency local secrets manager that encrypts API keys and injects them directly into application memory without writing plaintext .env files to disk.',
  url: 'https://mayfly.dev',
  license: 'https://opensource.org/licenses/AGPL-3.0',
  offers: {
    '@type': 'Offer',
    price: '0',
    priceCurrency: 'USD',
  },
  author: {
    '@type': 'Person',
    name: 'Vishnu Nandan',
    url: 'https://github.com/vishnunandan555',
  },
};

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html
      lang="en"
      className={`${inter.variable} ${spaceGrotesk.variable} ${jetbrains.variable} ${instrumentSerif.variable}`}
      suppressHydrationWarning
    >
      <head>
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
        />
      </head>
      <body className="flex min-h-screen flex-col bg-background font-sans antialiased">
        <RootProvider>{children}</RootProvider>
      </body>
    </html>
  );
}
