import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';
import React from 'react';

export const baseOptions: BaseLayoutProps = {
  nav: {
    title: (
      <div className="flex items-center gap-2 font-semibold tracking-tight text-neutral-100">
        <span className="inline-flex size-6 items-center justify-center rounded-md bg-emerald-500/10 ring-1 ring-inset ring-emerald-500/20">
          <span className="size-2 rounded-full bg-emerald-400" />
        </span>
        <span className="font-mono text-sm font-bold tracking-wider">MayFly</span>
      </div>
    ),
  },
  links: [
    {
      text: 'Home',
      url: '/',
    },
    {
      text: 'Docs',
      url: '/docs',
      active: 'nested-url',
    },
    {
      text: 'GitHub',
      url: 'https://github.com/vishnunandan555/mayfly',
      external: true,
    },
  ],
};
