import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from '@tanstack/react-router';
import { useAuth } from '@/services/auth/auth';
import { useForm, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { format } from "date-fns"
import { Calendar as CalendarIcon, PencilIcon, SparklesIcon, RefreshCw, Bold, Italic, Strikethrough, Code, Heading1, Heading2, Heading3, List, ListOrdered, Quote, Undo, Redo, ArrowUp, Square } from "lucide-react"
import { ExternalLinkIcon, UploadIcon } from '@radix-ui/react-icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useEditor, EditorContent } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import CodeBlock from '@tiptap/extension-code-block';
import { Extension } from '@tiptap/core';
import { Plugin, PluginKey } from 'prosemirror-state';
import { Decoration, DecorationSet } from 'prosemirror-view';
import MarkdownIt from 'markdown-it';
import { diffWords } from 'diff';
import type { Editor as TiptapEditor } from '@tiptap/core';
import { VITE_API_BASE_URL } from "@/services/constants";
import { apiPost, isAuthError } from '@/services/authenticatedFetch';
import '@/tiptap.css';

import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import {
  PromptInput,
  PromptInputTextarea,
  PromptInputActions,
  PromptInputAction,
} from "@/components/ui/prompt-input";
import { ChipInput } from "@/components/ui/chip-input";
import { Calendar } from "@/components/ui/calendar"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { useToast } from "@/hooks/use-toast";
import { ThinkShimmerBlock } from "@/components/ui/think-shimmer";
import { Markdown } from "@/components/ui/markdown";
import { getConversationHistory } from "@/services/conversations";
// Artifact accept/reject is now handled by the sticky DiffActionBar
import { WebSearchSteps, WebSearchToolContext } from "./WebSearchSteps";
import { DiffActionBar } from "./DiffActionBar";
import { DiffArtifact } from "./DiffArtifact";
import { ToolGroupDisplay } from "./ToolGroupDisplay";
import type { 
  ToolGroup, 
  ThinkingBlock, 
  Artifact as NewArtifact,
} from "./types";
import { 
  ToolCall, 
  ToolCallTrigger, 
  ToolCallContent, 
  ToolCallStatusItem 
} from "@/components/prompt-kit/tool-call";
import { 
  Steps, 
  StepsTrigger, 
  StepsContent, 
  StepsItem 
} from "@/components/prompt-kit/steps";
import { cn } from '@/lib/utils';
import { FileDiff, Wrench, BookOpen, FileSearch, PlusCircle, FileText, ImageIcon } from "lucide-react";
import { 
  updateArticle, 
  getArticle, 
  createArticle,
  generateArticleImage,
  getImageGeneration,
  getImageGenerationStatus,
  updateArticleWithContext
} from '@/services/blog';
import { Link } from '@tanstack/react-router';
import { ArticleListItem } from '@/services/types';
import { Switch } from '@/components/ui/switch';
import { Dialog, DialogTitle, DialogContent, DialogTrigger, DialogDescription, DialogFooter, DialogHeader, DialogClose } from '@/components/ui/dialog';
import { Drawer, DrawerTrigger, DrawerContent, DrawerHeader, DrawerTitle, DrawerDescription, DrawerFooter, DrawerClose } from '@/components/ui/drawer';
import { SourcesManager } from './SourcesManager';
import { SourcesPreview } from './SourcesPreview';

// Helper function to convert tool names to user-friendly display names
function getToolDisplayName(toolName: string): string {
  const toolDisplayMap: Record<string, string> = {
    'rewrite_document': 'Rewriting document',
    'edit_text': 'Editing text',
    'analyze_document': 'Analyzing document',
    'generate_image_prompt': 'Generating image prompt',
    'search_web': 'Searching the web',
    'fetch_url': 'Fetching content',
    'search_documents': 'Searching documents',
    'read_file': 'Reading file',
    'write_file': 'Writing file',
    'calculate': 'Calculating',
    'translate': 'Translating',
    'research': 'Researching'
  };
  
  if (toolDisplayMap[toolName]) {
    return toolDisplayMap[toolName];
  }
  
  const friendlyName = toolName
    .replace(/_/g, ' ')
    .replace(/([A-Z])/g, ' $1')
    .toLowerCase()
    .replace(/^./, str => str.toUpperCase())
    .trim();
  
  if (toolName.toLowerCase().includes('search')) {
    return `Searching for ${friendlyName.toLowerCase()}`;
  } else if (toolName.toLowerCase().includes('generate')) {
    return `Generating ${friendlyName.toLowerCase()}`;
  } else if (toolName.toLowerCase().includes('fetch') || toolName.toLowerCase().includes('get')) {
    return `Fetching ${friendlyName.toLowerCase()}`;
  } else if (toolName.toLowerCase().includes('analyze')) {
    return `Analyzing ${friendlyName.toLowerCase()}`;
  } else {
    return `Using ${friendlyName.toLowerCase()}`;
  }
}

const DEFAULT_IMAGE_PROMPT = [
  "A modern, minimalist illustration",
  "A vibrant, colorful scene",
  "A professional business setting",
  "A natural landscape",
  "An abstract design"
];

const articleSchema = z.object({
  title: z.string().min(1, 'Title is required'),
  content: z.string().min(1, 'Content is required'),
  image_url: z.union([z.string().url(), z.literal('')]).optional(),
  tags: z.array(z.string()),
  isDraft: z.boolean(),
});

type ArticleFormData = z.infer<typeof articleSchema>;

type SearchResult = {
  title: string;
  url: string;
  summary?: string;
  author?: string;
  published_date?: string;
  favicon?: string;
  highlights?: string[];
  text_preview?: string;
  has_full_text?: boolean;
};

type SourceInfo = {
  source_id: string;
  original_title: string;
  original_url: string;
  content_length: number;
  source_type?: string;
  search_query?: string;
};

type ChatMessage = {
  id?: string;
  role: 'user' | 'assistant' | 'tool';
  content: string;
  diffState?: 'accepted' | 'rejected';
  diffPreview?: {
    oldText: string;
    newText: string;
    reason?: string;
  };
  meta_data?: {
    artifact?: {
      id: string;
      type: string;
      status: string;
      content: string;
      diff_preview?: string;
      title?: string;
      description?: string;
      applied_at?: string;
    };
    task_status?: any;
    tool_execution?: any;
    tool_group?: ToolGroup;
    thinking?: ThinkingBlock;
    artifacts?: NewArtifact[];
    context?: any;
    user_action?: any;
  };
  // New: Tool group for unified tool call display
  tool_group?: ToolGroup;
  // New: Thinking/chain of thought
  thinking?: ThinkingBlock;
  // Legacy tool_context for backward compatibility
  tool_context?: {
    tool_name: string;
    tool_id: string;
    status: 'starting' | 'running' | 'completed' | 'error';
    // Web search specific
    search_query?: string;
    search_results?: SearchResult[];
    sources_created?: SourceInfo[];
    total_found?: number;
    sources_successful?: number;
    // Ask question specific
    answer?: string;
    citations?: Array<{
      url: string;
      title: string;
      author?: string;
      published_date?: string;
    }>;
    // Generic tool fields
    message?: string;
    input?: Record<string, unknown>;
    output?: Record<string, unknown>;
    reason?: string;
  };
  created_at?: string;
};

// === TipTap Diff Extension ================================================
const DIFF_PLUGIN_KEY = new PluginKey('diff-highlighter');
const DiffHighlighter = Extension.create({
  name: 'diffHighlighter',
  addStorage() {
    return {
      active: false as boolean,
      parts: [] as Array<{ added?: boolean; removed?: boolean; value: string }>,
    };
  },
  addCommands() {
    return {
      showDiff:
        (oldText: string, newText: string) => ({ tr, dispatch }: { tr: unknown; dispatch: (tr: unknown) => void }) => {
          try {
            const parts = diffWords(oldText, newText);
            // @ts-ignore
            this.storage.active = true;
            // @ts-ignore
            this.storage.parts = parts;
            // @ts-ignore - set meta to force plugin to recompute decorations
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (tr as any).setMeta(DIFF_PLUGIN_KEY, { updatedAt: Date.now(), active: true });
            if (dispatch) dispatch(tr);
            return true;
          } catch (e) {
            console.error('Failed to compute diff:', e);
            return false;
          }
        },
      clearDiff:
        () => ({ tr, dispatch }: { tr: unknown; dispatch: (tr: unknown) => void }) => {
          // @ts-ignore
          this.storage.active = false;
          // @ts-ignore
          this.storage.parts = [];
          // @ts-ignore
          (tr as any).setMeta(DIFF_PLUGIN_KEY, { updatedAt: Date.now(), active: false });
          if (dispatch) dispatch(tr);
          return true;
        },
    } as any;
  },
  addProseMirrorPlugins() {
    const ext = this;
    return [
      new Plugin({
        key: DIFF_PLUGIN_KEY,
        state: {
          init: () => DecorationSet.empty,
          apply(tr, _value) {
            // @ts-ignore
            if (!ext.storage.active || !(ext.storage.parts && ext.storage.parts.length)) {
              return DecorationSet.empty;
            }
            const doc = tr.doc;
            const decorations: Decoration[] = [];
            const textNodes: Array<{ from: number; to: number; text: string }> = [];
            doc.descendants((node, pos) => {
              if (node.isText) {
                textNodes.push({ from: pos, to: pos + node.nodeSize, text: node.text || '' });
              }
              return true;
            });

            const offsetToPos = (offset: number): number => {
              let acc = 0;
              for (const n of textNodes) {
                const len = n.text.length;
                if (offset <= acc + len) {
                  const within = offset - acc;
                  return n.from + within;
                }
                acc += len;
              }
              return doc.content.size - 1;
            };

            let newIdx = 0;
            // @ts-ignore
            for (const part of ext.storage.parts) {
              if (part.added) {
                const fromOff = newIdx;
                const toOff = newIdx + part.value.length;
                const from = offsetToPos(fromOff);
                const to = offsetToPos(toOff);
                if (from < to) {
                  decorations.push(Decoration.inline(from, to, { class: 'diff-insert' }));
                }
                newIdx += part.value.length;
              } else if (part.removed) {
                const at = offsetToPos(newIdx);
                const value = part.value;
                decorations.push(
                  Decoration.widget(
                    at,
                    () => {
                      const span = document.createElement('span');
                      span.className = 'diff-delete';
                      span.textContent = value;
                      return span;
                    },
                    { side: -1 }
                  )
                );
              } else {
                newIdx += part.value.length;
              }
            }

            const deco = DecorationSet.create(doc, decorations);
            return deco.map(tr.mapping, tr.doc);
          },
        },
        props: {
          decorations(state) {
            return (this as any).getState(state);
          },
        },
      }),
    ];
  },
});

// Formatting toolbar component
function FormattingToolbar({ editor }: { editor: TiptapEditor | null }) {
  if (!editor) {
    return null;
  }

  return (
    <div className="flex flex-wrap gap-1 p-2 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 rounded-t-md">
      {/* Text formatting */}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleBold().run()}
        className={editor.isActive('bold') ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Bold className="w-4 h-4" />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleItalic().run()}
        className={editor.isActive('italic') ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Italic className="w-4 h-4" />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleStrike().run()}
        className={editor.isActive('strike') ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Strikethrough className="w-4 h-4" />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleCode().run()}
        className={editor.isActive('code') ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Code className="w-4 h-4" />
      </Button>

      {/* Separator */}
      <div className="w-px h-6 bg-gray-300 dark:bg-gray-600 mx-1" />

      {/* Headings */}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
        className={editor.isActive('heading', { level: 1 }) ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Heading1 className="w-4 h-4" />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
        className={editor.isActive('heading', { level: 2 }) ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Heading2 className="w-4 h-4" />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
        className={editor.isActive('heading', { level: 3 }) ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Heading3 className="w-4 h-4" />
      </Button>

      {/* Separator */}
      <div className="w-px h-6 bg-gray-300 dark:bg-gray-600 mx-1" />

      {/* Lists */}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleBulletList().run()}
        className={editor.isActive('bulletList') ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <List className="w-4 h-4" />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleOrderedList().run()}
        className={editor.isActive('orderedList') ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <ListOrdered className="w-4 h-4" />
      </Button>

      {/* Separator */}
      <div className="w-px h-6 bg-gray-300 dark:bg-gray-600 mx-1" />

      {/* Code block */}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleCodeBlock().run()}
        className={editor.isActive('codeBlock') ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Code className="w-4 h-4" />
        <span className="ml-1 text-xs">Block</span>
      </Button>

      {/* Quote */}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleBlockquote().run()}
        className={editor.isActive('blockquote') ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Quote className="w-4 h-4" />
      </Button>

      {/* Separator */}
      <div className="w-px h-6 bg-gray-300 dark:bg-gray-600 mx-1" />

      {/* Undo/Redo */}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().undo().run()}
        disabled={!editor.can().undo()}
      >
        <Undo className="w-4 h-4" />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().redo().run()}
        disabled={!editor.can().redo()}
      >
        <Redo className="w-4 h-4" />
      </Button>
    </div>
  );
}

export function ImageLoader({ article, newImageGenerationRequestId, stagedImageUrl, setStagedImageUrl }: {
  article: ArticleListItem | null | undefined,
  newImageGenerationRequestId: string | null | undefined,
  stagedImageUrl: string | null | undefined,
  setStagedImageUrl: (url: string | null | undefined) => void
}) {
  const [imageUrl, setImageUrl] = useState<string | null>(null);

  useEffect(() => {
    const requestToFetch = newImageGenerationRequestId || article?.article.image_generation_request_id || null;
    async function fetchImageGeneration() {
      if (requestToFetch) {
        const imgGen = await getImageGeneration(requestToFetch);
        if (imgGen) {
          if (imgGen.outputUrl) {
            setImageUrl(imgGen.outputUrl);
            setStagedImageUrl(imgGen.outputUrl);
          } else {
            const status = await getImageGenerationStatus(requestToFetch);
            if (status.outputUrl) {
              setImageUrl(status.outputUrl);
              setStagedImageUrl(status.outputUrl);
            }
          }
        }
      }
    }
    fetchImageGeneration();

    if (stagedImageUrl !== undefined) {
      setImageUrl(stagedImageUrl);
    } else if (article && article.article.image_url) {
      setImageUrl(article.article.image_url);
    }
  }, [article, stagedImageUrl, newImageGenerationRequestId]);

  if (!article) {
    return null;
  }

  if (imageUrl) {
    return (
      <div className='flex items-center justify-center'>
        <img className='rounded-md aspect-video object-cover' src={imageUrl} alt={article.article.title || ''} width={'100%'} />
      </div>
    )
  }

  return null;
}

export default function ArticleEditor({ isNew }: { isNew?: boolean }) {
  const { toast } = useToast()
  const navigate = useNavigate();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  
  // Only use useParams when editing an existing article
  const params = !isNew ? useParams({ from: '/dashboard/blog/edit/$blogSlug' }) : null;
  const blogSlug = params?.blogSlug;
  
  // Loading states are now handled by React Query mutations
  // const [isLoading, setIsLoading] = useState(false);
  // const [isSaving, setIsSaving] = useState(false);
  const [generatingImage, setGeneratingImage] = useState(false);
  const [newImageGenerationRequestId, setNewImageGenerationRequestId] = useState<string | null>(null);
  const [stagedImageUrl, setStagedImageUrl] = useState<string | null | undefined>(undefined);
  const [generateImageOpen, setGenerateImageOpen] = useState(false);
  const [imageModalOpen, setImageModalOpen] = useState(false);
  const [generatingRewrite, setGeneratingRewrite] = useState(false);
  const [sourcesManagerOpen, setSourcesManagerOpen] = useState(false);
  const [sourcesRefreshTrigger] = useState(0);
  
  // Image versioning state
  const [imageVersions, setImageVersions] = useState<Array<{ url: string; prompt?: string; timestamp: number }>>([]);
  const [currentVersionIndex, setCurrentVersionIndex] = useState(-1);
  const [previewImageUrl, setPreviewImageUrl] = useState<string>('');

  // Image versioning functions
  const addImageVersion = (url: string, prompt?: string) => {
    const newVersion = { url, prompt, timestamp: Date.now() };
    setImageVersions(prev => [...prev, newVersion]);
    setCurrentVersionIndex(prev => prev + 1);
    setPreviewImageUrl(url);
    setStagedImageUrl(url);
    setValue('image_url', url);
  };

  const selectImageVersion = (index: number) => {
    if (index >= 0 && index < imageVersions.length) {
      setCurrentVersionIndex(index);
      const selectedVersion = imageVersions[index];
      setPreviewImageUrl(selectedVersion.url);
      setStagedImageUrl(selectedVersion.url);
      setValue('image_url', selectedVersion.url);
    }
  };

  const removeImageVersion = (index: number) => {
    if (imageVersions.length > 1) {
      const newVersions = imageVersions.filter((_, i) => i !== index);
      setImageVersions(newVersions);
      
      if (index === currentVersionIndex) {
        const newIndex = Math.max(0, Math.min(index, newVersions.length - 1));
        setCurrentVersionIndex(newIndex);
        if (newVersions[newIndex]) {
          setPreviewImageUrl(newVersions[newIndex].url);
          setStagedImageUrl(newVersions[newIndex].url);
          setValue('image_url', newVersions[newIndex].url);
        }
      } else if (index < currentVersionIndex) {
        setCurrentVersionIndex(prev => prev - 1);
      }
    }
  };

  /* --------------------------------------------------------------------- */
  /* Chat (right-hand panel)                                               */
  /* --------------------------------------------------------------------- */
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [chatLoading, setChatLoading] = useState(false);
  const [chatInput, setChatInput] = useState('');
  const [isThinking, setIsThinking] = useState(false);
  const [thinkingMessage, setThinkingMessage] = useState<string>('Thinking...');
  const chatMessagesRef = useRef<HTMLDivElement>(null);
  
  // (deprecated) pending edit/patch state removed in favor of inline diffs
  
  // Track processed tool messages to avoid re-applying old patches
  const [processedToolMessages, setProcessedToolMessages] = useState<Set<string>>(new Set());

  // Use React Query to fetch article data
  const { data: article, isLoading: articleLoading, error } = useQuery({
    queryKey: ['article', blogSlug],
    queryFn: () => getArticle(blogSlug as string),
    enabled: !isNew && !!blogSlug,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });

  // Mutation for creating new articles
  const createArticleMutation = useMutation({
    mutationFn: (data: {
      title: string;
      content: string;
      image_url?: string;
      tags: string[];
      isDraft: boolean;
      authorId: number;
    }) => createArticle(data),
    onSuccess: () => {
      toast({ title: "Success", description: "Article created successfully." });
      queryClient.invalidateQueries({ queryKey: ['articles'] });
      navigate({ to: '/dashboard/blog' });
    },
    onError: (error) => {
      console.error('Error creating article:', error);
      toast({ title: "Error", description: "Failed to create article. Please try again.", variant: "destructive" });
    }
  });

  // Mutation for updating existing articles
  const updateArticleMutation = useMutation({
    mutationFn: (data: {
      slug: string;
      updateData: {
        title: string;
        content: string;
        image_url?: string;
        tags: string[];
        is_draft: boolean;
        published_at: number | null;
      };
      returnToDashboard?: boolean;
    }) => updateArticle(data.slug, data.updateData),
    onSuccess: (response, variables) => {
      toast({ title: "Success", description: "Article updated successfully." });
      
      const newSlug = response?.article?.slug;
      const oldSlug = variables.slug;
      const articleId = response?.article?.id;
      
      // Update single article cache
      if (newSlug && newSlug !== oldSlug) {
        // Slug changed - update URL without triggering navigation/refresh
        const newUrl = `/dashboard/blog/edit/${newSlug}`;
        window.history.replaceState(null, '', newUrl);
        
        // Remove old slug cache and set new slug cache
        queryClient.removeQueries({ queryKey: ['article', oldSlug] });
        queryClient.setQueryData(['article', newSlug], response);
      } else {
        // Slug unchanged - just update the cache
        queryClient.setQueryData(['article', oldSlug], response);
      }
      
      // Update all article list queries by replacing the matching article
      // This updates dashboard list, sidebar, and any other cached article lists
      queryClient.setQueriesData<{ articles: ArticleListItem[]; total_pages: number; include_drafts: boolean } | undefined>(
        { queryKey: ['articles'], exact: false },
        (oldData) => {
          if (!oldData?.articles) return oldData;
          return {
            ...oldData,
            articles: oldData.articles.map((item) =>
              item.article.id === articleId ? response : item
            ),
          };
        }
      );
      
      // Update sidebar infinite query
      queryClient.setQueriesData<{ pages: { articles: ArticleListItem[] }[] } | undefined>(
        { queryKey: ['sidebar-articles'], exact: false },
        (oldData) => {
          if (!oldData?.pages) return oldData;
          return {
            ...oldData,
            pages: oldData.pages.map((page) => ({
              ...page,
              articles: page.articles.map((item) =>
                item.article.id === articleId ? response : item
              ),
            })),
          };
        }
      );
      
      if (variables.returnToDashboard) {
        navigate({ to: '/dashboard/blog' });
      } else {
        // If we are *not* navigating away, refresh local state:
        const data = variables.updateData;
        setValue('title', data.title);
        setValue('content', data.content);
        setValue('image_url', data.image_url || '');
        setValue('tags', data.tags);
        setValue('isDraft', data.is_draft);
      }
    },
    onError: (error) => {
      console.error('Error updating article:', error);
      toast({ title: "Error", description: "Failed to update article. Please try again.", variant: "destructive" });
    }
  });

  const { register, handleSubmit, setValue, formState: { errors }, control, reset } = useForm<ArticleFormData>({
    resolver: zodResolver(articleSchema),
    defaultValues: {
      title: '',
      content: '',
      image_url: '',
      tags: [],
      isDraft: false,
    }
  });

  // Watch only the specific fields that need reactive UI updates (NOT content - that causes re-renders on every keystroke)
  const watchedTags = useWatch({ control, name: 'tags' });
  const watchedIsDraft = useWatch({ control, name: 'isDraft' });

  const [imagePrompt, setImagePrompt] = useState<string | null>(DEFAULT_IMAGE_PROMPT[Math.floor(Math.random() * DEFAULT_IMAGE_PROMPT.length)]);

  /* --------------------------------------------------------------------- */
  /* Tiptap Editor Setup                                                   */
  /* --------------------------------------------------------------------- */
  const mdParserRef = useRef<MarkdownIt>();
  if (!mdParserRef.current) {
    mdParserRef.current = new MarkdownIt({ typographer: true, html: true });
  }
  const mdParser = mdParserRef.current;

  const [diffing, setDiffing] = useState(false);
  const [originalDocument, setOriginalDocument] = useState<string>('');
  const [pendingNewDocument, setPendingNewDocument] = useState<string>('');
  const [currentDiffReason, setCurrentDiffReason] = useState<string>('');


  // Inline diff lifecycle helpers
  const enterDiffPreview = (oldHtml: string, newHtml: string, reason?: string) => {
    if (!editor) return;
    // Use full HTML content for accurate diff generation - don't strip anything
    const oldText = oldHtml;
    const newText = newHtml;
    editor.commands.setContent(newHtml);
    setOriginalDocument(oldHtml);
    setPendingNewDocument(newHtml);
    setCurrentDiffReason(reason || '');
    setDiffing(true);
    // @ts-ignore custom command provided by DiffHighlighter
    if ((editor as any).commands?.showDiff) {
      // @ts-ignore
      (editor as any).commands.showDiff(oldText, newText);
    }
    // Force a tiny transaction to ensure decorations render even if doc didn't change further
    editor.chain().focus().setTextSelection({ from: 1, to: 1 }).run();
  };

  const clearDiffDecorations = () => {
    if (!editor) return;
    // @ts-ignore
    if ((editor as any).commands?.clearDiff) {
      // @ts-ignore
      (editor as any).commands.clearDiff();
    }
  };

  const acceptDiff = () => {
    if (!editor) return;
    
    // Apply the pending changes
    editor.commands.setContent(pendingNewDocument || editor.getHTML());
    setValue('content', editor.getHTML());
    clearDiffDecorations();
    setDiffing(false);
    setPendingNewDocument('');
    setCurrentDiffReason('');
  };

  const rejectDiff = () => {
    if (!editor) return;
    
    // Revert to original content
    editor.commands.setContent(originalDocument || editor.getHTML());
    setValue('content', originalDocument || editor.getHTML());
    clearDiffDecorations();
    setDiffing(false);
    setPendingNewDocument('');
    setCurrentDiffReason('');
  };

  const editor = useEditor({
    extensions: [
      StarterKit,
      CodeBlock.configure({
        HTMLAttributes: {
          class: 'bg-gray-100 dark:bg-gray-800 p-4 rounded-md border',
        },
      }),
      DiffHighlighter,
    ],
    content: '', // Start empty, content is synced when article loads
    editorProps: {
      attributes: {
        class:
          'w-full p-4 focus:outline-none prose prose-sm max-w-none dark:prose-invert',
      },
    },
    onUpdate({ editor }: { editor: TiptapEditor }) {
      if (!diffing) {
        setValue('content', editor.getHTML());
      }
    },
  });

  if (!user) {
    return <div>Please log in to edit articles.</div>;
  }

  // Consume from ImageLoader and sync with versioning
  useEffect(() => {
    if (stagedImageUrl) {
      setValue('image_url', stagedImageUrl);
      setPreviewImageUrl(stagedImageUrl);
      
      // Add to versions if it's a new URL
      if (!imageVersions.some(v => v.url === stagedImageUrl)) {
        addImageVersion(stagedImageUrl);
      }
    }
  }, [stagedImageUrl, setValue]);

  // Populate form when article data is loaded
  useEffect(() => {
    if (article && !isNew) {
      // Extract tag names from the server response format
      const tagNames = article.tags ? article.tags
        .map((tag: any) => tag?.name?.toUpperCase())
        .filter((name: string | undefined) => !!name && name !== '') : [];
      const newValues = {
        title: article.article.title || '',
        content: article.article.content || '',
        image_url: article.article.image_url || '',
        tags: tagNames,
        isDraft: article.article.is_draft,
      } as ArticleFormData;
      reset(newValues);
      
      // Initialize image versions if there's an existing image
      if (article.article.image_url) {
        setImageVersions([{ url: article.article.image_url, timestamp: Date.now() }]);
        setCurrentVersionIndex(0);
        setPreviewImageUrl(article.article.image_url);
      } else {
        setImageVersions([]);
        setCurrentVersionIndex(-1);
        setPreviewImageUrl('');
      }
      
      // Sync editor with fresh content - load directly as HTML since content is already HTML
      if (editor) {
        editor.commands.setContent(newValues.content);
      }
    } else if (isNew) {
      const blank: ArticleFormData = {
        title: '',
        content: '',
        image_url: '',
        tags: [],
        isDraft: false,
      };
      reset(blank);
      setImageVersions([]);
      setCurrentVersionIndex(-1);
      setPreviewImageUrl('');
      if (editor) {
        editor.commands.setContent('');
      }
    }
  }, [article, isNew, reset, editor]);

  // Load conversation history with artifacts when article is loaded
  useEffect(() => {
    if (article?.article?.id && !isNew) {
      const loadConversations = async () => {
        try {
          const result = await getConversationHistory(article.article.id);
          
          // Backend now handles initial greeting, so always use what it returns
          if (!result.messages || result.messages.length === 0) {
            setChatMessages([]);
            return;
          }
          
          // Convert database messages to chat messages with artifact metadata and tool_context
          const loadedMessages = result.messages.map((msg: any) => {
            const chatMsg: ChatMessage = {
              id: msg.id,
              role: msg.role,
              content: msg.content,
              meta_data: msg.meta_data,
              created_at: msg.created_at,
            };
            
            // Check for new tool_group format first
            if (msg.meta_data?.tool_group) {
              chatMsg.tool_group = msg.meta_data.tool_group;
            }
            // Reconstruct tool_context from legacy metadata for tool execution
            else if (msg.meta_data?.tool_execution) {
              const toolExec = msg.meta_data.tool_execution;
              const output = toolExec.output;
              const toolName = toolExec.tool_name;
              
              // Convert to tool_group format for unified handling
              chatMsg.tool_group = {
                group_id: toolExec.tool_id || `group-${msg.id}`,
                status: toolExec.success ? 'completed' : 'error',
                calls: [{
                  id: toolExec.tool_id || `call-${msg.id}`,
                  name: toolName,
                  input: typeof toolExec.input === 'object' ? toolExec.input : {},
                  status: toolExec.success ? 'completed' : 'error',
                  result: typeof output === 'object' ? output : undefined,
                  error: toolExec.error,
                  started_at: toolExec.executed_at || msg.created_at,
                  duration_ms: toolExec.duration_ms,
                }],
              };
              
              // Also set legacy tool_context for backward compatibility with WebSearchSteps
              if (toolName === 'search_web_sources') {
                chatMsg.tool_context = {
                  tool_name: 'search_web_sources',
                  tool_id: toolExec.tool_id || '',
                  status: 'completed',
                  search_query: output?.query || '',
                  search_results: output?.search_results || [],
                  sources_created: output?.sources_created || [],
                  total_found: output?.total_found || 0,
                  sources_successful: output?.sources_successful || 0,
                  message: output?.message
                };
              } else if (toolName === 'ask_question') {
                chatMsg.tool_context = {
                  tool_name: 'ask_question',
                  tool_id: toolExec.tool_id || '',
                  status: 'completed',
                  answer: output?.answer,
                  citations: output?.citations || [],
                };
              }
            }
            
            return chatMsg;
          }) as ChatMessage[];
          
          setChatMessages(loadedMessages);
        } catch (error) {
          console.error('[Editor] Failed to load conversation history:', error);
          setChatMessages([]);
        }
      };
      
      loadConversations();
    } else if (isNew) {
      setChatMessages([]);
    }
  }, [article?.article?.id, isNew]);

  // Auto-scroll chat to bottom when messages change
  useEffect(() => {
    if (chatMessagesRef.current) {
      chatMessagesRef.current.scrollTop = chatMessagesRef.current.scrollHeight;
    }
  }, [chatMessages]);

  const onSubmit = async (data: ArticleFormData, returnToDashboard: boolean = true) => {
    if (!user) {
      toast({ title: "Error", description: "You must be logged in to edit an article." });
      return;
    }

    // Ensure staged image URL is synced to form data before saving
    const finalImageUrl = stagedImageUrl !== undefined ? stagedImageUrl : data.image_url;

    if (isNew) {
      createArticleMutation.mutate({
        title: data.title,
        content: data.content,
        image_url: finalImageUrl || undefined,
        tags: data.tags,
        isDraft: data.isDraft,
        authorId: user.id,
      });
    } else {        
      const updateData = {
        title: data.title,
        content: data.content, // HTML content from Tiptap editor
        image_url: finalImageUrl || undefined,
        tags: data.tags,
        is_draft: data.isDraft,
        published_at: (() => {
          if (data.isDraft) return null;
          if (article?.article.published_at && article.article.published_at !== '') {
            return typeof article.article.published_at === 'string'
              ? new Date(article.article.published_at).getTime()
              : article.article.published_at;
          }
          return Date.now();
        })(),
      };
      
      updateArticleMutation.mutate({
        slug: blogSlug as string,
        updateData,
        returnToDashboard
      });
    }
  };

  const rewriteArticle = async () => {
    if (!article?.article.id || !editor) return;
    setGeneratingRewrite(true);
    try {
      const oldHtml = editor.getHTML();
      
      const result = await updateArticleWithContext(article.article.id);
      
      if (result.success) {
        const newHtml = result.content;
        const reason = 'Full-document rewrite';
        enterDiffPreview(oldHtml, newHtml, reason);
        // Add a simple message about the proposed changes
        setChatMessages((prev) => [
          ...prev,
          { role: 'assistant', content: '📋 I\'ve prepared a full-document rewrite. Review the changes in the editor and use "Keep All" or "Reject" below.' }
        ]);
      }
    } catch (error) {
      console.error('Error rewriting article:', error);
      toast({ title: 'Error', description: 'Failed to rewrite article. Please try again.' });
    } finally {
      setGeneratingRewrite(false);
    }
  };

  // Apply text edit from AI assistant
  const applyTextEdit = (originalText: string, newText: string, reason: string) => {
    if (!editor) return;
    
    const oldHtml = editor.getHTML();
    const currentText = editor.getText();
    const index = currentText.indexOf(originalText);
    if (index === -1) {
      toast({
        title: 'Edit Warning',
        description: 'Could not locate the text to edit. The document may have changed.',
        variant: 'destructive'
      });
      return;
    }
    const from = index;
    const to = index + originalText.length;
    editor.chain().focus().setTextSelection({ from, to }).insertContent(newText).run();
    const newHtml = editor.getHTML();
    enterDiffPreview(oldHtml, newHtml, reason);
  };

  // Apply patch from AI assistant (new implementation)
  const applyPatch = (patch: any, originalText: string, newText: string, reason: string) => {
    if (!editor) return;
    
    const oldHtml = editor.getHTML();
    const currentText = editor.getText();
    const index = currentText.indexOf(originalText);
    if (index === -1) {
      toast({
        title: 'Patch Failed',
        description: 'Could not locate the text to edit. The document may have changed.',
        variant: 'destructive'
      });
      return;
    }
    const from = index;
    const to = index + originalText.length;
    editor.chain().focus().setTextSelection({ from, to }).insertContent(newText).run();
    const newHtml = editor.getHTML();
    enterDiffPreview(oldHtml, newHtml, reason);
  };

  // Apply document rewrite from AI assistant
  const applyDocumentRewrite = (newContent: string, reason: string, originalContent?: string) => {
    if (!editor) return;
    
    const oldHtml = editor.getHTML();
    
    // Convert markdown to HTML for the new content
    const newHtml = mdParser.render(newContent);
    
    // Create diff preview
    enterDiffPreview(oldHtml, newHtml, reason);
  };

  const sendChatWithMessage = async (message: string) => {
    const text = message.trim();
    
    if (!text) {
      return;
    }

    // Get current document content to send separately
    const currentContent = editor?.getText() || '';

    // Check if this looks like an edit request
    const isEditRequest = /\b(rewrite|edit|improve|change|update|fix|enhance|modify)\b/i.test(text);

    // Show original user message in UI
    const baseMessages = [...chatMessages, { role: 'user', content: text } as ChatMessage];
    setChatMessages(baseMessages);
    setChatInput(''); // Clear the input state

    // Add placeholder assistant message
    const assistantIndex = baseMessages.length;
    setChatMessages((prev) => [...prev, { role: 'assistant', content: '' } as ChatMessage]);

    // Send only the new message text - backend will load context from database
    await performChatRequest(text, assistantIndex, isEditRequest, currentContent);
  };

  const sendChat = async () => {
    const text = chatInput.trim();
    
    if (!text) {
      return;
    }

    await sendChatWithMessage(text);
  };

  const performChatRequest = async (messageText: string, assistantIndex: number, isEditRequest: boolean, documentContent: string) => {
    setChatLoading(true);
    try {
      if (!article?.article?.id) {
        throw new Error('Article ID is required');
      }
      
      // Submit the request with single message - backend loads context from DB
      const result = await apiPost<{ requestId: string; status: string }>('/agent', {
        message: messageText,  // Single message string
        documentContent: documentContent,
        articleId: article.article.id  // Required for loading context
      });
      
      if (!result.requestId) {
        throw new Error('No request ID received');
      }

      // Connect to WebSocket and stream the response
      await streamChatResponse(result.requestId, assistantIndex, isEditRequest);

    } catch (err) {
      console.error('Chat error:', err);
      
      // Remove the optimistic message on error
      setChatMessages((prev) => prev.slice(0, -1));
      
      // Show user-friendly error
      if (isAuthError(err)) {
        toast({ 
          title: "Session Expired", 
          description: "Your session has expired. Please log in again.",
          variant: "destructive"
        });
      } else if (err instanceof Error) {
        if (err.message.includes('Failed to fetch') || err.message.includes('NetworkError')) {
          toast({ 
            title: "Connection Error", 
            description: "Cannot connect to the writing assistant. Make sure the backend server is running.",
            variant: "destructive"
          });
        } else {
          toast({ 
            title: "Error", 
            description: err.message || "An error occurred while processing your request.",
            variant: "destructive"
          });
        }
      }
    } finally {
      setChatLoading(false);
    }
  };

  const streamChatResponse = async (requestId: string, assistantIndex: number, isEditRequest: boolean) => {
    return new Promise<void>((resolve, reject) => {
      const wsUrl = `${VITE_API_BASE_URL.replace('http://', 'ws://').replace('https://', 'wss://')}/websocket`;
      
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        ws.send(JSON.stringify({
          action: 'subscribe',
          requestId: requestId
        }));
      };

      let currentAssistantContent = '';
      let hasInitialContent = false;
      let placeholderRemoved = false; // Track if the assistant placeholder was removed by a tool message

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          
          if (msg.error) {
            console.error('Stream error:', msg.error);
            toast({ 
              title: "Assistant Error", 
              description: msg.error,
              variant: "destructive"
            });
            setChatMessages((prev) => prev.slice(0, -1));
            ws.close();
            reject(new Error(msg.error));
            return;
          }

          // Handle new block-based message types
          if (msg.type) {
            switch (msg.type) {
              case 'thinking':
                // Handle thinking state - show shimmer
                setIsThinking(true);
                setThinkingMessage(msg.thinking_message || 'Thinking...');
                break;

              case 'content_delta':
                // Handle real-time content chunks
                setIsThinking(false);
                if (msg.content) {
                  setChatMessages((prev) => {
                    const updated = [...prev];
                    if (updated[assistantIndex]) {
                      updated[assistantIndex] = {
                        ...updated[assistantIndex],
                        content: (updated[assistantIndex].content || '') + msg.content
                      };
                    }
                    return updated;
                  });
                }
                break;

              case 'user':
                // User message blocks (context) - no action needed
                break;
                
              case 'system':
                // System message blocks (context) - no action needed
                break;
                
              case 'text':
                // Hide thinking state on first text chunk
                setIsThinking(false);
                // Handle assistant text responses as separate messages
                if (msg.content) {
                  // If this is the first text block
                  if (!hasInitialContent) {
                    currentAssistantContent = msg.content;
                    hasInitialContent = true;
                    
                    // If placeholder was removed by a tool message, append to end
                    // Otherwise update the placeholder in place
                    if (placeholderRemoved) {
                      setChatMessages((prev) => [
                        ...prev,
                        { role: 'assistant', content: currentAssistantContent } as ChatMessage
                      ]);
                    } else {
                      setChatMessages((prev) => {
                        const updated = [...prev];
                        updated[assistantIndex] = { 
                          role: 'assistant', 
                          content: currentAssistantContent 
                        } as ChatMessage;
                        return updated;
                      });
                    }
                  } else {
                    // For subsequent text blocks, add as new messages
                    setChatMessages((prev) => [
                      ...prev,
                      { role: 'assistant', content: msg.content }
                    ]);
                  }
                }
                break;
                
              case 'tool_use':
                // Hide thinking state on tool use
                setIsThinking(false);
                
                // Display tool usage feedback using tool_context for all tools
                if (msg.tool_name) {
                  // All tools now use tool_context for unified rendering
                  const toolContext: ChatMessage['tool_context'] = {
                    tool_name: msg.tool_name,
                    tool_id: msg.tool_id || '',
                    status: 'running',
                    input: msg.tool_input,
                  };
                  
                  // Add tool-specific fields
                  if (msg.tool_name === 'search_web_sources') {
                    toolContext.search_query = msg.tool_input?.query || 'researching...';
                  } else if (msg.tool_name === 'edit_text' || msg.tool_name === 'rewrite_document') {
                    toolContext.reason = msg.tool_input?.reason || '';
                  }
                  
                  setChatMessages((prev) => [
                    ...prev,
                    { 
                      role: 'assistant', 
                      content: '',
                      tool_context: toolContext
                    }
                  ]);
                }
                break;
                
              case 'tool_result':
                // Hide thinking state on tool result
                setIsThinking(false);
                
                // Handle tool results with structured data
                if (msg.tool_result) {
                  // Create a unique identifier for this tool message
                  const toolMessageId = `${requestId}-${Date.now()}-tool-result`;
                  const isNewMessage = !processedToolMessages.has(toolMessageId);
                  
                  try {
                    if (msg.tool_result.content) {
                      const toolResult = JSON.parse(msg.tool_result.content);
                      
                      // Update the matching tool_context message with completed status
                      setChatMessages((prev) => {
                        const updated = [...prev];
                        // Find last matching tool_use message that's still running
                        for (let i = updated.length - 1; i >= 0; i--) {
                          if (updated[i].tool_context?.tool_name === toolResult.tool_name 
                              && (updated[i].tool_context?.status === 'running' || updated[i].tool_context?.status === 'starting')) {
                            
                            // Build updated tool context based on tool type
                            const updatedContext: ChatMessage['tool_context'] = {
                              ...updated[i].tool_context!,
                              status: 'completed',
                              output: toolResult,
                            };
                            
                            // Add tool-specific fields
                            if (toolResult.tool_name === 'search_web_sources') {
                              updatedContext.search_results = toolResult.search_results || [];
                              updatedContext.sources_created = toolResult.sources_created || [];
                              updatedContext.total_found = toolResult.total_found || 0;
                              updatedContext.sources_successful = toolResult.sources_successful || 0;
                              updatedContext.message = toolResult.message;
                            } else if (toolResult.tool_name === 'edit_text' || toolResult.tool_name === 'rewrite_document') {
                              updatedContext.reason = toolResult.reason;
                            }
                            
                            updated[i] = {
                              ...updated[i],
                              tool_context: updatedContext
                            };
                            break;
                          }
                        }
                        return updated;
                      });
                      
                      // For edit tools, also apply the diff preview
                      if (toolResult.tool_name === 'edit_text' && isNewMessage) {
                        if (toolResult.edit_type === 'patch' && toolResult.patch) {
                          applyPatch(toolResult.patch, toolResult.original_text, toolResult.new_text, toolResult.reason);
                        } else {
                          applyTextEdit(toolResult.original_text, toolResult.new_text, toolResult.reason);
                        }
                        setProcessedToolMessages(prev => new Set(prev).add(toolMessageId));
                      } else if (toolResult.tool_name === 'rewrite_document' && isNewMessage) {
                        applyDocumentRewrite(toolResult.new_content, toolResult.reason, toolResult.original_content);
                        setProcessedToolMessages(prev => new Set(prev).add(toolMessageId));
                      }
                    }
                  } catch (error) {
                    // Update tool context to error state
                    setChatMessages((prev) => {
                      const updated = [...prev];
                      for (let i = updated.length - 1; i >= 0; i--) {
                        if (updated[i].tool_context?.status === 'running') {
                          updated[i] = {
                            ...updated[i],
                            tool_context: {
                              ...updated[i].tool_context!,
                              status: 'error',
                              message: 'Tool execution completed with warnings'
                            }
                          };
                          break;
                        }
                      }
                      return updated;
                    });
                  }
                }
                break;

              case 'full_message':
                // Handle full message with complete meta_data for artifact tools
                setIsThinking(false);
                if (msg.full_message) {
                  // Remove the empty assistant placeholder and add the full message
                  // This ensures correct ordering: tool message comes before any follow-up text
                  placeholderRemoved = true;
                  
                  // Build the chat message with tool_group conversion (same as loadConversations)
                  const fullMsg = msg.full_message;
                  const chatMsg: ChatMessage = {
                    id: fullMsg.id,
                    role: fullMsg.role as 'assistant',
                    content: fullMsg.content,
                    meta_data: fullMsg.meta_data,
                    created_at: fullMsg.created_at,
                  };
                  
                  // Convert tool_execution to tool_group format for unified rendering
                  if (fullMsg.meta_data?.tool_execution) {
                    const toolExec = fullMsg.meta_data.tool_execution;
                    const output = toolExec.output;
                    const toolName = toolExec.tool_name;
                    
                    // Convert to tool_group format
                    chatMsg.tool_group = {
                      group_id: toolExec.tool_id || `group-${fullMsg.id}`,
                      status: toolExec.success !== false ? 'completed' : 'error',
                      calls: [{
                        id: toolExec.tool_id || `call-${fullMsg.id}`,
                        name: toolName,
                        input: typeof toolExec.input === 'object' ? toolExec.input : {},
                        status: toolExec.success !== false ? 'completed' : 'error',
                        result: typeof output === 'object' ? output : undefined,
                        error: toolExec.error,
                        started_at: toolExec.executed_at || fullMsg.created_at,
                        duration_ms: toolExec.duration_ms,
                      }],
                    };
                    
                    // Also set legacy tool_context for backward compatibility
                    if (toolName === 'search_web_sources') {
                      chatMsg.tool_context = {
                        tool_name: 'search_web_sources',
                        tool_id: toolExec.tool_id || '',
                        status: 'completed',
                        search_query: output?.query || '',
                        search_results: output?.search_results || [],
                        sources_created: output?.sources_created || [],
                        total_found: output?.total_found || 0,
                        sources_successful: output?.sources_successful || 0,
                        message: output?.message
                      };
                    } else if (toolName === 'ask_question') {
                      chatMsg.tool_context = {
                        tool_name: 'ask_question',
                        tool_id: toolExec.tool_id || '',
                        status: 'completed',
                        answer: output?.answer,
                        citations: output?.citations || [],
                      };
                    }
                  }
                  
                  setChatMessages((prev) => {
                    const updated = [...prev];
                    // Remove the empty placeholder at assistantIndex if it exists and is empty
                    if (updated[assistantIndex]?.content === '' && 
                        !updated[assistantIndex]?.tool_context && 
                        !updated[assistantIndex]?.meta_data) {
                      updated.splice(assistantIndex, 1);
                    }
                    // Add the full message with tool_group
                    updated.push(chatMsg);
                    return updated;
                  });
                  
                  // Apply diff if it's an edit tool
                  const artifact = fullMsg.meta_data?.artifact;
                  const toolExec = fullMsg.meta_data?.tool_execution;
                  if (artifact && toolExec?.output) {
                    const toolName = toolExec.tool_name;
                    if (toolName === 'edit_text') {
                      applyTextEdit(toolExec.output.original_text, toolExec.output.new_text, toolExec.output.reason);
                    } else if (toolName === 'rewrite_document') {
                      applyDocumentRewrite(toolExec.output.new_content, toolExec.output.reason, toolExec.output.original_content);
                    }
                  }
                }
                break;
                
              case 'done':
                setIsThinking(false); // Hide thinking state on completion
                ws.close();
                
                // After response is complete, check if we should show a document edit option
                if (isEditRequest && currentAssistantContent.length > 100) {
                  const codeBlockMatch = currentAssistantContent.match(/```(?:markdown|md)?\n([\s\S]*?)\n```/);
                  if (codeBlockMatch) {
                    const suggestedContent = codeBlockMatch[1].trim();
                    if (suggestedContent.length > 50) {
                      const oldHtml = editor?.getHTML() || '';
                      const newHtml = mdParser.render(suggestedContent);
                      const reason = 'AI-suggested content from code block';
                      enterDiffPreview(oldHtml, newHtml, reason);
                    }
                  }
                }
                
                resolve();
                break;
                
              case 'error':
                console.error('Stream error:', msg.error);
                toast({ 
                  title: "Assistant Error", 
                  description: msg.error,
                  variant: "destructive"
                });
                setChatMessages((prev) => prev.slice(0, -1));
                ws.close();
                reject(new Error(msg.error));
                break;
            }
          }
          
          // Backward compatibility: Handle legacy role-based messages
          else if (msg.role === 'tool' && msg.content) {
            // Create a unique identifier for this tool message
            const toolMessageId = `${requestId}-${Date.now()}-${msg.content.slice(0, 50)}`;
            const isNewMessage = !processedToolMessages.has(toolMessageId);
            
            try {
              // Parse the tool result content to extract artifacts
              const toolResult = JSON.parse(msg.content);
              
              // Handle edit_text tool specifically - only for new messages
              if (toolResult.tool_name === 'edit_text' && isNewMessage) {
                if (toolResult.edit_type === 'patch' && toolResult.patch) {
                  applyPatch(toolResult.patch, toolResult.original_text, toolResult.new_text, toolResult.reason);
                } else {
                  applyTextEdit(toolResult.original_text, toolResult.new_text, toolResult.reason);
                }
                
                // Mark this tool message as processed
                setProcessedToolMessages(prev => new Set(prev).add(toolMessageId));
              } else if (toolResult.tool_name === 'rewrite_document' && isNewMessage) {
                // Handle document rewrite with diff preview
                applyDocumentRewrite(toolResult.new_content, toolResult.reason, toolResult.original_content);
                
                // Mark this tool message as processed
                setProcessedToolMessages(prev => new Set(prev).add(toolMessageId));
              }
            } catch (error) {
              // Add error message
              setChatMessages((prev) => [
                ...prev,
                { role: 'assistant', content: '⚠️ Tool execution completed with warnings' }
              ]);
            }
          }
          
          // Backward compatibility: Handle legacy assistant messages
          else if (msg.role === 'assistant' && msg.content) {
            // For legacy messages, treat them as text blocks
            if (!hasInitialContent) {
              currentAssistantContent = msg.content;
              hasInitialContent = true;
              setChatMessages((prev) => {
                const updated = [...prev];
                updated[assistantIndex] = { 
                  role: 'assistant', 
                  content: currentAssistantContent 
                } as ChatMessage;
                return updated;
              });
            } else {
              // Add as new message
              setChatMessages((prev) => [
                ...prev,
                { role: 'assistant', content: msg.content }
              ]);
            }
          }
          
          // Backward compatibility: Handle legacy done signal
          else if (msg.done) {
            ws.close();
            
            // After response is complete, check if we should show a document edit option
            if (isEditRequest && currentAssistantContent.length > 100) {
              const codeBlockMatch = currentAssistantContent.match(/```(?:markdown|md)?\n([\s\S]*?)\n```/);
              if (codeBlockMatch) {
                const suggestedContent = codeBlockMatch[1].trim();
                if (suggestedContent.length > 50) {
                  const oldHtml = editor?.getHTML() || '';
                  const newHtml = mdParser.render(suggestedContent);
                  const reason = 'AI-suggested content from code block (legacy)';
                  enterDiffPreview(oldHtml, newHtml, reason);
                }
              }
            }
            
            resolve();
          }
        } catch (parseError) {
          console.error('Failed to parse WebSocket message:', parseError);
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        toast({ 
          title: "Connection Error", 
          description: "Failed to connect to WebSocket for real-time streaming",
          variant: "destructive"
        });
        setChatMessages((prev) => prev.slice(0, -1));
        reject(error);
      };

      ws.onclose = (event) => {
        if (event.code !== 1000) { // 1000 is normal closure
          console.error('WebSocket closed unexpectedly:', event.code, event.reason);
        }
      };

      // Set a timeout to prevent hanging
      setTimeout(() => {
        if (ws.readyState !== WebSocket.CLOSED) {
          ws.close();
          reject(new Error('WebSocket timeout'));
        }
      }, 120000); // 2 minutes timeout
    });
  };

  // Show loading state while fetching article
  if (articleLoading && !isNew) {
    return (
      <section className="flex-1 p-0 md:p-4">
        <div className="flex items-center justify-center h-64">
          <div>Loading article...</div>
        </div>
      </section>
    );
  }

  // Show error state if fetch failed
  if (error && !isNew) {
    return (
      <section className="flex-1 p-0 md:p-4">
        <div className="flex items-center justify-center h-64">
          <div>Error loading article. Please try again.</div>
        </div>
      </section>
    );
  }

  return (
    <section className="flex gap-4 p-0 md:p-4 h-[calc(100vh-60px)]">
      <div className="flex-1">
        {/* Article Metadata Card */}
        
            {/* Article Title Section */}
            <div className="mb-6">
              <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4">
                <div className="flex-1 w-full sm:w-auto">
                  <Input
                    {...register('title')}
                    placeholder="Article Title"
                    className="w-full text-lg font-medium"
                  />
                  {errors.title && <p className="text-red-500 text-sm mt-1">{errors.title.message}</p>}
                </div>
              </div>
            </div>
            {/* Article Tools Section */}
            <div className="grid grid-cols-1 lg:grid-cols-5 gap-4 mb-4">
              {/* Header Image Section */}
              <div className="flex flex-col gap-2">
                <Dialog open={imageModalOpen} onOpenChange={setImageModalOpen}>
                  <DialogTrigger asChild>
                    <Card className="w-full h-32 flex items-center justify-center overflow-hidden cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">
                      <div className="flex flex-col px-2 gap-2">
                        <div className="text-xs">Image</div>
                      <ImageLoader
                        article={article}
                        newImageGenerationRequestId={newImageGenerationRequestId}
                        stagedImageUrl={stagedImageUrl}
                        setStagedImageUrl={setStagedImageUrl}
                      />
                      </div>
                      {(!stagedImageUrl && !article?.article.image_url) && (
                        <div className="text-center">
                          <UploadIcon className="w-6 h-6 mx-auto mb-1 text-muted-foreground" />
                          <span className="text-xs text-muted-foreground">Click to add image</span>
                        </div>
                      )}
                    </Card>
                  </DialogTrigger>

                  {/* Modal content for image editing */}
                  <DialogContent className="sm:max-w-4xl">
                    <DialogHeader>
                      <DialogTitle>Edit Header Image</DialogTitle>
                      <DialogDescription>Update or generate a header image for your article.</DialogDescription>
                    </DialogHeader>
                    
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                      {/* Image Preview Section */}
                      <div className="space-y-4">
                        <div className="text-sm font-medium">Preview</div>
                        <div className="aspect-video rounded-lg border-2 border-dashed border-gray-200 dark:border-gray-700 overflow-hidden bg-gray-50 dark:bg-gray-900">
                          {previewImageUrl ? (
                            <img 
                              src={previewImageUrl} 
                              alt="Image preview" 
                              className="w-full h-full object-cover"
                            />
                          ) : (
                            <div className="w-full h-full flex items-center justify-center text-gray-400">
                              <div className="text-center">
                                <UploadIcon className="w-12 h-12 mx-auto mb-2" />
                                <p>No image selected</p>
                              </div>
                            </div>
                          )}
                        </div>
                        
                        {/* Image Versions */}
                        {imageVersions.length > 0 && (
                          <div className="space-y-2">
                            <div className="text-sm font-medium">Versions ({imageVersions.length})</div>
                            <div className="grid grid-cols-3 gap-2 max-h-48 overflow-y-auto">
                              {imageVersions.map((version, index) => (
                                <div
                                  key={index}
                                  className={cn(
                                    "aspect-video rounded-md border-2 cursor-pointer overflow-hidden",
                                    index === currentVersionIndex 
                                      ? "border-indigo-500 ring-2 ring-indigo-200" 
                                      : "border-gray-200 hover:border-gray-300"
                                  )}
                                  onClick={() => selectImageVersion(index)}
                                >
                                  <img 
                                    src={version.url} 
                                    alt={`Version ${index + 1}`}
                                    className="w-full h-full object-cover"
                                  />
                                </div>
                              ))}
                            </div>
                          </div>
                        )}
                      </div>
                      
                      {/* Controls Section */}
                      <div className="space-y-4">
                        <div className="space-y-2">
                          <label className="block text-sm font-medium">Image URL</label>
                          <Input
                            className="w-full"
                            value={previewImageUrl}
                            onChange={(e) => {
                              setPreviewImageUrl(e.target.value);
                              if (e.target.value) {
                                addImageVersion(e.target.value);
                              }
                            }}
                            placeholder="Enter image URL..."
                          />
                          {errors.image_url && <p className="text-red-500 text-sm">{errors.image_url.message}</p>}
                        </div>
                        
                        <div className="space-y-3">
                          <div className="text-sm font-medium">Generate New Image</div>
                          <div className="flex items-center gap-2">
                            <Dialog open={generateImageOpen} onOpenChange={setGenerateImageOpen}>
                              <DialogTrigger asChild>
                                <Button variant="outline" className="flex-1">
                                  <PencilIcon className="w-4 h-4 mr-2 text-indigo-500" /> 
                                  Custom Prompt
                                </Button>
                              </DialogTrigger>
                              <DialogContent className="sm:max-w-[600px]">
                                <DialogHeader>
                                  <DialogTitle>Generate New Image</DialogTitle>
                                  <DialogDescription>
                                    Generate a new image for your article header.
                                  </DialogDescription>
                                </DialogHeader>
                                <div className="flex flex-col items-start gap-4 w-full">
                                  <Textarea
                                    value={imagePrompt || ''}
                                    onChange={(e) => setImagePrompt(e.target.value)}
                                    placeholder="Prompt"
                                    className="h-[300px] w-full"
                                  />
                                </div>
                                <DialogFooter>
                                  <div className="flex justify-end gap-2 w-full">
                                    <DialogClose asChild>
                                      <Button variant="outline">Cancel</Button>
                                    </DialogClose>
                                    <Button
                                      type="submit"
                                      onClick={async () => {
                                        const result = await generateArticleImage(imagePrompt || '', article?.article.id || '');
                                        if (result.success) {
                                          setNewImageGenerationRequestId(result.generationRequestId);
                                          // Add to versions when image is generated
                                          if (result.generationRequestId) {
                                            setTimeout(async () => {
                                              const status = await getImageGenerationStatus(result.generationRequestId);
                                              if (status.outputUrl) {
                                                addImageVersion(status.outputUrl, imagePrompt || '');
                                              }
                                            }, 3000);
                                          }
                                          toast({ title: 'Success', description: 'Image generated successfully.' });
                                          setGenerateImageOpen(false);
                                        } else {
                                          toast({ title: 'Error', description: 'Failed to generate image. Please try again.' });
                                        }
                                      }}
                                    >Generate</Button>
                                  </div>
                                </DialogFooter>
                              </DialogContent>
                            </Dialog>
                            <Button
                              variant="outline"
                              size="icon"
                              disabled={generatingImage}
                              onClick={async (e) => {
                                setGeneratingImage(true);
                                e.preventDefault();
                                const result = await generateArticleImage(article?.article.title || '', article?.article.id || '');
                                if (result.success) {
                                  setNewImageGenerationRequestId(result.generationRequestId);
                                  // Add to versions when image is generated
                                  if (result.generationRequestId) {
                                    setTimeout(async () => {
                                      const status = await getImageGenerationStatus(result.generationRequestId);
                                      if (status.outputUrl) {
                                        addImageVersion(status.outputUrl, article?.article.title || '');
                                      }
                                    }, 3000);
                                  }
                                  toast({ title: 'Success', description: 'Image generated successfully.' });
                                } else {
                                  toast({ title: 'Error', description: 'Failed to generate image. Please try again.' });
                                }
                                setGeneratingImage(false);
                              }}
                            >
                              <SparklesIcon className={cn('w-4 h-4 text-indigo-500', generatingImage && 'animate-spin')} />
                            </Button>
                          </div>
                        </div>
                        
                        {/* Version Controls */}
                        {imageVersions.length > 0 && (
                          <div className="space-y-3">
                            <div className="text-sm font-medium">Version Controls</div>
                            <div className="flex items-center gap-2">
                              <Button
                                variant="outline"
                                size="sm"
                                disabled={currentVersionIndex <= 0}
                                onClick={() => selectImageVersion(currentVersionIndex - 1)}
                              >
                                Previous
                              </Button>
                              <span className="text-sm text-gray-500 flex-1 text-center">
                                {currentVersionIndex + 1} of {imageVersions.length}
                              </span>
                              <Button
                                variant="outline"
                                size="sm"
                                disabled={currentVersionIndex >= imageVersions.length - 1}
                                onClick={() => selectImageVersion(currentVersionIndex + 1)}
                              >
                                Next
                              </Button>
                            </div>
                            {currentVersionIndex >= 0 && imageVersions.length > 1 && (
                              <Button
                                variant="destructive"
                                size="sm"
                                className="w-full"
                                onClick={() => removeImageVersion(currentVersionIndex)}
                              >
                                Delete Current Version
                              </Button>
                            )}
                          </div>
                        )}
                      </div>
                    </div>
                    
                    <DialogFooter>
                      <DialogClose asChild>
                        <Button variant="outline">Cancel</Button>
                      </DialogClose>
                      <DialogClose asChild>
                        <Button onClick={() => {
                          if (previewImageUrl) {
                            setStagedImageUrl(previewImageUrl);
                            setValue('image_url', previewImageUrl);
                          }
                        }}>
                          Apply Image
                        </Button>
                      </DialogClose>
                    </DialogFooter>
                  </DialogContent>
                </Dialog>
              </div>

              {/* Sources Preview Section */}
              <SourcesPreview
                articleId={article?.article.id}
                onOpenDrawer={() => setSourcesManagerOpen(true)}
                disabled={!article && isNew}
                refreshTrigger={sourcesRefreshTrigger}
              />

              {/* Tags Section */}
              <div className="space-y-3">
                <Drawer direction="right">
                  <DrawerTrigger asChild>
                    <Card className="p-3 h-32 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">
                      <div className="h-full flex flex-col">
                        <div className="flex-1 space-y-1">
                          {watchedTags && watchedTags.length > 0 ? (
                            <div className="flex flex-wrap gap-1">
                              {watchedTags.slice(0, 3).map((tag, idx) => (
                                <span
                                  key={idx}
                                  className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200"
                                >
                                  {tag}
                                </span>
                              ))}
                              {watchedTags.length > 3 && (
                                <span className="text-xs text-muted-foreground">
                                  +{watchedTags.length - 3} more
                                </span>
                              )}
                            </div>
                          ) : (
                            <div className="text-center text-muted-foreground">
                              <span className="text-xs">Click to add tags</span>
                            </div>
                          )}
                        </div>
                        <div className="text-xs text-muted-foreground mt-2">
                          {watchedTags?.length || 0} tag{(watchedTags?.length || 0) !== 1 ? 's' : ''}
                        </div>
                      </div>
                    </Card>
                  </DrawerTrigger>

                  {/* Drawer content for tags editing */}
                  <DrawerContent className="w-full sm:max-w-sm ml-auto">
                    <DrawerHeader>
                      <DrawerTitle>Edit Tags</DrawerTitle>
                      <DrawerDescription>Add or remove tags for your article.</DrawerDescription>
                    </DrawerHeader>
                    <div className="space-y-4 px-4">
                      <div className="space-y-2">
                        <label className="block text-md font-medium leading-6 text-gray-900 dark:text-white">Article Tags</label>
                        <ChipInput
                          value={watchedTags}
                          onChange={(tags) => setValue('tags', tags.map((tag: string) => tag.toUpperCase()))}
                          placeholder="Type and press Enter to add tags..."
                        />
                        {errors.tags && <p className="text-red-500 text-sm">{errors.tags.message}</p>}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        Tags help categorize your article and make it easier to find. Press Enter or comma to add a tag.
                      </div>
                    </div>
                    <DrawerFooter>
                      <DrawerClose asChild>
                        <Button variant="outline" className="w-full">Done</Button>
                      </DrawerClose>
                    </DrawerFooter>
                  </DrawerContent>
                </Drawer>
              </div>

              {/* Publishing Settings Section */}
              <div className="space-y-3">
                <Drawer direction="right">
                  <DrawerTrigger asChild>
                    <Card className="p-3 h-32 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">
                      <div className="h-full flex flex-col justify-between">
                        <div className="space-y-2">
                          <div className="flex items-center justify-between">
                            <span className="text-xs font-medium">Status:</span>
                            <span className={cn("text-xs font-medium", watchedIsDraft ? "text-orange-600" : "text-green-600")}>
                              {watchedIsDraft ? "Draft" : "Published"}
                            </span>
                          </div>
                          <div className="flex items-center justify-between">
                            <span className="text-xs font-medium">Date:</span>
                            <span className="text-xs text-muted-foreground">
                              {article?.article.published_at ? format(new Date(article.article.published_at), 'MMM d') : 'Not set'}
                            </span>
                          </div>
                        </div>
                        <div className="text-xs text-muted-foreground">
                          Click to edit settings
                        </div>
                      </div>
                    </Card>
                  </DrawerTrigger>

                  {/* Drawer content for publishing settings */}
                  <DrawerContent className="w-full sm:max-w-sm ml-auto">
                    <DrawerHeader>
                      <DrawerTitle>Publishing Settings</DrawerTitle>
                      <DrawerDescription>Configure publication status and date.</DrawerDescription>
                    </DrawerHeader>
                    <div className="space-y-4 px-4">
                      {/* Publication Status */}
                      <div className="space-y-3">
                        <div className="flex items-center justify-between">
                          <label htmlFor="isDraft" className="text-sm font-medium">Publication Status</label>
                          <div className="flex items-center gap-2">
                            <span className={cn("text-sm", watchedIsDraft ? "text-muted-foreground" : "text-green-600")}>
                              {watchedIsDraft ? "Draft" : "Published"}
                            </span>
                            <Switch
                              id="isDraft"
                              checked={!watchedIsDraft}
                              onCheckedChange={(checked) => {
                                setValue('isDraft', !checked);
                              }}
                            />
                          </div>
                        </div>

                        {/* Published Date */}
                        <div className="space-y-2">
                          <label htmlFor="publishedAt" className="text-sm font-medium">Publication Date</label>
                          <Popover>
                            <PopoverTrigger asChild>
                              <Button
                                variant={"outline"}
                                className={cn(
                                  'w-full justify-start text-left font-normal',
                                  !article?.article.published_at && 'text-muted-foreground'
                                )}
                              >
                                <CalendarIcon className="mr-2 h-4 w-4" />
                                {article?.article.published_at ? format(new Date(article.article.published_at), 'PPP') : 'Pick a date'}
                              </Button>
                            </PopoverTrigger>
                            <PopoverContent className="w-auto p-0">
                              <Calendar
                                mode="single"
                                selected={article?.article.published_at ? new Date(article.article.published_at) : undefined}
                                onSelect={() => {
                                  /* Not a form field; selection handled elsewhere if needed */
                                }}
                                initialFocus
                              />
                            </PopoverContent>
                          </Popover>
                        </div>
                      </div>
                    </div>
                    <DrawerFooter>
                      <DrawerClose asChild>
                        <Button variant="outline" className="w-full">Done</Button>
                      </DrawerClose>
                    </DrawerFooter>
                  </DrawerContent>
                </Drawer>
              </div>

              {/* Actions Section */}
              <div className="space-y-3">
                <Card className="p-3 h-32 space-y-2">
                  {!isNew && (
                    <>
                      <Link
                        to="/blog"
                        params={{ slug: article?.article.slug || '' }}
                        search={{ page: undefined, tag: undefined, search: undefined }}
                        target="_blank"
                        className="flex items-center gap-2 text-xs text-gray-900 dark:text-white hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors p-1 rounded hover:bg-gray-50 dark:hover:bg-gray-800"
                      >
                        <ExternalLinkIcon className="w-3 h-3" />
                        View Article
                      </Link>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="w-full text-xs justify-start h-7"
                        onClick={rewriteArticle}
                        disabled={generatingRewrite}
                      >
                        <RefreshCw className={cn('w-3 h-3 mr-2 text-indigo-500', generatingRewrite && 'animate-spin')} /> 
                        Regenerate
                      </Button>
                    </>
                  )}
                  {isNew && (
                    <div className="text-xs text-muted-foreground text-center py-8">
                      Save article to access actions
                    </div>
                  )}
                </Card>
              </div>

            </div>

          <form className="">

              <div className="border border-gray-300 dark:border-gray-600 rounded-md">
                <FormattingToolbar editor={editor} />
                <EditorContent
                  editor={editor}
                  className="tiptap w-full border-none rounded-b-md h-[calc(100vh-400px)] overflow-y-auto focus:outline-none"
                />
                {errors.content && <p className="text-red-500">{errors.content.message}</p>}
              </div>

<div className="w-full flex flex-row gap-2 justify-between mt-4">
              <Button variant="secondary">
                <Link to="/dashboard/blog">
                  {isNew ? 'Cancel' : 'Go Back'}
                </Link>
              </Button>
              <div className='flex items-center justify-center gap-2'>
                {!isNew && 
                  <Button
                    variant="outline"
                  type="submit"
                  onClick={() => {
                    if (diffing) {
                      // Reject pending diff changes before saving
                      rejectDiff();
                    }
                    handleSubmit((data) => onSubmit(data, false))();
                  }}
                  disabled={updateArticleMutation.isPending}>
                   {updateArticleMutation.isPending ? 'Saving...' : 'Save'}
                  </Button>
                }
              <Button type='submit' disabled={createArticleMutation.isPending || updateArticleMutation.isPending} onClick={() => {
                if (diffing) {
                  // Reject pending diff changes before saving
                  rejectDiff();
                }
                handleSubmit((data) => onSubmit(data, true))();
              }}>
                {(createArticleMutation.isPending || updateArticleMutation.isPending) ? 
                  (isNew ? 'Creating...' : 'Updating...') : 
                  (isNew ? 'Create Article' : 'Save & Return')
                }
              </Button>
              </div>
              </div>

          </form>

      </div>

      {/* Chat side-panel */}
      <div className="hidden xl:flex flex-col w-96 border rounded-md">
        <div ref={chatMessagesRef} className="flex-1 overflow-y-auto p-4 space-y-3">
          {chatMessages.map((m, i) => {
            // Helper function to render tool messages with unified UI
            const renderToolMessage = () => {
              // NEW: Use ToolGroupDisplay for tool_group (new architecture)
              if (m.tool_group) {
                return (
                  <ToolGroupDisplay 
                    key={i} 
                    group={m.tool_group}
                    onArtifactAction={(toolId, action) => {
                      // Handle artifact action for tools like edit_text
                      const call = m.tool_group?.calls.find(c => c.id === toolId);
                      if (call && (call.name === 'edit_text' || call.name === 'rewrite_document')) {
                        const result = call.result;
                        if (action === 'accept' && result && editor) {
                          const oldContent = (result.original_text || result.original_content) as string;
                          const newContent = (result.new_text || result.new_content) as string;
                          const reason = (result.reason as string) || '';
                          
                          if (newContent) {
                            if (call.name === 'edit_text' && oldContent) {
                              // edit_text: do find-and-replace
                              applyTextEdit(oldContent, newContent, reason);
                            } else {
                              // rewrite_document: replace entire document
                              const currentHtml = editor.getHTML();
                              const newHtml = mdParser.render(newContent);
                              enterDiffPreview(currentHtml, newHtml, reason);
                            }
                          }
                        }
                      }
                    }}
                  />
                );
              }
              
              // Legacy: Use tool_context for backward compatibility
              if (!m.tool_context) return null;
              
              const { tool_name, status, reason } = m.tool_context;
              const toolStatus = status === 'starting' ? 'pending' : status;
              
              // Tools with custom UI use ToolCall component
              const customUITools = ['search_web_sources', 'ask_question', 'edit_text', 'rewrite_document'];
              
              if (customUITools.includes(tool_name)) {
                // Web search tool
                if (tool_name === 'search_web_sources') {
                  return <WebSearchSteps key={i} tool_context={m.tool_context as WebSearchToolContext} />;
                }
                
                // Ask question tool - use similar display to web search
                if (tool_name === 'ask_question') {
                  const { answer, citations } = m.tool_context;
                  return (
                    <ToolCall key={i} status={toolStatus === 'running' ? 'running' : 'completed'} defaultOpen={true}>
                      <ToolCallTrigger icon={<FileSearch className="h-4 w-4" />}>
                        Ask question
                      </ToolCallTrigger>
                      <ToolCallContent>
                        {toolStatus === 'running' ? (
                          <ToolCallStatusItem status="running">
                            Finding answer...
                          </ToolCallStatusItem>
                        ) : (
                          <div className="space-y-2">
                            <ToolCallStatusItem status="completed">
                              Answer found with {citations?.length || 0} citations
                            </ToolCallStatusItem>
                            {answer && (
                              <div className="text-sm text-muted-foreground bg-muted/50 rounded-md p-2">
                                {answer}
                              </div>
                            )}
                          </div>
                        )}
                      </ToolCallContent>
                    </ToolCall>
                  );
                }
                
                // Edit/rewrite tools - use DiffArtifact for unified diff display
                const displayName = tool_name === 'edit_text' ? 'Text edit' : 'Document rewrite';
                
                // While running, show a loading state
                if (toolStatus === 'running') {
                  return (
                    <ToolCall key={i} status="running" defaultOpen={true}>
                      <ToolCallTrigger icon={<FileDiff className="h-4 w-4" />}>
                        {displayName}{reason ? `: ${reason}` : ''}
                      </ToolCallTrigger>
                      <ToolCallContent>
                        <ToolCallStatusItem status="running">
                          {tool_name === 'edit_text' ? 'Analyzing text selection...' : 'Analyzing document...'}
                        </ToolCallStatusItem>
                      </ToolCallContent>
                    </ToolCall>
                  );
                }
                
                // Get diff data from tool_context.output - handle both edit_text and rewrite_document formats
                const toolOutput = m.tool_context?.output as { 
                  original_text?: string;  // edit_text format
                  new_text?: string;       // edit_text format
                  original_content?: string; // rewrite_document format
                  new_content?: string;      // rewrite_document format
                } | undefined;
                
                // edit_text uses original_text/new_text, rewrite_document uses original_content/new_content
                const oldText = toolOutput?.original_text || toolOutput?.original_content || '';
                const newText = toolOutput?.new_text || toolOutput?.new_content || '';
                const isEditText = tool_name === 'edit_text';
                
                // Show DiffArtifact with diff preview and Apply button
                return (
                  <DiffArtifact
                    key={i}
                    title={`${displayName}${reason ? `: ${reason}` : ''}`}
                    description={reason}
                    oldText={oldText}
                    newText={newText}
                    onApply={() => {
                      if (!editor || !newText) return;
                      
                      if (isEditText && oldText) {
                        // edit_text: do find-and-replace
                        applyTextEdit(oldText, newText, reason || '');
                      } else {
                        // rewrite_document: replace entire document
                        const currentHtml = editor.getHTML();
                        const newHtml = mdParser.render(newText);
                        enterDiffPreview(currentHtml, newHtml, reason || '');
                      }
                    }}
                  />
                );
              }
              
              // Simple tools use Steps component
              const getToolIcon = () => {
                switch (tool_name) {
                  case 'analyze_document': return <BookOpen className="h-4 w-4" />;
                  case 'get_relevant_sources': return <FileSearch className="h-4 w-4" />;
                  case 'add_context_from_sources': return <PlusCircle className="h-4 w-4" />;
                  case 'generate_text_content': return <FileText className="h-4 w-4" />;
                  case 'generate_image_prompt': return <ImageIcon className="h-4 w-4" />;
                  default: return <Wrench className="h-4 w-4" />;
                }
              };
              
              const displayName = getToolDisplayName(tool_name);
              
              return (
                <Steps key={i} defaultOpen={false}>
                  <StepsTrigger icon={getToolIcon()}>
                    {displayName}
                  </StepsTrigger>
                  <StepsContent>
                    <StepsItem status={toolStatus === 'running' ? 'running' : toolStatus === 'error' ? 'error' : 'completed'}>
                      {toolStatus === 'running' ? 'Processing...' : toolStatus === 'error' ? (m.tool_context.message || 'Failed') : 'Completed'}
                    </StepsItem>
                  </StepsContent>
                </Steps>
              );
            };
            
            switch (m.role) {
              case 'tool': {
                // Legacy tool messages - render as simple Steps
                try {
                  const toolResult = JSON.parse(m.content);
                  const displayName = getToolDisplayName(toolResult.tool_name || 'unknown');
                  
                  return (
                    <Steps key={i} defaultOpen={false}>
                      <StepsTrigger icon={<Wrench className="h-4 w-4" />}>
                        {displayName}
                      </StepsTrigger>
                      <StepsContent>
                        <StepsItem status="completed">
                          Completed
                        </StepsItem>
                      </StepsContent>
                    </Steps>
                  );
                } catch (_e) {
                  return (
                    <Steps key={i} defaultOpen={false}>
                      <StepsTrigger icon={<Wrench className="h-4 w-4" />}>
                        Tool executed
                      </StepsTrigger>
                      <StepsContent>
                        <StepsItem status="completed">
                          Completed
                        </StepsItem>
                      </StepsContent>
                    </Steps>
                  );
                }
              }
              case 'assistant': {
                // NEW: Render tool group messages (new architecture)
                if (m.tool_group) {
                  return renderToolMessage();
                }
                
                // Render tool messages with unified UI (legacy)
                if (m.tool_context) {
                  return renderToolMessage();
                }
                
                // Render artifacts from metadata using unified DiffArtifact component
                if (m.meta_data?.artifact) {
                  const artifact = m.meta_data.artifact;
                  const toolExec = m.meta_data?.tool_execution;
                  const toolOutput = toolExec?.output as { 
                    original_text?: string;  // edit_text format
                    new_text?: string;       // edit_text format
                    original_content?: string; // rewrite_document format
                    new_content?: string;      // rewrite_document format
                  } | undefined;
                  
                  // Get diff data - handle both edit_text and rewrite_document formats
                  const oldText = toolOutput?.original_text || toolOutput?.original_content || '';
                  const newText = toolOutput?.new_text || toolOutput?.new_content || artifact.content || '';
                  const isEditText = toolExec?.tool_name === 'edit_text';
                  
                  return (
                    <DiffArtifact
                      key={i}
                      title={artifact.title || 'Document changes'}
                      description={artifact.description}
                      oldText={oldText}
                      newText={newText}
                      onApply={() => {
                        if (!editor || !newText) return;
                        
                        if (isEditText && oldText) {
                          // edit_text: do find-and-replace
                          applyTextEdit(oldText, newText, artifact.description || '');
                        } else {
                          // rewrite_document: replace entire document
                          const currentHtml = editor.getHTML();
                          const newHtml = mdParser.render(newText);
                          enterDiffPreview(currentHtml, newHtml, artifact.description || '');
                        }
                      }}
                    />
                  );
                }
                
                // Skip __DIFF_ACTIONS__ messages - diff actions are now handled by the sticky DiffActionBar
                if (m.content === '__DIFF_ACTIONS__') {
                  return null;
                }
                  
                // Regular assistant message
                if (!m.content || m.content === '') {
                  return null; // Don't render empty messages
                }
                
                return (
                  <div key={i} className="w-full">
                    <div className="prose prose-sm dark:prose-invert max-w-none text-sm">
                      <Markdown>{m.content}</Markdown>
                    </div>
                  </div>
                );
              }
              case 'user':
                default: {
                  return (
                    <div key={i} className="w-full flex justify-end">
                      <div className="max-w-xs whitespace-pre-wrap rounded-lg px-3 py-2 text-sm bg-indigo-500 text-white">
                        {m.content}
                      </div>
                    </div>
                  );
                }
              }
            })}
            
            {/* Thinking state shimmer */}
            {isThinking && (
              <ThinkShimmerBlock message={thinkingMessage} />
            )}
          </div>
          
          {/* Sticky diff action bar - shows when there are pending changes */}
          {diffing && !chatLoading && (
            <DiffActionBar 
              onKeepAll={acceptDiff}
              onReject={rejectDiff}
            />
          )}
          
        <div className="p-4 border-t space-y-2">
          <div className="flex gap-1 flex-wrap">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                const message = 'Please rewrite this article to make it more engaging and clear';
                setChatInput(message);
                sendChatWithMessage(message);
              }}
              disabled={chatLoading}
            >
              ✨ Improve
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                const message = 'Fix any grammar and spelling issues in this article';
                setChatInput(message);
                sendChatWithMessage(message);
              }}
              disabled={chatLoading}
            >
              ✓ Fix Grammar
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                const message = 'Make this article shorter and more concise';
                setChatInput(message);
                sendChatWithMessage(message);
              }}
              disabled={chatLoading}
            >
              📝 Shorten
            </Button>
          </div>
          <PromptInput
            value={chatInput}
            onValueChange={setChatInput}
            onSubmit={sendChat}
            isLoading={chatLoading}
            className="flex-1"
          >
            <PromptInputTextarea
              placeholder="Ask the assistant or click a quick action above…"
              className="min-h-[60px]"
            />
            <PromptInputActions className="justify-end pt-2">
              <PromptInputAction
                tooltip={chatLoading ? "Stop generation" : "Send message"}
              >
                <Button
                  variant="default"
                  size="icon"
                  className="h-8 w-8 rounded-full"
                  disabled={!chatLoading && !chatInput.trim()}
                  onClick={sendChat}
                >
                  {chatLoading ? (
                    <Square className="size-5 fill-current" />
                  ) : (
                    <ArrowUp className="size-5" />
                  )}
                </Button>
              </PromptInputAction>
            </PromptInputActions>
          </PromptInput>
        </div>
      </div>

      {/* Sources Manager */}
      {article && (
        <SourcesManager
          articleId={article.article.id}
          isOpen={sourcesManagerOpen}
          onOpenChange={setSourcesManagerOpen}
        />
      )}
    </section>
  );
}
