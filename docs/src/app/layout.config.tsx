import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';
import React from 'react';

export const baseOptions: BaseLayoutProps = {
  nav: {
    title: (
      <div className="flex items-center gap-2.5 font-semibold tracking-tight text-neutral-900 dark:text-neutral-100">
        <img
          src="/icon.png"
          alt="MayFly Logo"
          width={22}
          height={22}
          className="size-5.5 object-contain shrink-0"
        />
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
