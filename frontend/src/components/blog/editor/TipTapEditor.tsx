import { useEffect, useRef } from 'react';
import { EditorContent, useEditor } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import {
  Bold,
  Code,
  Heading2,
  Heading3,
  Italic,
  Link2,
  List,
  ListOrdered,
  Quote,
  SquareCode,
  Strikethrough,
} from 'lucide-react';
import { Toggle } from '@/components/ui/toggle';
import { Badge } from '@/components/ui/badge';
import type { TextRange } from '@/lib/added-text';
import { htmlToMarkdown, looksLikeMarkdown, markdownToHtml } from './markdown';
import {
  AddedHighlight,
  ArticleCodeBlock,
  ArticleImage,
  ArticleLink,
  ArticlePlaceholder,
  clearAddedHighlights,
} from './tiptap-extensions';

const EMPTY_RANGES: TextRange[] = [];

const extensions = [
  StarterKit.configure({
    codeBlock: false,
  }),
  ArticleCodeBlock,
  ArticleLink,
  ArticleImage,
  AddedHighlight,
  ArticlePlaceholder,
];

interface TipTapEditorProps {
  content: string;
  onChange: (markdown: string) => void;
  highlights?: TextRange[];
  title?: string;
  authorName?: string;
  imageUrl?: string;
  tags?: string[];
}

