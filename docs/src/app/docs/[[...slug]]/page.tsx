import { source } from '@/lib/source';
import {
  DocsBody,
  DocsDescription,
  DocsPage,
  DocsTitle,
} from 'fumadocs-ui/page';
import { notFound } from 'next/navigation';
import defaultMdxComponents from 'fumadocs-ui/mdx';
import { Tab, Tabs } from 'fumadocs-ui/components/tabs';
import { Callout } from 'fumadocs-ui/components/callout';
import { Step, Steps } from 'fumadocs-ui/components/steps';
import { Card, Cards } from 'fumadocs-ui/components/card';
import type { Metadata } from 'next';

interface PageProps {
  params: Promise<{ slug?: string[] }>;
}

export default async function Page(props: PageProps) {
  const params = await props.params;
  const page = source.getPage(params.slug);

  if (!page) {
    notFound();
  }

  const data = page.data as Record<string, any>;
  const MDX = data.body ?? data.exports?.default;
  const baseUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://mayfly.dev';
  const pageUrl = `${baseUrl}${page.url}`;

  // Rich Snippet JSON-LD for Technical Documentation Articles
  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'TechArticle',
    headline: data.title,
    description: data.description || `Documentation for ${data.title} in MayFly`,
    url: pageUrl,
    inLanguage: 'en-US',
    publisher: {
      '@type': 'Organization',
      name: 'MayFly',
      logo: {
        '@type': 'ImageObject',
        url: `${baseUrl}/icon.png`,
      },
    },
    author: {
      '@type': 'Person',
      name: 'Vishnu Nandan',
      url: 'https://github.com/vishnunandan555',
    },
  };

  const breadcrumbJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Home',
        item: baseUrl,
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Documentation',
        item: `${baseUrl}/docs`,
      },
      ...(params.slug && params.slug.length > 0
        ? [
            {
              '@type': 'ListItem',
              position: 3,
              name: data.title,
              item: pageUrl,
            },
          ]
        : []),
    ],
  };

  return (
    <DocsPage toc={data.toc} full={data.full}>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(articleJsonLd) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(breadcrumbJsonLd) }}
      />
      <DocsTitle className="text-2xl sm:text-3xl font-semibold tracking-tight text-neutral-900 dark:text-neutral-100">
        {data.title}
      </DocsTitle>
      {data.description && (
        <DocsDescription className="text-sm sm:text-base text-neutral-600 dark:text-neutral-400 mt-2">
          {data.description}
        </DocsDescription>
      )}
      <DocsBody className="prose-table:w-full prose-table:overflow-x-auto">
        <MDX
          components={{
            ...defaultMdxComponents,
            Tab,
            Tabs,
            Callout,
            Step,
            Steps,
            Card,
            Cards,
          }}
        />
      </DocsBody>
    </DocsPage>
  );
}

export async function generateStaticParams() {
  return source.generateParams();
}

export async function generateMetadata(props: PageProps): Promise<Metadata> {
  const params = await props.params;
  const page = source.getPage(params.slug);

  if (!page) {
    return {};
  }

  const data = page.data as Record<string, any>;
  const baseUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://mayfly.dev';
  const pageUrl = `${baseUrl}${page.url}`;
  const title = `${data.title} | MayFly Documentation`;
  const description =
    data.description ||
    'MayFly: Zero-Disk Secrets Manager & In-Memory Process Injector for developers.';

  return {
    title: data.title,
    description: description,
    alternates: {
      canonical: pageUrl,
    },
    openGraph: {
      title: title,
      description: description,
      url: pageUrl,
      type: 'article',
      siteName: 'MayFly Documentation',
      images: [
        {
          url: '/icon.png',
          width: 512,
          height: 512,
          alt: `${data.title} - MayFly Documentation`,
        },
      ],
    },
    twitter: {
      card: 'summary_large_image',
      title: title,
      description: description,
      images: ['/icon.png'],
      creator: '@vishnunandan555',
    },
  };
}
