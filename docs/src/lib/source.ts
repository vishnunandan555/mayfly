import { docs } from '../../.source';
import { loader } from 'fumadocs-core/source';
import { createElement } from 'react';
import * as icons from 'lucide-react';

const rawSource = docs.toFumadocsSource() as any;

export const source = loader({
  baseUrl: '/docs',
  source: {
    files: typeof rawSource.files === 'function' ? rawSource.files() : rawSource.files,
  },
  icon(icon) {
    if (!icon) return;
    const IconComponent = (icons as Record<string, any>)[icon];
    if (IconComponent) {
      return createElement(IconComponent, { className: 'size-4' });
    }
  },
});
