import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from '@tanstack/react-router';
import { useAuth } from '@/services/auth/auth';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { format } from "date-fns"
import { Calendar as CalendarIcon, PencilIcon, SparklesIcon, RefreshCw, Bold, Italic, Strikethrough, Code, Heading1, Heading2, Heading3, List, ListOrdered, Quote, Undo, Redo, ChevronDown as ChevronDownIcon, ChevronUp as ChevronUpIcon } from "lucide-react"
import { ExternalLinkIcon, UploadIcon } from '@radix-ui/react-icons';
import { IconLoader2 } from '@tabler/icons-react';
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
import { ChipInput } from "@/components/ui/chip-input";
import { Calendar } from "@/components/ui/calendar"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { useToast } from "@/hooks/use-toast";
import { ThinkShimmerBlock } from "@/components/ui/think-shimmer";
import { getConversationHistory } from "@/services/conversations";
import { acceptArtifact, rejectArtifact } from "@/services/artifacts";
import { WebSearchSteps } from "./WebSearchSteps";
import { cn } from '@/lib/utils';
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
    context?: any;
    user_action?: any;
  };
  tool_context?: {
    tool_name: string;
    tool_id: string;
    status: 'starting' | 'running' | 'completed' | 'error';
    search_query?: string;
    search_results?: SearchResult[];
    sources_created?: SourceInfo[];
    total_found?: number;
    sources_successful?: number;
    message?: string;
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

// Diff preview component for chat messages
function DiffPreview({ diffPreview, diffState }: { 
  diffPreview: { oldText: string; newText: string; reason?: string }, 
  diffState: 'accepted' | 'rejected' 
}) {
  const [isExpanded, setIsExpanded] = useState(false);
  const parts = diffWords(diffPreview.oldText, diffPreview.newText);
  const maxLines = 3; // Show first 3 lines by default
  
  type DiffPart = { added?: boolean; removed?: boolean; value: string; truncated?: boolean };
  
  // Split parts into lines to determine if we need truncation
  let currentLine = 0;
  let truncatedParts: DiffPart[] = [];
  let hasMoreContent = false;
  
  for (let i = 0; i < parts.length; i++) {
    const part = parts[i];
    const lines = part.value.split('\n');
    
    if (currentLine + lines.length <= maxLines || isExpanded) {
      // Include the whole part
      truncatedParts.push(part);
      currentLine += lines.length - 1; // -1 because last line doesn't count as a new line
    } else {
      // Truncate this part
      const remainingLines = maxLines - currentLine;
      if (remainingLines > 0) {
        const truncatedValue = lines.slice(0, remainingLines).join('\n');
        truncatedParts.push({
          ...part,
          value: truncatedValue,
          truncated: true
        });
      }
      hasMoreContent = true;
      break;
    }
  }
  
  // If we're not expanded and there's more content, show the expand button
  const showExpandButton = !isExpanded && (hasMoreContent || parts.some(p => p.value.split('\n').length > maxLines));
  
  return (
    <div className={cn(
      "mt-2 p-0 rounded-md text-xs"
    )}>
      <div className={cn(
        "flex flex-col items-start gap-1 mb-1 font-medium",
        diffState === 'accepted' ? "text-green-700 dark:text-green-300" : "text-red-700 dark:text-red-300"
      )}>
        <div className="flex flex-row items-center gap-1">
          <span>{diffState === 'accepted' ? '✅' : '❌'}</span>
          <span>{diffState === 'accepted' ? 'Accepted' : 'Rejected'}</span>
        </div>
        {diffPreview.reason && <span>• {diffPreview.reason}</span>}
      </div>
      <div className="font-mono text-xs whitespace-pre-wrap">
        {(isExpanded ? parts : truncatedParts).map((part, index) => (
          <span
            key={index}
            className={cn(
              part.added ? "bg-green-200 dark:bg-green-800 text-green-900 dark:text-green-100" : 
              part.removed ? "bg-red-200 dark:bg-red-800 text-red-900 dark:text-red-100 line-through" : 
              "text-gray-700 dark:text-gray-300"
            )}
          >
            {part.value}
            {'truncated' in part && part.truncated && <span className="text-gray-500">...</span>}
          </span>
        ))}
      </div>
      {/* Show expand/collapse button at bottom */}
      {(showExpandButton || isExpanded) && (
        <div className="flex justify-center mt-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setIsExpanded(!isExpanded)}
            className="h-6 px-2 text-xs text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            {isExpanded ? (
              <>
                <ChevronUpIcon className="h-3 w-3 mr-1" />
                Show less
              </>
            ) : (
              <>
                <ChevronDownIcon className="h-3 w-3 mr-1" />
                Show more
              </>
            )}
          </Button>
        </div>
      )}
    </div>
  );
}

