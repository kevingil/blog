import { useMemo } from 'react';
import { marked, Token } from 'marked';
import hljs from 'highlight.js';
import { Badge } from '@/components/ui/badge';

// Configure marked with highlight.js -- same setup as the published blog page
marked.use({
  renderer: {
    code(this: unknown, token: Token & { lang?: string; text: string }) {
      const lang = token.lang && hljs.getLanguage(token.lang) ? token.lang : 'plaintext';
      const highlighted = hljs.highlight(token.text, { language: lang }).value;
      return `<pre><code class="hljs language-${lang}">${highlighted}</code></pre>`;
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
    return marked(content) as string;
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
