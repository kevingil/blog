import { useEffect, useRef, useState, type ReactNode } from 'react';
import { EditorContent, useEditor } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import {
  Bold,
  Code,
  Heading2,
  Heading3,
  ImageIcon,
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
import { BlurhashImage } from '@/components/media/BlurhashImage';
import type { TextRange } from '@/lib/added-text';
import { htmlToMarkdown, looksLikeMarkdown, markdownToHtml } from './markdown';
import { ArticleSources, type ArticleCitation } from '../ArticleSources';
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
  onTitleChange?: (title: string) => void;
  titleError?: string;
  authorName?: string;
  imageUrl?: string;
  imageBlurhash?: string | null;
  onEditImage?: () => void;
  tags?: string[];
  meta?: ReactNode;
  sideControls?: ReactNode;
  mobileControls?: ReactNode;
  onDropCover?: (file: File) => Promise<void> | void;
  onUploadBodyImage?: (file: File) => Promise<{ url: string; alt: string }>;
  sources?: ArticleCitation[];
}

function imageFiles(list: FileList | null | undefined): File[] {
  if (!list || list.length === 0) return [];
  return Array.from(list).filter((file) => (
    file.type.startsWith('image/') || /\.(png|jpe?g|gif|webp|avif|heic|heif|svg)$/i.test(file.name)
  ));
}

function dragHasFiles(event: { dataTransfer: DataTransfer | null }): boolean {
  return Array.from(event.dataTransfer?.types ?? []).includes('Files');
}