// Formatting toolbar component
function FormattingToolbar({ editor }: { editor: TiptapEditor | null }) {
  if (!editor) {
    return null;
  }

  return (
    <div className="flex flex-wrap gap-1 p-2 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 rounded-t-md">
      {/* Text formatting */}
      <Button
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleBold().run()}
        className={editor.isActive('bold') ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Bold className="w-4 h-4" />
      </Button>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleItalic().run()}
        className={editor.isActive('italic') ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Italic className="w-4 h-4" />
      </Button>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleStrike().run()}
        className={editor.isActive('strike') ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Strikethrough className="w-4 h-4" />
      </Button>
      <Button
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
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
        className={editor.isActive('heading', { level: 1 }) ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Heading1 className="w-4 h-4" />
      </Button>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
        className={editor.isActive('heading', { level: 2 }) ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <Heading2 className="w-4 h-4" />
      </Button>
      <Button
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
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().toggleBulletList().run()}
        className={editor.isActive('bulletList') ? 'bg-gray-200 dark:bg-gray-700' : ''}
      >
        <List className="w-4 h-4" />
      </Button>
      <Button
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
        variant="ghost"
        size="sm"
        onClick={() => editor.chain().focus().undo().run()}
        disabled={!editor.can().undo()}
      >
        <Undo className="w-4 h-4" />
      </Button>
      <Button
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
  const chatInputRef = useRef<HTMLTextAreaElement>(null);
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
    onSuccess: (_, variables) => {
      toast({ title: "Success", description: "Article updated successfully." });
      queryClient.invalidateQueries({ queryKey: ['article', blogSlug] });
      queryClient.invalidateQueries({ queryKey: ['articles'] });
      
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

  const { register, handleSubmit, setValue, formState: { errors }, watch, reset } = useForm<ArticleFormData>({
    resolver: zodResolver(articleSchema),
    defaultValues: {
      title: '',
      content: '',
      image_url: '',
      tags: [],
      isDraft: false,
    }
  });

  // Watch form values to ensure UI reflects current state
  const watchedValues = watch();

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
  const [activeDiffMessageIndex, setActiveDiffMessageIndex] = useState<number | null>(null);
  

  
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
    
    // Save diff preview data before clearing - use full HTML content
    const oldText = originalDocument;
    const newText = pendingNewDocument;
    
    // Update the __DIFF_ACTIONS__ message with accepted state
    if (activeDiffMessageIndex !== null) {
      setChatMessages((prev) => {
        const updated = [...prev];
        if (updated[activeDiffMessageIndex]?.content === '__DIFF_ACTIONS__') {
          updated[activeDiffMessageIndex] = {
            ...updated[activeDiffMessageIndex],
            diffState: 'accepted',
            diffPreview: {
              oldText,
              newText,
              reason: currentDiffReason,
            },
          };
        }
        return updated;
      });
    }
    
    editor.commands.setContent(pendingNewDocument || editor.getHTML());
    setValue('content', editor.getHTML());
    clearDiffDecorations();
    setDiffing(false);
    setPendingNewDocument('');
    setCurrentDiffReason('');
    setActiveDiffMessageIndex(null);
  };

  const rejectDiff = () => {
    if (!editor) return;
    
    // Save diff preview data before clearing - use full HTML content
    const oldText = originalDocument;
    const newText = pendingNewDocument;
    
    // Update the __DIFF_ACTIONS__ message with rejected state
    if (activeDiffMessageIndex !== null) {
      setChatMessages((prev) => {
        const updated = [...prev];
        if (updated[activeDiffMessageIndex]?.content === '__DIFF_ACTIONS__') {
          updated[activeDiffMessageIndex] = {
            ...updated[activeDiffMessageIndex],
            diffState: 'rejected',
            diffPreview: {
              oldText,
              newText,
              reason: currentDiffReason,
            },
          };
        }
        return updated;
      });
    }
    
    editor.commands.setContent(originalDocument || editor.getHTML());
    setValue('content', originalDocument || editor.getHTML());
    clearDiffDecorations();
    setDiffing(false);
    setPendingNewDocument('');
    setCurrentDiffReason('');
    setActiveDiffMessageIndex(null);
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
    content: watchedValues.content || '', // Content is already HTML
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

  // When form values are externally reset (e.g., after fetching the article or clearing for a new one) we
  // synchronise those changes into the editor exactly once.
  useEffect(() => {
    if (!editor) return;
    // If the change came from user typing inside the editor, `editor.getHTML()` already matches `watchedValues.content`.
    // We only want to update when the two differ – i.e., an external change.
    if (watch('content') !== editor.getHTML()) {
      // Load content directly as HTML since we're now saving as HTML
      editor.commands.setContent(watch('content') || '');
    }
  }, [watch('content'), editor]);

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
      console.log("Populating form with article data:", article);
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
      console.log("Resetting form for new article");
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
  }, [article, isNew, reset]);

  // Load conversation history with artifacts when article is loaded
  useEffect(() => {
    if (article?.article?.id && !isNew) {
      console.log('[Editor] 🔄 Loading conversation history for article:', article.article.id);
      
      const loadConversations = async () => {
        try {
          const result = await getConversationHistory(article.article.id);
          console.log('[Editor] 📦 API response:', result);
          console.log('[Editor] 📊 Message count from API:', result.messages?.length || 0);
          
          if (!result.messages || result.messages.length === 0) {
            console.log('[Editor] ⚠️  No messages in API response, showing greeting');
            setChatMessages([
              { role: 'assistant', content: 'Hi! I can help you improve your article. Try asking me to "rewrite the introduction" or "make the content more engaging".' }
            ]);
            return;
          }
          
          // Convert database messages to chat messages with artifact metadata and tool_context
          const loadedMessages = result.messages.map((msg: any, idx: number) => {
            console.log(`[Editor] 🔄 Transforming message ${idx + 1}:`, {
              id: msg.id,
              role: msg.role,
              contentPreview: msg.content?.substring(0, 50),
              hasMetadata: !!msg.meta_data,
              hasArtifact: !!msg.meta_data?.artifact,
              hasToolExecution: !!msg.meta_data?.tool_execution
            });
            
            const chatMsg: ChatMessage = {
              id: msg.id,
              role: msg.role,
              content: msg.content,
              meta_data: msg.meta_data,
              created_at: msg.created_at,
            };
            
            // Reconstruct tool_context from metadata for search tools
            if (msg.meta_data?.tool_execution?.tool_name === 'search_web_sources') {
              const output = msg.meta_data.tool_execution.output;
              console.log(`[Editor]    🔍 Reconstructing search tool_context from metadata`);
              chatMsg.tool_context = {
                tool_name: 'search_web_sources',
                tool_id: msg.meta_data.tool_execution.tool_id || '',
                status: 'completed',
                search_query: output?.query || '',
                search_results: output?.search_results || [],
                sources_created: output?.sources_created || [],
                total_found: output?.total_found || 0,
                sources_successful: output?.sources_successful || 0,
                message: output?.message
              };
            }
            
            return chatMsg;
          }) as ChatMessage[];
          
          console.log('[Editor] ✅ Transformed', loadedMessages.length, 'messages');
          console.log('[Editor] 📋 Transformed messages:', loadedMessages);
          console.log('[Editor] 🎯 Setting chatMessages state...');
          
          setChatMessages(loadedMessages);
          
          // Force a small delay to ensure state update completes
          setTimeout(() => {
            console.log('[Editor] ✅ chatMessages state should be updated now');
          }, 100);
          
        } catch (error) {
          console.error('[Editor] ❌ Failed to load conversation history:', error);
          // Show greeting on error
          setChatMessages([
            { role: 'assistant', content: 'Hi! I can help you improve your article. Try asking me to "rewrite the introduction" or "make the content more engaging".' }
          ]);
        }
      };
      
      loadConversations();
    } else if (isNew) {
      console.log('[Editor] 📝 New article - showing greeting only');
      setChatMessages([
        { role: 'assistant', content: 'Hi! I can help you improve your article. Try asking me to "rewrite the introduction" or "make the content more engaging".' }
      ]);
    }
  }, [article?.article?.id, isNew]);

  // Debug: Log current form values
  useEffect(() => {
    console.log("Current form values:", watchedValues);
  }, [watchedValues]);

  // Auto-scroll chat to bottom when messages change
  useEffect(() => {
    console.log('[Editor] 🔔 chatMessages state changed!');
    console.log('[Editor]    Count:', chatMessages.length);
    console.log('[Editor]    Messages:', chatMessages);
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
      
      console.log('=== ARTICLE UPDATE DATA ===');
      console.log('Blog Slug:', blogSlug);
      console.log('Update Data:', updateData);
      console.log('Staged Image URL:', stagedImageUrl);
      console.log('Final Image URL:', finalImageUrl);
      console.log('==========================');
      
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
        setChatMessages((prev) => {
          const newMessages: ChatMessage[] = [
            ...prev,
            { role: 'assistant', content: '📋 Proposed full-document changes' },
            { role: 'assistant', content: '__DIFF_ACTIONS__' },
          ];
          // Set the active diff message index (last message)
          setActiveDiffMessageIndex(newMessages.length - 1);
          return newMessages;
        });
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
    
    console.log('Applying text edit:', { originalText, newText, reason });
    
    const oldHtml = editor.getHTML();
    const currentText = editor.getText();
    const index = currentText.indexOf(originalText);
    if (index === -1) {
      console.warn('Original text not found in document:', originalText);
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
    setChatMessages((prev) => {
      const newMessages: ChatMessage[] = [
        ...prev,
        { role: 'assistant', content: `📋 Proposed changes: ${reason}` },
        { role: 'assistant', content: '__DIFF_ACTIONS__' },
      ];
      // Set the active diff message index (last message)
      setActiveDiffMessageIndex(newMessages.length - 1);
      return newMessages;
    });
  };

  // Apply patch from AI assistant (new implementation)
  const applyPatch = (patch: any, originalText: string, newText: string, reason: string) => {
    if (!editor) return;
    
    console.log('Applying patch:', { patch, originalText, newText, reason });
    
    const oldHtml = editor.getHTML();
    const currentText = editor.getText();
    const index = currentText.indexOf(originalText);
    if (index === -1) {
      console.warn('Original text not found in document for patch:', originalText);
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
    setChatMessages((prev) => {
      const newMessages: ChatMessage[] = [
        ...prev,
        { role: 'assistant', content: `📋 Proposed changes: ${reason}` },
        { role: 'assistant', content: '__DIFF_ACTIONS__' },
      ];
      // Set the active diff message index (last message)
      setActiveDiffMessageIndex(newMessages.length - 1);
      return newMessages;
    });
  };

  // Apply document rewrite from AI assistant
  const applyDocumentRewrite = (newContent: string, reason: string, originalContent?: string) => {
    if (!editor) return;
    
    console.log('Applying document rewrite:', { newContent, reason, originalContent });
    
    const oldHtml = editor.getHTML();
    
    // Convert markdown to HTML for the new content
    const newHtml = mdParser.render(newContent);
    
    // Create diff preview
    enterDiffPreview(oldHtml, newHtml, reason);
    setChatMessages((prev) => {
      const newMessages: ChatMessage[] = [
        ...prev,
        { role: 'assistant', content: `📋 Document rewrite: ${reason}` },
        { role: 'assistant', content: '__DIFF_ACTIONS__' },
      ];
      // Set the active diff message index (last message)
      setActiveDiffMessageIndex(newMessages.length - 1);
      return newMessages;
    });
  };

  const sendChatWithMessage = async (message: string) => {
    const text = message.trim();
    console.log("Sending chat with message:", text);
    
    if (!text) {
      console.log("No text to send");
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

    // Create messages for API - only send user messages, not assistant responses
    // This prevents the backend from streaming back previous assistant messages as context
    const userMessages = chatMessages.filter(msg => msg.role === 'user');
    const apiMessages = [...userMessages, { role: 'user', content: text } as ChatMessage];

    // Rest of the chat logic...
    await performChatRequest(apiMessages, assistantIndex, isEditRequest, currentContent);
  };

  const sendChat = async () => {
    console.log("Sending chat");
    const text = chatInput.trim();
    console.log("Chat Text:", text);
    console.log("Chat Input State:", chatInput);
    console.log("Ref current value:", chatInputRef.current?.value);
    
    if (!text) {
      console.log("No text to send");
      return;
    }

    await sendChatWithMessage(text);
  };

  const performChatRequest = async (apiMessages: ChatMessage[], assistantIndex: number, isEditRequest: boolean, documentContent: string) => {
    setChatLoading(true);
    try {
      console.log('Sending chat request:', { 
        messages: apiMessages.map(m => ({ role: m.role, content: m.content.substring(0, 100) + (m.content.length > 100 ? '...' : '') })),
        documentContent: documentContent ? `${documentContent.substring(0, 100)}...` : 'none',
        articleId: article?.article?.id || 'not available'
      });
      
      // Submit the request and get immediate response with request ID
      const result = await apiPost<{ requestId: string; status: string }>('/agent', {
        messages: apiMessages,
        documentContent: documentContent,
        articleId: article?.article?.id || null  // Include article ID for source search
      });

      console.log('Got request response:', result);
      
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
      console.log('Connecting to WebSocket:', wsUrl);
      
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('WebSocket connected, subscribing to request:', requestId);
        ws.send(JSON.stringify({
          action: 'subscribe',
          requestId: requestId
        }));
      };

      let currentAssistantContent = '';
      let hasInitialContent = false;

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          console.log('WebSocket message:', msg);
          
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
                console.log('Thinking:', msg.thinking_message);
                break;

              case 'user':
                // Display user message blocks (usually shown as context)
                console.log('User message block:', msg.content);
                break;
                
              case 'system':
                // Display system message blocks (usually shown as context)
                console.log('System message block:', msg.content);
                break;
                
              case 'text':
                // Hide thinking state on first text chunk
                setIsThinking(false);
                // Handle assistant text responses as separate messages
                if (msg.content) {
                  // If this is the first text block, update the existing assistant message
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
                
                // Display tool usage feedback as a separate message
                if (msg.tool_name) {
                  // Special handling for search_web_sources
                  if (msg.tool_name === 'search_web_sources') {
                    const searchQuery = msg.tool_input?.query || 'researching...';
                    setChatMessages((prev) => [
                      ...prev,
                      { 
                        role: 'assistant', 
                        content: '',
                        tool_context: {
                          tool_name: msg.tool_name,
                          tool_id: msg.tool_id || '',
                          status: 'starting',
                          search_query: searchQuery
                        }
                      }
                    ]);
                  } else {
                    // Regular tool display
                    const toolDisplayName = getToolDisplayName(msg.tool_name);
                    const toolMessage = `🔧 ${toolDisplayName}...`;
                    
                    setChatMessages((prev) => [
                      ...prev,
                      { role: 'assistant', content: toolMessage }
                    ]);
                  }
                  
                  console.log('Tool use:', msg.tool_name, msg.tool_input);
                }
                break;
                
              case 'tool_result':
                // Hide thinking state on tool result
                setIsThinking(false);
                
                // Handle tool results with structured data
                if (msg.tool_result) {
                  console.log('Tool result:', msg.tool_result);
                  
                  // Create a unique identifier for this tool message
                  const toolMessageId = `${requestId}-${Date.now()}-tool-result`;
                  const isNewMessage = !processedToolMessages.has(toolMessageId);
                  
                  try {
                    // Check if this is an edit_text tool result
                    if (msg.tool_result.content) {
                      const toolResult = JSON.parse(msg.tool_result.content);
                      
                      // Check if this is a search tool result
                      if (toolResult.tool_name === 'search_web_sources') {
                        // Find the matching tool_use message and update it with results
                        setChatMessages((prev) => {
                          const updated = [...prev];
                          // Find last search_web_sources tool_use message
                          for (let i = updated.length - 1; i >= 0; i--) {
                            if (updated[i].tool_context?.tool_name === 'search_web_sources' 
                                && updated[i].tool_context?.status === 'starting') {
                              updated[i] = {
                                ...updated[i],
                                tool_context: {
                                  ...updated[i].tool_context!,
                                  status: 'completed',
                                  search_results: toolResult.search_results || [],
                                  sources_created: toolResult.sources_created || [],
                                  total_found: toolResult.total_found || 0,
                                  sources_successful: toolResult.sources_successful || 0,
                                  message: toolResult.message
                                }
                              };
                              break;
                            }
                          }
                          return updated;
                        });
                      } else if (toolResult.tool_name === 'edit_text' && isNewMessage) {
                        // Use inline diff preview and chat actions
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
                      } else {
                        // For non-edit tools, add generic completion message
                        setChatMessages((prev) => [
                          ...prev,
                          { role: 'assistant', content: '✅ Task completed' }
                        ]);
                      }
                    }
                    
                    // Add the tool result to chat history (but don't display it visually)
                    // This keeps the technical result available for debugging
                    console.log('Tool result data:', JSON.stringify(msg.tool_result));
                    
                  } catch (error) {
                    console.error('Error parsing tool result:', error);
                    // Add error message
                    setChatMessages((prev) => [
                      ...prev,
                      { role: 'assistant', content: '⚠️ Tool execution completed with warnings' }
                    ]);
                  }
                }
                break;
                
              case 'done':
                console.log('Stream completed');
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
                      setChatMessages((prev) => {
                        const newMessages: ChatMessage[] = [
                          ...prev,
                          { role: 'assistant', content: '📋 Proposed full-document changes' },
                          { role: 'assistant', content: '__DIFF_ACTIONS__' },
                        ];
                        // Set the active diff message index (last message)
                        setActiveDiffMessageIndex(newMessages.length - 1);
                        return newMessages;
                      });
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
            console.log('Legacy tool message:', msg);
            
            // Create a unique identifier for this tool message
            const toolMessageId = `${requestId}-${Date.now()}-${msg.content.slice(0, 50)}`;
            const isNewMessage = !processedToolMessages.has(toolMessageId);
            
            try {
              // Parse the tool result content to extract artifacts
              const toolResult = JSON.parse(msg.content);
              console.log('Parsed tool result:', toolResult);
              
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
              
              // Add the tool message to chat history (for debugging, not displayed)
              console.log('Legacy tool message data:', msg.content);
              
            } catch (error) {
              console.error('Error parsing tool result:', error);
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
            console.log('Stream completed');
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
                  setChatMessages((prev) => {
                    const newMessages: ChatMessage[] = [
                      ...prev,
                      { role: 'assistant', content: '📋 Proposed full-document changes' },
                      { role: 'assistant', content: '__DIFF_ACTIONS__' },
                    ];
                    // Set the active diff message index (last message)
                    setActiveDiffMessageIndex(newMessages.length - 1);
                    return newMessages;
                  });
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
        console.log('WebSocket closed:', event.code, event.reason);
        if (event.code !== 1000) { // 1000 is normal closure
          console.error('WebSocket closed unexpectedly');
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
                    value={watchedValues.title}
                    onChange={(e) => setValue('title', e.target.value)}
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
                          {watchedValues.tags && watchedValues.tags.length > 0 ? (
                            <div className="flex flex-wrap gap-1">
                              {watchedValues.tags.slice(0, 3).map((tag, idx) => (
                                <span
                                  key={idx}
                                  className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200"
                                >
                                  {tag}
                                </span>
                              ))}
                              {watchedValues.tags.length > 3 && (
                                <span className="text-xs text-muted-foreground">
                                  +{watchedValues.tags.length - 3} more
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
                          {watchedValues.tags?.length || 0} tag{(watchedValues.tags?.length || 0) !== 1 ? 's' : ''}
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
                          value={watchedValues.tags}
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
                            <span className={cn("text-xs font-medium", watchedValues.isDraft ? "text-orange-600" : "text-green-600")}>
                              {watchedValues.isDraft ? "Draft" : "Published"}
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
                            <span className={cn("text-sm", watchedValues.isDraft ? "text-muted-foreground" : "text-green-600")}>
                              {watchedValues.isDraft ? "Draft" : "Published"}
                            </span>
                            <Switch
                              id="isDraft"
                              checked={!watchedValues.isDraft}
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
                {/* Hidden input to keep react-hook-form registration for content */}
                <input type="hidden" {...register('content')} value={watchedValues.content} />
                {errors.content && <p className="text-red-500">{errors.content.message}</p>}
                {/* Diff preview is inline; accept/decline in chat */}
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
                      // Discard diff changes
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
                  // Discard diff changes
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
            switch (m.role) {
              case 'tool': {
                try {
                  const toolResult = JSON.parse(m.content);
                  const label = toolResult.tool_name === 'edit_text'
                    ? `${toolResult.edit_type === 'patch' ? 'Patch generated' : 'Text edit proposed'}: ${toolResult.reason}`
                    : 'Tool executed';
                  return (
                    <div key={i} className="w-full flex justify-center">
                      <div className="max-w-xs bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg px-3 py-2 text-xs">
                        <div className="flex items-center gap-2">
                          <span className="text-blue-600 dark:text-blue-400">🔧</span>
                          <span className="text-blue-700 dark:text-blue-300 truncate">{label}</span>
                        </div>
                      </div>
                    </div>
                  );
                } catch (_e) {
                  return (
                    <div key={i} className="w-full flex justify-center">
                      <div className="max-w-xs bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg px-3 py-2 text-xs">
                        <div className="flex items-center gap-2">
                          <span className="text-gray-600 dark:text-gray-400">⚙️</span>
                          <span className="text-gray-700 dark:text-gray-300 truncate">Tool executed</span>
                        </div>
                      </div>
                    </div>
                  );
                }
              }
              case 'assistant': {
                // Render web search with Steps UI
                if (m.tool_context?.tool_name === 'search_web_sources') {
                  return <WebSearchSteps key={i} tool_context={m.tool_context} />;
                }
                
                // Render artifacts from metadata
                if (m.meta_data?.artifact) {
                  const artifact = m.meta_data.artifact;
                  
                  // Show artifact based on status
                  if (artifact.status === 'pending') {
                    return (
                      <div key={i} className="w-full flex justify-start">
                        <div className="max-w-sm rounded-lg px-3 py-2 text-sm bg-gray-100 dark:bg-gray-800">
                          <div className="mb-2">
                            <div className="font-medium text-sm">{artifact.title || 'Artifact'}</div>
                            {artifact.description && (
                              <div className="text-xs text-gray-600 dark:text-gray-400">{artifact.description}</div>
                            )}
                          </div>
                          <div className="flex gap-2">
                            <Button 
                              size="sm" 
                              onClick={async () => {
                                if (m.id) {
                                  await acceptArtifact(m.id);
                                  // Update local state
                                  setChatMessages(prev => prev.map((msg, idx) => 
                                    idx === i && msg.meta_data?.artifact 
                                      ? { ...msg, meta_data: { ...msg.meta_data, artifact: { ...msg.meta_data.artifact, status: 'accepted' } } }
                                      : msg
                                  ));
                                }
                              }}
                            >
                              Accept
                            </Button>
                            <Button 
                              size="sm" 
                              variant="outline" 
                              onClick={async () => {
                                if (m.id) {
                                  await rejectArtifact(m.id);
                                  // Update local state
                                  setChatMessages(prev => prev.map((msg, idx) => 
                                    idx === i && msg.meta_data?.artifact 
                                      ? { ...msg, meta_data: { ...msg.meta_data, artifact: { ...msg.meta_data.artifact, status: 'rejected' } } }
                                      : msg
                                  ));
                                }
                              }}
                            >
                              Reject
                            </Button>
                          </div>
                        </div>
                      </div>
                    );
                  } else if (artifact.status === 'accepted') {
                    return (
                      <div key={i} className="w-full flex justify-start">
                        <div className="max-w-sm rounded-lg px-3 py-2 text-sm bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800">
                          <div className="flex items-center gap-2 text-green-700 dark:text-green-300">
                            <span>✅</span>
                            <div>
                              <div className="font-medium">{artifact.title || 'Artifact'} - Accepted</div>
                              {artifact.description && (
                                <div className="text-xs">{artifact.description}</div>
                              )}
                            </div>
                          </div>
                        </div>
                      </div>
                    );
                  } else if (artifact.status === 'rejected') {
                    return (
                      <div key={i} className="w-full flex justify-start">
                        <div className="max-w-sm rounded-lg px-3 py-2 text-sm bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
                          <div className="flex items-center gap-2 text-red-700 dark:text-red-300">
                            <span>❌</span>
                            <div>
                              <div className="font-medium">{artifact.title || 'Artifact'} - Rejected</div>
                              {artifact.description && (
                                <div className="text-xs">{artifact.description}</div>
                              )}
                            </div>
                          </div>
                        </div>
                      </div>
                    );
                  }
                }
                
                if (m.content === '__DIFF_ACTIONS__') {
                  // Show diff preview if the action has been taken, otherwise show buttons
                  if (m.diffState && m.diffPreview) {
                    return (
                      <div key={i} className="w-full flex justify-start">
                        <div className="max-w-sm rounded-lg px-3 py-2 text-sm bg-gray-100 dark:bg-gray-800">
                          <DiffPreview diffPreview={m.diffPreview} diffState={m.diffState} />
                        </div>
                      </div>
                    );
                  } else {
                    // Show action buttons
                    return (
                      <div key={i} className="w-full flex justify-start">
                        <div className="max-w-xs rounded-lg px-3 py-2 text-sm bg-gray-100 dark:bg-gray-800">
                          <div className="flex gap-2">
                            <Button size="sm" onClick={acceptDiff} disabled={!diffing}>Accept</Button>
                            <Button size="sm" variant="outline" onClick={rejectDiff} disabled={!diffing}>Discard</Button>
                          </div>
                        </div>
                      </div>
                    );
                  }
                }
                  
                  // Regular assistant message
                  return (
                    <div key={i} className="w-full flex justify-start">
                      <div className="max-w-xs whitespace-pre-wrap rounded-lg px-3 py-2 text-sm bg-gray-200 dark:bg-gray-700 dark:text-white">
                        {m.content || (chatLoading && i === chatMessages.length - 1 ? (
                          <div className="flex items-center gap-1">
                            <div className="flex space-x-1">
                              <IconLoader2 className="w-4 h-4 text-indigo-500 animate-spin" />
                            </div>
                            <span className="text-xs opacity-75">thinking...</span>
                          </div>
                        ) : m.content)}
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
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                const message = 'Fix any typos and improve the first paragraph to be more engaging';
                setChatInput(message);
                sendChatWithMessage(message);
              }}
              disabled={chatLoading}
            >
              ✏️ Edit Text
            </Button>
          </div>
          <div className="flex gap-2">
            <Textarea
              ref={chatInputRef}
              value={chatInput}
              onChange={(e) => setChatInput(e.target.value)}
              rows={2}
              placeholder="Ask the assistant or click a quick action above…"
              className="flex-1 resize-none"
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  sendChat();
                }
              }}
            />
            <Button onClick={sendChat} disabled={chatLoading}>
              {chatLoading ? '…' : 'Send'}
            </Button>
          </div>
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
