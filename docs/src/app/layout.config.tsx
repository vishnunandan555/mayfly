import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';
import React from 'react';

export const baseOptions: BaseLayoutProps = {
  nav: {
    title: (
      <div className="flex items-center gap-2.5 font-semibold tracking-tight text-neutral-100">
        <span className="inline-flex size-5 items-center justify-center rounded-full bg-amber-500/10 ring-1 ring-inset ring-amber-500/30">
          <span className="size-2 rounded-full bg-amber-400 shadow-[0_0_6px_rgba(245,158,11,0.6)]" />
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