export function TipTapEditor({
  content,
  onChange,
  highlights = EMPTY_RANGES,
  title,
  onTitleChange,
  titleError,
  authorName,
  imageUrl,
  imageBlurhash,
  onEditImage,
  tags,
  meta,
  sideControls,
  mobileControls,
  onDropCover,
  onUploadBodyImage,
  sources = [],
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
  const [coverDrag, setCoverDrag] = useState(false);
  const [bodyDrag, setBodyDrag] = useState(false);
  const bodyDragDepth = useRef(0);

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
      handleDrop(_view, event) {
        if (imageFiles(event.dataTransfer?.files).length === 0) return false;
        event.preventDefault();
        return true;
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

  const formatToolbar = (
    <div className="flex shrink-0 items-center gap-0 border-t border-border/50 px-0.5 py-0.5 md:px-1 md:py-1" aria-label="Text formatting">
      <div className="flex min-w-0 flex-1 items-center gap-0 overflow-x-auto [scrollbar-width:none] md:gap-0.5 [&::-webkit-scrollbar]:hidden [&_button]:shrink-0 [&_button]:max-md:h-7 [&_button]:max-md:min-w-7 [&_button]:max-md:px-1">
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('bold')} onPressedChange={() => editor?.chain().focus().toggleBold().run()} aria-label="Bold" title="Bold">
          <Bold className="h-3.5 w-3.5" />
        </Toggle>
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('italic')} onPressedChange={() => editor?.chain().focus().toggleItalic().run()} aria-label="Italic" title="Italic">
          <Italic className="h-3.5 w-3.5" />
        </Toggle>
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('strike')} onPressedChange={() => editor?.chain().focus().toggleStrike().run()} aria-label="Strikethrough" title="Strikethrough">
          <Strikethrough className="h-3.5 w-3.5" />
        </Toggle>
        <div className="mx-0.5 h-4 w-px shrink-0 bg-border md:mx-1" />
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('heading', { level: 2 })} onPressedChange={() => editor?.chain().focus().toggleHeading({ level: 2 }).run()} aria-label="Heading 2" title="Heading 2">
          <Heading2 className="h-3.5 w-3.5" />
        </Toggle>
        <Toggle size="sm" type="button" pressed={!!editor?.isActive('heading', { level: 3 })} onPressedChange={() => editor?.chain().focus().toggleHeading({ level: 3 }).run()} aria-label="Heading 3" title="Heading 3">
          <Heading3 className="h-3.5 w-3.5" />
        </Toggle>
        <div className="mx-0.5 h-4 w-px shrink-0 bg-border md:mx-1" />
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
        <div className="mx-0.5 h-4 w-px shrink-0 bg-border md:mx-1" />
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
        {mobileControls ? (
          <div className="flex shrink-0 items-center gap-1 pr-1 md:hidden">
            {mobileControls}
          </div>
        ) : null}
      </div>
  );

  return (
    <div className="article-editor relative flex h-full min-h-0 flex-col bg-background">
      {sideControls}
      <div className="min-h-0 flex-1 overflow-auto">
        <article className="article-editor-document">
          <input
            value={title ?? ''}
            onChange={(event) => onTitleChange?.(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') event.preventDefault();
            }}
            placeholder="Article Title"
            aria-label="Article title"
            className="article-editor-title"
          />
          {titleError ? <p className="mb-2 text-sm text-red-500">{titleError}</p> : null}
          {meta}
          {onEditImage ? (
            <button
              type="button"
              onClick={onEditImage}
              onDragEnter={(event) => {
                if (!dragHasFiles(event)) return;
                event.preventDefault();
                event.stopPropagation();
                setCoverDrag(true);
              }}
              onDragOver={(event) => {
                if (!dragHasFiles(event)) return;
                event.preventDefault();
                event.stopPropagation();
                setCoverDrag(true);
              }}
              onDragLeave={(event) => {
                event.stopPropagation();
                setCoverDrag(false);
              }}
              onDrop={(event) => {
                event.preventDefault();
                event.stopPropagation();
                setCoverDrag(false);
                bodyDragDepth.current = 0;
                setBodyDrag(false);
                const file = imageFiles(event.dataTransfer.files)[0];
                if (file && onDropCover) void onDropCover(file);
              }}
              className={[
                imageUrl ? 'article-cover-preview' : 'article-cover-preview article-cover-preview-empty',
                coverDrag ? 'article-cover-preview-dropping' : '',
              ].filter(Boolean).join(' ')}
              aria-label="Edit header image"
            >
              {imageUrl && !coverDrag ? (
                <BlurhashImage
                  src={imageUrl}
                  alt={title || 'Article image'}
                  blurhash={imageBlurhash}
                  className="absolute inset-0"
                  imgClassName="h-full w-full object-cover"
                />
              ) : (
                <span className="relative z-[1] flex flex-col items-center gap-2 text-sm text-muted-foreground">
                  <ImageIcon className="h-5 w-5" />
                  {coverDrag ? 'Drop image to upload' : 'Drop an image, or click to choose one'}
                </span>
              )}
            </button>
          ) : null}
          {authorName && (
            <div className="mb-6 flex items-center">
              <p className="font-semibold">{authorName}</p>
            </div>
          )}
          <div
            className="article-body-drop"
            onDragEnter={(event) => {
              if (!dragHasFiles(event)) return;
              event.preventDefault();
              bodyDragDepth.current += 1;
              setBodyDrag(true);
            }}
            onDragOver={(event) => {
              if (!dragHasFiles(event)) return;
              event.preventDefault();
            }}
            onDragLeave={() => {
              bodyDragDepth.current = Math.max(0, bodyDragDepth.current - 1);
              if (bodyDragDepth.current === 0) setBodyDrag(false);
            }}
            onDrop={(event) => {
              bodyDragDepth.current = 0;
              setBodyDrag(false);
              const files = imageFiles(event.dataTransfer.files);
              if (files.length === 0 || !onUploadBodyImage || !editor) return;
              event.preventDefault();
              const coords = editor.view.posAtCoords({ left: event.clientX, top: event.clientY });
              const position = coords?.pos ?? editor.state.selection.from;
              const upload = onUploadBodyImage;
              void (async () => {
                try {
                  const uploaded = [];
                  for (const file of files) {
                    uploaded.push(await upload(file));
                  }
                  const html = markdownToHtml(uploaded.map((file) => `![${file.alt}](${file.url})`).join('\n\n'));
                  editor.chain().focus().insertContentAt(position, html).run();
                } catch {
                  // The upload callback reports the failure.
                }
              })();
            }}
          >
            {bodyDrag && !coverDrag ? (
              <div className="article-body-drop-overlay">Drop image to insert</div>
            ) : null}
            <EditorContent editor={editor} />
          </div>
          <ArticleSources sources={sources} />
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
      {formatToolbar}
    </div>
  );
}