export function TipTapEditor({
  content,
  onChange,
  highlights = EMPTY_RANGES,
  title,
  authorName,
  imageUrl,
  tags,
}: TipTapEditorProps) {
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const applyingExternal = useRef(false);
  const lastEmitted = useRef(content);
  const lastAppliedMarkdown = useRef(content);
  const rangesRef = useRef(highlights);
  rangesRef.current = highlights;
  const highlightKey = JSON.stringify(highlights);
  const lastHighlightKey = useRef(highlightKey);
  const editorRef = useRef<ReturnType<typeof useEditor>>(null);

  const editor = useEditor({
    immediatelyRender: true,
    shouldRerenderOnTransaction: true,
    extensions,
    content: markdownToHtml(content, highlights),
    editorProps: {
      attributes: {
        class: 'article-tiptap blog-post prose max-w-none dark:prose-invert min-h-[50vh] focus:outline-none',
        spellcheck: 'true',
      },
      handlePaste(_view, event) {
        const current = editorRef.current;
        if (!current) return false;
        const html = event.clipboardData?.getData('text/html') ?? '';
        const text = event.clipboardData?.getData('text/plain') ?? '';
        if (html.trim() || !looksLikeMarkdown(text)) return false;
        event.preventDefault();
        current.chain().focus().insertContent(markdownToHtml(text)).run();
        return true;
      },
    },
    onUpdate: ({ editor: current, transaction }) => {
      if (applyingExternal.current || !transaction.docChanged) return;
      const markdown = htmlToMarkdown(current.getHTML());
      if (markdown === lastEmitted.current) return;
      lastEmitted.current = markdown;
      onChangeRef.current(markdown);
    },
  }, []);

  editorRef.current = editor;

  useEffect(() => {
    if (!editor || editor.isDestroyed) return;
    const ranges = rangesRef.current;
    const highlightsChanged = highlightKey !== lastHighlightKey.current;
    const contentFromEditor = content === lastEmitted.current;

    if (!highlightsChanged && contentFromEditor) {
      lastAppliedMarkdown.current = content;
      return;
    }

    if (highlightsChanged && ranges.length === 0 && contentFromEditor) {
      lastHighlightKey.current = highlightKey;
      lastAppliedMarkdown.current = content;
      applyingExternal.current = true;
      try {
        clearAddedHighlights(editor);
      } finally {
        applyingExternal.current = false;
      }
      return;
    }

    if (!highlightsChanged && content === lastAppliedMarkdown.current) return;

    lastAppliedMarkdown.current = content;
    lastHighlightKey.current = highlightKey;
    lastEmitted.current = content;
    applyingExternal.current = true;
    try {
      editor.commands.setContent(markdownToHtml(content, ranges), false);
    } finally {
      applyingExternal.current = false;
    }
  }, [editor, content, highlightKey]);

  return (
    <div className="article-editor flex h-full min-h-0 flex-col bg-background">
      <div className="flex shrink-0 flex-wrap items-center gap-0.5 border-b px-2 py-1">
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('bold')} onPressedChange={() => editor?.chain().focus().toggleBold().run()} aria-label="Bold" title="Bold">
          <Bold className="h-3.5 w-3.5" />
        </Toggle>
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('italic')} onPressedChange={() => editor?.chain().focus().toggleItalic().run()} aria-label="Italic" title="Italic">
          <Italic className="h-3.5 w-3.5" />
        </Toggle>
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('strike')} onPressedChange={() => editor?.chain().focus().toggleStrike().run()} aria-label="Strikethrough" title="Strikethrough">
          <Strikethrough className="h-3.5 w-3.5" />
        </Toggle>
        <div className="mx-1 h-4 w-px bg-border" />
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('heading', { level: 2 })} onPressedChange={() => editor?.chain().focus().toggleHeading({ level: 2 }).run()} aria-label="Heading 2" title="Heading 2">
          <Heading2 className="h-3.5 w-3.5" />
        </Toggle>
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('heading', { level: 3 })} onPressedChange={() => editor?.chain().focus().toggleHeading({ level: 3 }).run()} aria-label="Heading 3" title="Heading 3">
          <Heading3 className="h-3.5 w-3.5" />
        </Toggle>
        <div className="mx-1 h-4 w-px bg-border" />
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('bulletList')} onPressedChange={() => editor?.chain().focus().toggleBulletList().run()} aria-label="Bullet list" title="Bullet list">
          <List className="h-3.5 w-3.5" />
        </Toggle>
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('orderedList')} onPressedChange={() => editor?.chain().focus().toggleOrderedList().run()} aria-label="Numbered list" title="Numbered list">
          <ListOrdered className="h-3.5 w-3.5" />
        </Toggle>
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('blockquote')} onPressedChange={() => editor?.chain().focus().toggleBlockquote().run()} aria-label="Quote" title="Quote">
          <Quote className="h-3.5 w-3.5" />
        </Toggle>
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('code')} onPressedChange={() => editor?.chain().focus().toggleCode().run()} aria-label="Inline code" title="Inline code">
          <Code className="h-3.5 w-3.5" />
        </Toggle>
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('codeBlock')} onPressedChange={() => editor?.chain().focus().toggleCodeBlock().run()} aria-label="Code block" title="Code block">
          <SquareCode className="h-3.5 w-3.5" />
        </Toggle>
        <div className="mx-1 h-4 w-px bg-border" />
        <Toggle
          size="sm"
          type="button"
          pressed={!!editor?.isActive('link')}
          onPressedChange={() => {
            if (!editor) return;
            const previous = editor.getAttributes('link').href as string | undefined;
            const href = window.prompt('Link URL', previous || 'https://');
            if (href == null) return;
            if (!href.trim()) {
              editor.chain().focus().unsetMark('link').run();
              return;
            }
            editor.chain().focus().setMark('link', { href: href.trim() }).run();
          }}
          aria-label="Link"
          title="Link"
        >
          <Link2 className="h-3.5 w-3.5" />
        </Toggle>
      </div>
      <div className="min-h-0 flex-1 overflow-auto">
        <article className="mx-auto max-w-4xl px-8 pb-8 pt-6">
          {title && <h1 className="mb-4 text-4xl font-bold">{title}</h1>}
          {imageUrl && (
            <img
              src={imageUrl}
              alt={title || 'Article image'}
              className="mb-6 aspect-video w-full rounded-2xl object-cover"
            />
          )}
          {authorName && (
            <div className="mb-6 flex items-center">
              <p className="font-semibold">{authorName}</p>
            </div>
          )}
          <EditorContent editor={editor} />
          {tags && tags.length > 0 && (
            <div className="mb-8 mt-8 flex flex-wrap gap-2">
              {tags.map((tag) => (
                <Badge key={tag} variant="secondary" className="border-1 border-solid border-indigo-500 text-primary">
                  {tag}
                </Badge>
              ))}
            </div>
          )}
        </article>
      </div>
    </div>
  );
}
