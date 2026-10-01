import { Marked } from 'marked';
import type { TextRange } from '@/lib/added-text';
import { turndownService } from './turndown';

type MarkdownToken = {
  type: string;
  raw?: string;
  text?: string;
  lang?: string;
  depth?: number;
  ordered?: boolean;
  start?: number | string;
  tokens?: MarkdownToken[];
  items?: MarkdownToken[];
  _from?: number;
  _to?: number;
};

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

function tokenOverlaps(token: MarkdownToken, ranges: TextRange[]): boolean {
  if (token._from == null || token._to == null || ranges.length === 0) return false;
  const raw = token.raw ?? '';
  for (const range of ranges) {
    const start = Math.max(token._from, range.from);
    const end = Math.min(token._to, range.to);
    if (start >= end) continue;
    const slice = raw.slice(start - token._from, end - token._from);
    if (slice.trim().length > 0) return true;
  }
  return false;
}

function assignNested(token: MarkdownToken) {
  if (token.type === 'list') {
    assignListItems(token);
    return;
  }
  if (token.type !== 'blockquote' || !token.tokens) return;
  const raw = token.raw ?? '';
  const base = token._from ?? 0;
  let search = 0;
  for (const child of token.tokens) {
    const childRaw = child.raw ?? '';
    if (!childRaw) continue;
    const index = raw.indexOf(childRaw, search);
    if (index < 0) continue;
    child._from = base + index;
    child._to = child._from + childRaw.length;
    search = index + childRaw.length;
    assignNested(child);
  }
}

function assignListItems(token: MarkdownToken) {
  const raw = token.raw ?? '';
  const base = token._from ?? 0;
  let search = 0;
  for (const item of token.items ?? []) {
    const itemRaw = item.raw ?? '';
    const index = itemRaw ? raw.indexOf(itemRaw, search) : -1;
    if (index >= 0) {
      item._from = base + index;
      item._to = item._from + itemRaw.length;
      search = index + itemRaw.length;
    }
    for (const child of item.tokens ?? []) {
      if ((child.type === 'list' || child.type === 'blockquote') && item._from != null && child.raw) {
        const childIndex = itemRaw.indexOf(child.raw);
        if (childIndex >= 0) {
          child._from = item._from + childIndex;
          child._to = child._from + child.raw.length;
          assignNested(child);
        }
      }
    }
  }
}

function assignOffsets(tokens: MarkdownToken[]) {
  let cursor = 0;
  for (const token of tokens) {
    const raw = token.raw ?? '';
    token._from = cursor;
    token._to = cursor + raw.length;
    cursor += raw.length;
    assignNested(token);
  }
}

function wrapPlain(text: string): string {
  if (!text.trim()) return text;
  return `<mark data-added="true">${text}</mark>`;
}

function wrapAddedText(html: string): string {
  if (!html.includes('<')) return wrapPlain(html);
  const wrapped = html.replace(/>([^<]+)</g, (_match, text: string) => `>${wrapPlain(text)}<`);
  return wrapped.replace(/^([^<]+)/, wrapPlain).replace(/>([^<]+)$/, (_match, text: string) => `>${wrapPlain(text)}`);
}

function markImages(html: string): string {
  return html.replace(/<img\b([^>]*?)>/g, (match, attrs: string) => {
    if (/\sdata-added=/.test(attrs)) return match;
    return `<img${attrs} data-added="true">`;
  });
}

function createRenderer(ranges: TextRange[]) {
  return {
    heading(this: { parser: { parseInline: (tokens: MarkdownToken[]) => string } }, token: MarkdownToken) {
      const inner = this.parser.parseInline(token.tokens ?? []);
      const body = tokenOverlaps(token, ranges) ? wrapAddedText(inner) : inner;
      return `<h${token.depth}>${body}</h${token.depth}>\n`;
    },
    paragraph(this: { parser: { parseInline: (tokens: MarkdownToken[]) => string } }, token: MarkdownToken) {
      const inner = this.parser.parseInline(token.tokens ?? []);
      if (!tokenOverlaps(token, ranges)) return `<p>${inner}</p>\n`;
      return `<p>${markImages(wrapAddedText(inner))}</p>\n`;
    },
    code(token: MarkdownToken) {
      const lang = String(token.lang || '').replace(/[^\w#+-]/g, '');
      const escaped = escapeHtml(token.text ?? '');
      const added = tokenOverlaps(token, ranges);
      const body = added ? `<mark data-added="true">${escaped}</mark>` : escaped;
      const preAttrs = added ? ' data-added="true"' : '';
      const classAttr = lang ? ` class="language-${lang}"` : '';
      return `<pre${preAttrs}><code${classAttr}>${body}</code></pre>\n`;
    },
    list(this: { listitem: (token: MarkdownToken) => string }, token: MarkdownToken) {
      const tag = token.ordered ? 'ol' : 'ul';
      const start = token.ordered && token.start && token.start !== 1 ? ` start="${token.start}"` : '';
      const body = (token.items ?? []).map((item) => this.listitem(item)).join('');
      return `<${tag}${start}>\n${body}</${tag}>\n`;
    },
    listitem(this: { parser: { parse: (tokens: MarkdownToken[]) => string } }, token: MarkdownToken) {
      const parsed = this.parser.parse(token.tokens ?? []).trim();
      const block = /^<(p|ul|ol|pre|blockquote|h[1-6]|div)\b/i.test(parsed) ? parsed : `<p>${parsed}</p>`;
      const inner = tokenOverlaps(token, ranges) ? wrapAddedText(block) : block;
      return `<li>${inner}</li>\n`;
    },
    blockquote(this: { parser: { parse: (tokens: MarkdownToken[]) => string } }, token: MarkdownToken) {
      const body = this.parser.parse(token.tokens ?? []);
      const located = (token.tokens ?? []).some((child) => child.type !== 'space' && child._from != null);
      if (!located && tokenOverlaps(token, ranges)) {
        return `<blockquote>\n${wrapAddedText(body)}</blockquote>\n`;
      }
      return `<blockquote>\n${body}</blockquote>\n`;
    },
  };
}

export function markdownToHtml(markdown: string, ranges: TextRange[] = []): string {
  if (!markdown) return '';
  const marked = new Marked({ renderer: createRenderer(ranges) as never });
  const tokens = marked.lexer(markdown) as MarkdownToken[];
  assignOffsets(tokens);
  return marked.parser(tokens as never);
}

export function htmlToMarkdown(html: string): string {
  if (!html || !html.trim()) return '';
  return turndownService.turndown(html).trim();
}

export function looksLikeMarkdown(text: string): boolean {
  if (!text.trim()) return false;
  return /(^|\n)(#{1,6} |> ?|```|\s*[-*+] |\s*\d+\. )/m.test(text)
    || /!\[[^\]]*\]\([^)]+\)/.test(text)
    || /\[[^\]]+\]\([^)]+\)/.test(text)
    || /(\*\*|__).+?\1/.test(text);
}
