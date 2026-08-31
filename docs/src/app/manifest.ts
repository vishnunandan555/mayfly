import type { MetadataRoute } from 'next';

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: 'MayFly: Zero-Disk Secrets Manager & Process Injector',
    short_name: 'MayFly',
    description:
      'Zero-dependency local secrets manager that encrypts API keys and injects them directly into application memory.',
    start_url: '/',
    display: 'standalone',
    background_color: '#0a0d0c',
    theme_color: '#f59e0b',
    icons: [
      {
        src: '/icon.png',
        sizes: '512x512',
        type: 'image/png',
        purpose: 'any',
      },
    ],
  };
}
