import { useMemo } from 'react';
import { Marked } from 'marked';
import hljs from 'highlight.js';
import { Badge } from '@/components/ui/badge';

/**
 * Walk top-level block tokens and stamp each with its 1-based source line
 * number so the renderer can emit `data-source-line` attributes.
 */
function annotateSourceLines(tokens: any[]): void {
  let lineNum = 1;
  for (const token of tokens) {
    if (token.type !== 'space') {
      token._sourceLine = lineNum;
    }
    const raw: string = token.raw ?? '';
    for (let i = 0; i < raw.length; i++) {
      if (raw[i] === '\n') lineNum++;
    }
  }
}

/** Helper – returns ` data-source-line="N"` or empty string */
function lineAttr(token: any): string {
  return token._sourceLine != null ? ` data-source-line="${token._sourceLine}"` : '';
}

// Marked instance with source-line annotations + syntax highlighting
const previewMarked = new Marked({
  renderer: {
    heading(this: any, token: any) {
      return `<h${token.depth}${lineAttr(token)}>${this.parser.parseInline(token.tokens)}</h${token.depth}>\n`;
    },
    paragraph(this: any, token: any) {
      return `<p${lineAttr(token)}>${this.parser.parseInline(token.tokens)}</p>\n`;
    },
    code(_token: any) {
      const token = _token as any;
      const lang = token.lang && hljs.getLanguage(token.lang) ? token.lang : 'plaintext';
      const highlighted = hljs.highlight(token.text, { language: lang }).value;
      return `<pre${lineAttr(token)}><code class="hljs language-${lang}">${highlighted}</code></pre>\n`;
    },
    list(this: any, token: any) {
      const tag = token.ordered ? 'ol' : 'ul';
      const startAttr = token.ordered && token.start !== 1 ? ` start="${token.start}"` : '';
      const body = token.items.map((item: any) => this.listitem(item)).join('');
      return `<${tag}${startAttr}${lineAttr(token)}>\n${body}</${tag}>\n`;
    },
    blockquote(this: any, token: any) {
      const body = this.parser.parse(token.tokens);
      return `<blockquote${lineAttr(token)}>\n${body}</blockquote>\n`;
    },
    hr(token: any) {
      return `<hr${lineAttr(token)}>\n`;
    },
  },
});

interface MarkdownPreviewProps {
  content: string;
  title?: string;
  authorName?: string;
  imageUrl?: string;
  tags?: string[];
}

export function MarkdownPreview({ content, title, authorName, imageUrl, tags }: MarkdownPreviewProps) {
  const renderedHtml = useMemo(() => {
    if (!content) return '';
    const tokens = previewMarked.lexer(content);
    annotateSourceLines(tokens);
    return previewMarked.parser(tokens);
  }, [content]);

  return (
    <div className="h-full overflow-auto bg-background">
      <article className="max-w-4xl mx-auto p-8">
        {title && <h1 className="text-4xl font-bold mb-4">{title}</h1>}
        {imageUrl && (
          <img
            src={imageUrl}
            alt={title || 'Article image'}
            className="rounded-2xl mb-6 object-cover aspect-video w-full"
          />
        )}
        {authorName && (
          <div className="flex items-center mb-6">
            <p className="font-semibold">{authorName}</p>
          </div>
        )}
        <div
          className="blog-post prose max-w-none dark:prose-invert mb-8"
          dangerouslySetInnerHTML={{ __html: renderedHtml }}
        />
        {tags && tags.length > 0 && (
          <div className="flex flex-wrap gap-2 mb-8">
            {tags.map((tag) => (
              <Badge key={tag} variant="secondary" className="text-primary border-solid border-1 border-indigo-500">
                {tag}
              </Badge>
            ))}
          </div>
        )}
      </article>
    </div>
  );
}
