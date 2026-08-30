import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';
import React from 'react';

export const baseOptions: BaseLayoutProps = {
  nav: {
    title: (
      <div className="flex items-center gap-2 font-semibold tracking-tight text-neutral-900 dark:text-neutral-100">
        <span className="font-mono text-base font-bold tracking-wider">MAYFLY</span>
        <span className="rounded border border-neutral-300 dark:border-neutral-700 bg-neutral-100 dark:bg-neutral-800 px-1.5 py-0.5 text-[10px] font-mono text-neutral-700 dark:text-neutral-300">
          ZERO-DEP
        </span>
      </div>
    ),
  },
  links: [
    {
      text: 'Documentation',
      url: '/docs',
      active: 'nested-url',
    },
    {
      text: 'Security Model',
      url: '/docs/architecture/security-model',
    },
    {
      text: 'CLI Reference',
      url: '/docs/cli/overview',
    },
    {
      text: 'GitHub',
      url: 'https://github.com/vishnunandan555/mayfly',
      external: true,
    },
  ],
};
