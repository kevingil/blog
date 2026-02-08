import TurndownService from 'turndown';

// #region agent log
console.log('[TURNDOWN] Module loaded at', new Date().toISOString());
// #endregion

// Module-level singleton: properly configured Turndown instance.
const turndownService = new TurndownService({
  headingStyle: 'atx',
  codeBlockStyle: 'fenced',
  emDelimiter: '*',
  bulletListMarker: '-',
});

// Custom rule: TipTap's <pre><code> blocks must produce proper fenced code blocks.
// Without this, Turndown escapes backticks/special chars and flattens code to one line.
turndownService.addRule('fencedCodeBlock', {
  filter: function (node: HTMLElement) {
    return node.nodeName === 'PRE';
  },
  replacement: function (_content: string, node: Node) {
    const el = node as HTMLElement;
    const codeEl = el.querySelector('code');
    const text = (codeEl || el).textContent || '';
    const langClass = codeEl?.getAttribute('class') || '';
    const langMatch = langClass.match(/language-(\S+)/);
    const lang = langMatch ? langMatch[1] : '';
    return '\n\n```' + lang + '\n' + text + '\n```\n\n';
  },
});

export { turndownService };
