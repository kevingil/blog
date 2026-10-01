import TurndownService from 'turndown';
// @ts-expect-error -- no types available for this package
import { gfm } from 'turndown-plugin-gfm';

// Module-level singleton: properly configured Turndown instance.
const turndownService = new TurndownService({
  headingStyle: 'atx',
  codeBlockStyle: 'fenced',
  emDelimiter: '*',
  bulletListMarker: '-',
});

// Enable GFM extensions (tables, strikethrough, task lists)
turndownService.use(gfm);

// Custom rule: TipTap's <pre><code> blocks must produce proper fenced code blocks.
// Without this, Turndown escapes backticks/special chars and flattens code to one line.
turndownService.addRule('fencedCodeBlock', {
  filter: function (node: HTMLElement) {
    return node.nodeName === 'PRE';
  },
  replacement: function (_content: string, node: Node) {
    const el = node as HTMLElement;
    const codeEl = el.querySelector('code');
    const text = ((codeEl || el).textContent || '').replace(/\n$/, '');
    const langClass = codeEl?.getAttribute('class') || '';
    const langMatch = langClass.match(/language-(\S+)/);
    const lang = langMatch ? langMatch[1] : '';
    return '\n\n```' + lang + '\n' + text + '\n```\n\n';
  },
});

// Highlight marks are editor-only. Saved markdown stays plain text.
turndownService.addRule('addedHighlight', {
  filter: function (node: HTMLElement) {
    return node.nodeName === 'MARK';
  },
  replacement: function (content: string) {
    return content;
  },
});

// TipTap stores each item as a paragraph. Flatten that so the markdown stays a tight list.
turndownService.addRule('tightListItem', {
  filter: 'li',
  replacement: function (content: string, node: Node) {
    const parent = node.parentNode as HTMLElement | null;
    const text = content
      .replace(/^\n+/, '')
      .replace(/\n+$/, '')
      .replace(/\n/g, '\n    ');
    let prefix = '- ';
    if (parent?.nodeName === 'OL') {
      const items = Array.from(parent.children).filter((child) => child.nodeName === 'LI');
      const index = items.indexOf(node as HTMLElement);
      const start = Number(parent.getAttribute('start') || '1');
      prefix = `${start + Math.max(index, 0)}. `;
    }
    return `${prefix}${text}\n`;
  },
});

export { turndownService };
