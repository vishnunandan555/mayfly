import type { MetadataRoute } from 'next';
import { source } from '@/lib/source';

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const baseUrl =
    process.env.NEXT_PUBLIC_SITE_URL ||
    (process.env.VERCEL_PROJECT_PRODUCTION_URL
      ? `https://${process.env.VERCEL_PROJECT_PRODUCTION_URL}`
      : process.env.VERCEL_URL
        ? `https://${process.env.VERCEL_URL}`
        : 'https://mayfly-docs.vercel.app');
  const currentDate = new Date();

  const routes: MetadataRoute.Sitemap = [
    {
      url: baseUrl,
      lastModified: currentDate,
      changeFrequency: 'daily',
      priority: 1.0,
    },
    {
      url: `${baseUrl}/docs`,
      lastModified: currentDate,
      changeFrequency: 'daily',
      priority: 0.9,
    },
  ];

  const pages = source.getPages();
  for (const page of pages) {
    if (page.url === '/docs') continue;
    routes.push({
      url: `${baseUrl}${page.url}`,
      lastModified: currentDate,
      changeFrequency: 'weekly',
      priority: 0.8,
    });
  }

  return routes;
}
