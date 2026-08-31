import { docs } from '../../.source';
import { loader } from 'fumadocs-core/source';
import { createElement } from 'react';
import * as icons from 'lucide-react';
import { STACK_ICONS } from './stack-icons';

const rawSource = docs.toFumadocsSource() as any;

export const source = loader({
  baseUrl: '/docs',
  source: {
    files: typeof rawSource.files === 'function' ? rawSource.files() : rawSource.files,
  },
  icon(icon) {
    if (!icon) return;
    const StackComponent = STACK_ICONS[icon];
    if (StackComponent) {
      return createElement(StackComponent, { className: 'size-4' });
    }
    const IconComponent = (icons as Record<string, any>)[icon];
    if (IconComponent) {
      return createElement(IconComponent, { className: 'size-4' });
    }
  },
});
