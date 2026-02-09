import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from '@tanstack/react-router';
import { useAuth } from '@/services/auth/auth';
import { useForm, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { format } from "date-fns"
import { Calendar as CalendarIcon, PencilIcon, SparklesIcon, RefreshCw, ArrowUp, Square, Settings, Trash2 } from "lucide-react"
import { ExternalLinkIcon, UploadIcon } from '@radix-ui/react-icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useEditor, EditorContent, type Editor as TiptapEditor } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import CodeBlock from '@tiptap/extension-code-block';
import MarkdownIt from 'markdown-it';
import { VITE_API_BASE_URL } from "@/services/constants";
import { apiPost, isAuthError } from '@/services/authenticatedFetch';
import '@/tiptap.css';

// Extracted editor modules
import { DiffHighlighter } from './editor/diff-highlighter';
import { FormattingToolbar } from './editor/FormattingToolbar';
import { ImageLoader } from './editor/ImageLoader';
import { turndownService } from './editor/turndown';
import { 
  DEFAULT_IMAGE_PROMPT, 
  articleSchema, 
  getToolDisplayName,
  type ArticleFormData, 
  type ChatMessage, 
  type SearchResult, 
  type SourceInfo 
} from './editor/editor-types';

// Card removed - no longer needed after toolbar simplification
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
import { getConversationHistory, clearConversationHistory } from "@/services/conversations";
// Artifact accept/reject is now handled by the sticky DiffActionBar
import { WebSearchSteps, WebSearchToolContext } from "./WebSearchSteps";
import { DiffActionBar } from "./DiffActionBar";
import { DiffArtifact } from "./DiffArtifact";
import { ToolGroupDisplay } from "./ToolGroupDisplay";
import type { 
  ToolGroup, 
  ThinkingBlock, 
  TurnStep,
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
import { 
  ChainOfThought,
  ChainOfThoughtStep,
  ChainOfThoughtTrigger,
  ChainOfThoughtContent,
  ChainOfThoughtItem,
  ReasoningStep 
} from "@/components/prompt-kit/chain-of-thought";
import { cn } from '@/lib/utils';
import { FileDiff, Wrench, BookOpen, FileSearch, PlusCircle, FileText, ImageIcon } from "lucide-react";
import { 
  updateArticle, 
  getArticle, 
  createArticle,
  generateArticleImage,
  getImageGeneration,
  getImageGenerationStatus,
  updateArticleWithContext,
  publishArticle,
  unpublishArticle,
  listArticleVersions,
  revertToVersion
} from '@/services/blog';
import { Link } from '@tanstack/react-router';
import { ArticleListItem, ArticleVersion, ArticleVersionListResponse, isPublished, hasDraftChanges } from '@/services/types';
import { Badge } from '@/components/ui/badge';
import { Globe, EyeOff, History, Tag } from 'lucide-react';
import { Dialog, DialogTitle, DialogContent, DialogTrigger, DialogDescription, DialogFooter, DialogHeader, DialogClose } from '@/components/ui/dialog';
import { Drawer, DrawerTrigger, DrawerContent, DrawerHeader, DrawerTitle, DrawerDescription, DrawerFooter, DrawerClose } from '@/components/ui/drawer';
import { SourcesManager } from './SourcesManager';
import { SourcesPreview } from './SourcesPreview';


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
  const [showSettingsDrawer, setShowSettingsDrawer] = useState(false);
  const [clearingChat, setClearingChat] = useState(false);
  const [expandedTable, setExpandedTable] = useState<React.ReactNode | null>(null);
  
  // Custom markdown components for chat with smaller text (14px)
  const chatMarkdownComponents = {
    code: function ChatCodeComponent({ className, children, ...props }: any) {
      const isInline =
        !props.node?.position?.start.line ||
        props.node?.position?.start.line === props.node?.position?.end.line;
      if (isInline) {
        return (
          <code className="bg-muted rounded-sm px-1 font-mono text-sm" {...props}>
            {children}
          </code>
        );
      }
      // Block code - render with smaller text
      return (
        <pre className="bg-muted rounded-md p-2 overflow-x-auto text-sm">
          <code className={className} {...props}>{children}</code>
        </pre>
      );
    },
    pre: function ChatPreComponent({ children }: any) {
      return <>{children}</>;
    },
    table: function ChatTableComponent({ children, ...props }: any) {
      return (
        <div className="relative my-2">
          <div className="overflow-x-auto max-h-48 border rounded-md">
            <table className="min-w-full text-sm" {...props}>
              {children}
            </table>
          </div>
          <button
            type="button"
            onClick={() => setExpandedTable(
              <div className="w-full">
                <table className="w-full text-base border-collapse border" {...props}>
                  {children}
                </table>
              </div>
            )}
            className="mt-1 text-xs text-muted-foreground hover:text-foreground flex items-center gap-1"
          >
            <ExternalLinkIcon className="w-3 h-3" />
            Expand table
          </button>
        </div>
      );
    },
    thead: function ChatTheadComponent({ children, ...props }: any) {
      return <thead className="bg-muted/50 sticky top-0" {...props}>{children}</thead>;
    },
    th: function ChatThComponent({ children, ...props }: any) {
      return <th className="px-2 py-1 text-left font-medium border-b" {...props}>{children}</th>;
    },
    td: function ChatTdComponent({ children, ...props }: any) {
      return <td className="px-2 py-1 border-b" {...props}>{children}</td>;
    },
  };
  
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
      publish: boolean;
      authorId: string;
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

  // Mutation for updating existing articles (saves to draft_* fields)
  const updateArticleMutation = useMutation({
    mutationFn: (data: {
      slug: string;
      updateData: {
        title: string;
        content: string;
        image_url?: string;
        tags: string[];
      };
      returnToDashboard?: boolean;
    }) => updateArticle(data.slug, data.updateData),
    onSuccess: (response, variables) => {
      toast({ title: "Success", description: "Draft saved successfully." });
      
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
      
      // Invalidate version history cache since we created a new draft version
      queryClient.invalidateQueries({ queryKey: ['article-versions', oldSlug] });
      
      if (variables.returnToDashboard) {
        navigate({ to: '/dashboard/blog' });
      } else {
        // If we are *not* navigating away, refresh local state:
        const data = variables.updateData;
        setValue('title', data.title);
        setValue('content', data.content);
        setValue('image_url', data.image_url || '');
        setValue('tags', data.tags);
      }
    },
    onError: (error) => {
      console.error('Error updating article:', error);
      const errorMessage = error instanceof Error ? error.message : "Failed to save draft. Please try again.";
      toast({ title: "Error", description: errorMessage, variant: "destructive" });
    }
  });

  // Mutation for publishing an article (copies draft to published)
  const publishMutation = useMutation({
    mutationFn: () => publishArticle(blogSlug as string),
    onSuccess: (response) => {
      toast({ title: "Success", description: "Article published successfully." });
      queryClient.setQueryData(['article', blogSlug], response);
      queryClient.invalidateQueries({ queryKey: ['articles'] });
      queryClient.invalidateQueries({ queryKey: ['article-versions', blogSlug] });
    },
    onError: (error) => {
      console.error('Error publishing article:', error);
      const errorMessage = error instanceof Error ? error.message : "Failed to publish article. Please try again.";
      toast({ title: "Error", description: errorMessage, variant: "destructive" });
    }
  });

  // Mutation for unpublishing an article (removes from public view)
  const unpublishMutation = useMutation({
    mutationFn: () => unpublishArticle(blogSlug as string),
    onSuccess: (response) => {
      toast({ title: "Success", description: "Article unpublished successfully." });
      queryClient.setQueryData(['article', blogSlug], response);
      queryClient.invalidateQueries({ queryKey: ['articles'] });
      queryClient.invalidateQueries({ queryKey: ['article-versions', blogSlug] });
    },
    onError: (error) => {
      console.error('Error unpublishing article:', error);
      toast({ title: "Error", description: "Failed to unpublish article. Please try again.", variant: "destructive" });
    }
  });

  // State for version history
  const [showVersions, setShowVersions] = useState(false);
  const [selectedVersion, setSelectedVersion] = useState<ArticleVersion | null>(null);

  // Query for fetching version history
  const { data: versionsData, isLoading: versionsLoading } = useQuery({
    queryKey: ['article-versions', blogSlug],
    queryFn: () => listArticleVersions(blogSlug as string),
    enabled: !!blogSlug && !isNew && showVersions,
  });

  // Mutation for reverting to a previous version
  const revertMutation = useMutation({
    mutationFn: (versionId: string) => revertToVersion(blogSlug as string, versionId),
    onSuccess: (response) => {
      toast({ title: "Success", description: "Reverted to previous version." });
      queryClient.setQueryData(['article', blogSlug], response);
      queryClient.invalidateQueries({ queryKey: ['articles'] });
      queryClient.invalidateQueries({ queryKey: ['article-versions', blogSlug] });
      
      // Reset form with reverted content
      const tagNames = response.tags ? response.tags
        .map((tag: any) => tag?.name?.toUpperCase())
        .filter((name: string | undefined) => !!name && name !== '') : [];
      reset({
        title: response.article.draft_title,
        content: response.article.draft_content,
        image_url: response.article.draft_image_url || '',
        tags: tagNames,
      });
      
      // Sync editor content
      if (editor) {
        editor.commands.setContent(response.article.draft_content || '');
      }
      
      setSelectedVersion(null);
      setShowVersions(false);
    },
    onError: (error) => {
      console.error('Error reverting to version:', error);
      toast({ title: "Error", description: "Failed to revert to version. Please try again.", variant: "destructive" });
    }
  });

  const { register, handleSubmit, setValue, formState: { errors }, control, reset } = useForm<ArticleFormData>({
    resolver: zodResolver(articleSchema),
    defaultValues: {
      title: '',
      content: '',
      image_url: '',
      tags: [],
    }
  });

  // Watch only the specific fields that need reactive UI updates (NOT content - that causes re-renders on every keystroke)
  const watchedTags = useWatch({ control, name: 'tags' });

  const [imagePrompt, setImagePrompt] = useState<string | null>(DEFAULT_IMAGE_PROMPT[Math.floor(Math.random() * DEFAULT_IMAGE_PROMPT.length)]);

  /* --------------------------------------------------------------------- */
  /* Tiptap Editor Setup                                                   */
  /* --------------------------------------------------------------------- */
  const mdParserRef = useRef<MarkdownIt>();
  if (!mdParserRef.current) {
    mdParserRef.current = new MarkdownIt({ typographer: true, html: true });
  }
  const mdParser = mdParserRef.current;

  // turndownService is imported from ./editor/turndown (module-level singleton)

  const [diffing, setDiffing] = useState(false);
  const [originalDocument, setOriginalDocument] = useState<string>('');
  const [pendingNewDocument, setPendingNewDocument] = useState<string>('');
  const [currentDiffReason, setCurrentDiffReason] = useState<string>('');


  // Inline diff lifecycle helpers
  // editInfo is optional - if provided, uses precise edit boundaries instead of full-doc diff
  const enterDiffPreview = (
    oldHtml: string, 
    newHtml: string, 
    reason?: string,
    editInfo?: { originalText: string; newText: string; htmlIndex: number }
  ) => {
    if (!editor) return;
    
    // Set the editor content to the new HTML
    editor.commands.setContent(newHtml);
    
    // Store original and new HTML for accept/reject actions
    setOriginalDocument(oldHtml);
    setPendingNewDocument(newHtml);
    setCurrentDiffReason(reason || '');
    setDiffing(true);
    
    // @ts-ignore custom command provided by DiffHighlighter
    if ((editor as any).commands?.showDiff) {
      // @ts-ignore
      if (editInfo) {
        // Use precise edit boundaries - much more accurate for edit_text operations
        (editor as any).commands.showDiff(oldHtml, newHtml, editInfo);
      } else {
        // Fall back to full document diff for rewrite operations
        (editor as any).commands.showDiff(oldHtml, newHtml);
      }
    }
    
    // Force a tiny transaction to ensure decorations render
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

  // Populate form when article data is loaded (always load draft_* fields for editing)
  useEffect(() => {
    if (article && !isNew) {
      // Extract tag names from the server response format
      const tagNames = article.tags ? article.tags
        .map((tag: any) => tag?.name?.toUpperCase())
        .filter((name: string | undefined) => !!name && name !== '') : [];
      const newValues = {
        title: article.article.draft_title || '',
        content: article.article.draft_content || '',
        image_url: article.article.draft_image_url || '',
        tags: tagNames,
      } as ArticleFormData;
      reset(newValues);
      
      // Initialize image versions if there's an existing image
      if (article.article.draft_image_url) {
        setImageVersions([{ url: article.article.draft_image_url, timestamp: Date.now() }]);
        setCurrentVersionIndex(0);
        setPreviewImageUrl(article.article.draft_image_url);
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
            
            // Extract chain of thought steps from metadata
            if (msg.meta_data?.steps && msg.meta_data.steps.length > 0) {
              chatMsg.steps = msg.meta_data.steps.map((step: any) => {
                const turnStep: TurnStep = { type: step.type };
                if (step.type === 'reasoning' && step.reasoning) {
                  turnStep.thinking = {
                    content: step.reasoning.content || '',
                    duration_ms: step.reasoning.duration_ms,
                    visible: step.reasoning.visible ?? true,
                  };
                } else if (step.type === 'tool' && step.tool) {
                  // Convert tool step to tool_group format
                  turnStep.toolGroup = {
                    group_id: step.tool.tool_id,
                    status: step.tool.status === 'completed' ? 'completed' : 
                            step.tool.status === 'error' ? 'error' : 'running',
                    calls: [{
                      id: step.tool.tool_id,
                      name: step.tool.tool_name,
                      input: step.tool.input || {},
                      status: step.tool.status === 'completed' ? 'completed' : 
                              step.tool.status === 'error' ? 'error' : 'running',
                      result: step.tool.output,
                      error: step.tool.error,
                      started_at: step.tool.started_at || '',
                      completed_at: step.tool.completed_at,
                      duration_ms: step.tool.duration_ms,
                    }],
                  };
                  turnStep.type = 'tool_group'; // Convert 'tool' to 'tool_group' for frontend
                } else if (step.type === 'content' && step.content) {
                  turnStep.content = step.content;
                }
                return turnStep;
              });
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

  // Keyboard shortcut: Cmd+S (Mac) or Ctrl+S (Windows/Linux) to save
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 's') {
        e.preventDefault();
        // Don't save if already saving
        if (createArticleMutation.isPending || updateArticleMutation.isPending) {
          return;
        }
        // Reject pending diff changes before saving
        if (diffing) {
          rejectDiff();
        }
        handleSubmit((data) => onSubmit(data, false))();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [diffing, rejectDiff, handleSubmit, createArticleMutation.isPending, updateArticleMutation.isPending]);

  const onSubmit = async (data: ArticleFormData, returnToDashboard: boolean = true) => {
    if (!user) {
      toast({ title: "Error", description: "You must be logged in to edit an article." });
      return;
    }

    // Ensure staged image URL is synced to form data before saving
    const finalImageUrl = stagedImageUrl !== undefined ? stagedImageUrl : data.image_url;

    if (isNew) {
      // New articles are created as drafts by default
      // Use publishArticle() separately to publish
      createArticleMutation.mutate({
        title: data.title,
        content: data.content,
        image_url: finalImageUrl || undefined,
        tags: data.tags,
        publish: false, // Save as draft, publish is a separate action
        authorId: String(user.id),
      });
    } else {
      // Updates always go to draft_* fields
      // Use publishArticle() separately to publish
      const updateData = {
        title: data.title,
        content: data.content, // HTML content from Tiptap editor
        image_url: finalImageUrl || undefined,
        tags: data.tags,
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

  // Apply text edit from AI assistant (markdown-based str_replace)
  // Computes the edit as old_str/new_str chunks and uses precise diff mode.
  const applyTextEdit = (oldStr: string, newStr: string, reason: string, newMarkdown?: string) => {
    if (!editor) return;
    
    const currentHtml = editor.getHTML();
    
    let newHtml: string;
    let oldMd: string;
    if (newMarkdown) {
      // Backend applied the edit and returned full new markdown
      newHtml = mdParser.render(newMarkdown);
      // Reconstruct old markdown by reversing the edit (for the normalized old HTML)
      // Find new_str in new_markdown and replace with old_str to get old_markdown
      const newStrIdx = newMarkdown.indexOf(newStr);
      if (newStrIdx !== -1) {
        oldMd = newMarkdown.substring(0, newStrIdx) + oldStr + newMarkdown.substring(newStrIdx + newStr.length);
      } else {
        oldMd = turndownService.turndown(currentHtml);
      }
    } else {
      // Fallback: apply the edit locally on markdown
      oldMd = turndownService.turndown(currentHtml);
      const index = oldMd.indexOf(oldStr);
      if (index === -1) {
        toast({
          title: 'Edit Warning',
          description: 'Could not locate the text to edit in the document markdown. The document may have changed.',
          variant: 'destructive'
        });
        return;
      }
      newMarkdown = oldMd.substring(0, index) + newStr + oldMd.substring(index + oldStr.length);
      newHtml = mdParser.render(newMarkdown);
    }
    
    // Render old markdown to HTML through the same renderer so both sides are consistent
    const oldHtml = mdParser.render(oldMd);

    // Compute the edit position in the rendered new HTML by finding the new_str content.
    // Render old_str and new_str individually to get their HTML representations
    const oldStrHtml = mdParser.render(oldStr).trim();
    const newStrHtml = mdParser.render(newStr).trim();
    
    // Find where new_str HTML appears in the full new HTML
    const htmlIndex = newHtml.indexOf(newStrHtml);

    if (htmlIndex !== -1) {
      // Use PRECISE MODE with the exact edit boundaries
      enterDiffPreview(oldHtml, newHtml, reason, {
        originalText: oldStrHtml,
        newText: newStrHtml,
        htmlIndex: htmlIndex,
      });
    } else {
      // Fallback to full-document diff mode
      enterDiffPreview(oldHtml, newHtml, reason);
    }
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

    // Get current document content in both HTML and markdown formats
    const currentContent = editor?.getHTML() || '';
    const currentMarkdown = currentContent ? turndownService.turndown(currentContent) : '';

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
    await performChatRequest(text, assistantIndex, isEditRequest, currentContent, currentMarkdown);
  };

  const sendChat = async () => {
    const text = chatInput.trim();
    
    if (!text) {
      return;
    }

    await sendChatWithMessage(text);
  };

  const performChatRequest = async (messageText: string, assistantIndex: number, isEditRequest: boolean, documentContent: string, documentMarkdown?: string) => {
    setChatLoading(true);
    try {
      if (!article?.article?.id) {
        throw new Error('Article ID is required');
      }
      
      // Submit the request with single message - backend loads context from DB
      const result = await apiPost<{ requestId: string; status: string }>('/agent', {
        message: messageText,  // Single message string
        documentContent: documentContent,
        documentMarkdown: documentMarkdown || '',  // Markdown version for agent editing
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

              case 'reasoning_delta':
                // Handle reasoning/extended thinking content from LLM
                setIsThinking(true);
                setThinkingMessage('Reasoning...');
                if (msg.thinking_content) {
                  setChatMessages((prev) => {
                    const updated = [...prev];
                    if (updated[assistantIndex]) {
                      const currentMsg = updated[assistantIndex];
                      const steps = [...(currentMsg.steps || [])];
                      const stepIdx = msg.step_index ?? 0;
                      
                      // Ensure step exists at stepIdx
                      while (steps.length <= stepIdx) {
                        steps.push({ type: 'reasoning', thinking: { content: '', visible: true }, isStreaming: true });
                      }
                      
                      // Append to the reasoning step at stepIdx
                      if (steps[stepIdx].type === 'reasoning' && steps[stepIdx].thinking) {
                        steps[stepIdx] = {
                          ...steps[stepIdx],
                          thinking: {
                            ...steps[stepIdx].thinking!,
                            content: (steps[stepIdx].thinking!.content || '') + msg.thinking_content,
                          },
                          isStreaming: true,
                        };
                      }
                      
                      // Also update legacy thinking for backward compatibility
                      const currentMetaData = currentMsg.meta_data || {};
                      const currentThinking = currentMetaData.thinking || { content: '', visible: true };
                      
                      updated[assistantIndex] = {
                        ...currentMsg,
                        steps,
                        meta_data: {
                          ...currentMetaData,
                          thinking: {
                            ...currentThinking,
                            content: (currentThinking.content || '') + msg.thinking_content,
                          },
                        },
                        isReasoningStreaming: true,
                      };
                    }
                    return updated;
                  });
                }
                break;

              case 'content_delta':
                // Handle real-time content chunks
                setIsThinking(false);
                if (msg.content) {
                  setChatMessages((prev) => {
                    const updated = [...prev];
                    if (updated[assistantIndex]) {
                      const currentMsg = updated[assistantIndex];
                      const steps = [...(currentMsg.steps || [])];
                      const stepIdx = msg.step_index ?? steps.length;
                      
                      // Mark any previous reasoning steps as done
                      steps.forEach((step, idx) => {
                        if (step.type === 'reasoning' && step.isStreaming) {
                          steps[idx] = { ...step, isStreaming: false };
                        }
                      });
                      
                      // Ensure content step exists at stepIdx
                      if (steps.length <= stepIdx) {
                        steps.push({ type: 'content', content: '' });
                      }
                      
                      // Append to content step
                      if (steps[stepIdx].type === 'content') {
                        steps[stepIdx] = {
                          ...steps[stepIdx],
                          content: (steps[stepIdx].content || '') + msg.content,
                        };
                      }
                      
                      updated[assistantIndex] = {
                        ...currentMsg,
                        steps,
                        content: (currentMsg.content || '') + msg.content,
                        isReasoningStreaming: false, // Reasoning is complete when content starts
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
                // Handle assistant text responses - always update the message at assistantIndex
                if (msg.content) {
                  if (!hasInitialContent) {
                    currentAssistantContent = msg.content;
                    hasInitialContent = true;
                  } else {
                    // Append subsequent text blocks
                    currentAssistantContent += msg.content;
                  }
                  
                  setChatMessages((prev) => {
                    const updated = [...prev];
                    if (updated[assistantIndex]) {
                      updated[assistantIndex] = { 
                        ...updated[assistantIndex],
                        content: currentAssistantContent 
                      };
                    }
                    return updated;
                  });
                }
                break;
                
              case 'tool_use':
                // Hide thinking state on tool use
                setIsThinking(false);
                
                // Display tool usage feedback using tool_context for all tools
                if (msg.tool_name) {
                  // Also add tool step to the current assistant message's steps array
                  setChatMessages((prev) => {
                    const updated = [...prev];
                    if (updated[assistantIndex]) {
                      const currentMsg = updated[assistantIndex];
                      const steps = [...(currentMsg.steps || [])];
                      
                      // Mark any previous reasoning steps as done
                      steps.forEach((step, idx) => {
                        if (step.type === 'reasoning' && step.isStreaming) {
                          steps[idx] = { ...step, isStreaming: false };
                        }
                      });
                      
                      // Add tool step
                      steps.push({
                        type: 'tool_group',
                        toolGroup: {
                          group_id: msg.tool_id || `tool-${Date.now()}`,
                          status: 'running',
                          calls: [{
                            id: msg.tool_id || `call-${Date.now()}`,
                            name: msg.tool_name,
                            input: msg.tool_input as Record<string, unknown> || {},
                            status: 'running',
                            started_at: new Date().toISOString(),
                          }],
                        },
                      });
                      
                      updated[assistantIndex] = {
                        ...currentMsg,
                        steps,
                        isReasoningStreaming: false,
                      };
                    }
                    return updated;
                  });
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
                      
                      // Update tool steps in the assistant message
                      setChatMessages((prev) => {
                        const updated = [...prev];
                        if (updated[assistantIndex] && updated[assistantIndex].steps) {
                          const steps = [...updated[assistantIndex].steps!];
                          // Find and update the matching tool step
                          for (let i = steps.length - 1; i >= 0; i--) {
                            if (steps[i].type === 'tool_group' && steps[i].toolGroup) {
                              const calls = steps[i].toolGroup!.calls;
                              for (let j = 0; j < calls.length; j++) {
                                if (calls[j].name === toolResult.tool_name && calls[j].status === 'running') {
                                  const toolStatus = toolResult.is_error ? 'error' : 'completed';
                                  calls[j] = {
                                    ...calls[j],
                                    status: toolStatus,
                                    result: toolResult,
                                    error: toolResult.is_error ? (toolResult.error || 'Edit failed') : undefined,
                                    completed_at: new Date().toISOString(),
                                  };
                                  // Group status is error if any call errored
                                  const groupStatus = calls.some((c: any) => c.status === 'error') ? 'error' : 'completed';
                                  steps[i] = {
                                    ...steps[i],
                                    toolGroup: {
                                      ...steps[i].toolGroup!,
                                      status: groupStatus,
                                      calls: [...calls],
                                    },
                                  };
                                  break;
                                }
                              }
                              break;
                            }
                          }
                          updated[assistantIndex] = {
                            ...updated[assistantIndex],
                            steps,
                          };
                        }
                        return updated;
                      });
                      
                      // For edit tools, also apply the diff preview (only for successful results)
                      if (toolResult.tool_name === 'edit_text' && isNewMessage && !toolResult.is_error) {
                        applyTextEdit(toolResult.old_str || toolResult.original_text, toolResult.new_str || toolResult.new_text, toolResult.reason, toolResult.new_markdown);
                        setProcessedToolMessages(prev => new Set(prev).add(toolMessageId));
                      } else if (toolResult.tool_name === 'rewrite_document' && isNewMessage && !toolResult.is_error) {
                        applyDocumentRewrite(toolResult.new_content, toolResult.reason, toolResult.original_content);
                        setProcessedToolMessages(prev => new Set(prev).add(toolMessageId));
                      }
                    }
                  } catch (error) {
                    console.error('[Editor] Failed to parse tool result:', error);
                    // Update tool step to error state
                    setChatMessages((prev) => {
                      const updated = [...prev];
                      if (updated[assistantIndex] && updated[assistantIndex].steps) {
                        const steps = [...updated[assistantIndex].steps!];
                        // Find and mark last running tool step as error
                        for (let i = steps.length - 1; i >= 0; i--) {
                          if (steps[i].type === 'tool_group' && steps[i].toolGroup?.status === 'running') {
                            steps[i] = {
                              ...steps[i],
                              toolGroup: {
                                ...steps[i].toolGroup!,
                                status: 'error',
                              },
                            };
                            break;
                          }
                        }
                        updated[assistantIndex] = { ...updated[assistantIndex], steps };
                      }
                      return updated;
                    });
                  }
                }
                break;

              case 'full_message':
                // Handle full message with complete meta_data - merge with streaming state
                setIsThinking(false);
                if (msg.full_message) {
                  const fullMsg = msg.full_message;
                  
                  setChatMessages((prev) => {
                    const updated = [...prev];
                    if (updated[assistantIndex]) {
                      const existingMsg = updated[assistantIndex];
                      
                      // Get steps: prefer existing streaming steps, fall back to meta_data.steps
                      let finalSteps = existingMsg.steps || [];
                      
                      // If meta_data has steps and we don't have streaming steps, convert them
                      if (fullMsg.meta_data?.steps && fullMsg.meta_data.steps.length > 0 && finalSteps.length === 0) {
                        finalSteps = fullMsg.meta_data.steps.map((step: any) => {
                          const turnStep: TurnStep = { type: step.type };
                          if (step.type === 'reasoning' && step.reasoning) {
                            turnStep.thinking = {
                              content: step.reasoning.content || '',
                              duration_ms: step.reasoning.duration_ms,
                              visible: step.reasoning.visible ?? true,
                            };
                          } else if (step.type === 'tool' && step.tool) {
                            turnStep.toolGroup = {
                              group_id: step.tool.tool_id,
                              status: step.tool.status === 'completed' ? 'completed' : 
                                      step.tool.status === 'error' ? 'error' : 'running',
                              calls: [{
                                id: step.tool.tool_id,
                                name: step.tool.tool_name,
                                input: step.tool.input || {},
                                status: step.tool.status === 'completed' ? 'completed' : 
                                        step.tool.status === 'error' ? 'error' : 'running',
                                result: step.tool.output,
                                error: step.tool.error,
                                started_at: step.tool.started_at || '',
                                completed_at: step.tool.completed_at,
                                duration_ms: step.tool.duration_ms,
                              }],
                            };
                            turnStep.type = 'tool_group';
                          } else if (step.type === 'content' && step.content) {
                            turnStep.content = step.content;
                          }
                          return turnStep;
                        });
                      }
                      
                      // Clear streaming flags on all steps
                      finalSteps = finalSteps.map(step => ({
                        ...step,
                        isStreaming: false,
                      }));
                      
                      // Build tool_group from tool_execution if present
                      let toolGroup = existingMsg.tool_group;
                      if (fullMsg.meta_data?.tool_execution) {
                        const toolExec = fullMsg.meta_data.tool_execution;
                        const output = toolExec.output;
                        toolGroup = {
                          group_id: toolExec.tool_id || `group-${fullMsg.id}`,
                          status: toolExec.success !== false ? 'completed' : 'error',
                          calls: [{
                            id: toolExec.tool_id || `call-${fullMsg.id}`,
                            name: toolExec.tool_name,
                            input: typeof toolExec.input === 'object' ? toolExec.input : {},
                            status: toolExec.success !== false ? 'completed' : 'error',
                            result: typeof output === 'object' ? output : undefined,
                            error: toolExec.error,
                            started_at: toolExec.executed_at || fullMsg.created_at,
                            duration_ms: toolExec.duration_ms,
                          }],
                        };
                      }
                      
                      // Merge full_message data with existing streaming state
                      updated[assistantIndex] = {
                        ...existingMsg,
                        id: fullMsg.id,
                        content: fullMsg.content || existingMsg.content,
                        meta_data: fullMsg.meta_data,
                        created_at: fullMsg.created_at,
                        steps: finalSteps,
                        tool_group: toolGroup,
                        isReasoningStreaming: false,
                      };
                    }
                    return updated;
                  });
                  
                  // Don't apply diff here -- the tool_result handler already applied it.
                  // Applying again from full_message would overwrite the diff decorations.
                }
                break;
                
              case 'done':
                setIsThinking(false); // Hide thinking state on completion
                
                // Clear all streaming flags on the current assistant message
                setChatMessages((prev) => {
                  const updated = [...prev];
                  if (updated[assistantIndex]) {
                    const msg = updated[assistantIndex];
                    // Clear streaming flags on all steps
                    const finalSteps = msg.steps?.map(step => ({
                      ...step,
                      isStreaming: false,
                    }));
                    updated[assistantIndex] = {
                      ...msg,
                      steps: finalSteps,
                      isReasoningStreaming: false,
                    };
                  }
                  return updated;
                });
                
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
              
              // Handle edit_text tool specifically - only for new messages (skip error results)
              if (toolResult.tool_name === 'edit_text' && isNewMessage && !toolResult.is_error) {
                applyTextEdit(toolResult.old_str || toolResult.original_text, toolResult.new_str || toolResult.new_text, toolResult.reason, toolResult.new_markdown);
                
                // Mark this tool message as processed
                setProcessedToolMessages(prev => new Set(prev).add(toolMessageId));
              } else if (toolResult.tool_name === 'rewrite_document' && isNewMessage && !toolResult.is_error) {
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
      <div className="flex-1 flex flex-col">
        {/* Article Metadata Card */}
        
            {/* Article Title Section with Image and Save */}
            <div className="mb-6">
              <div className="flex flex-row items-center gap-3">
                {/* Edit Image Trigger */}
                <Dialog open={imageModalOpen} onOpenChange={setImageModalOpen}>
                  <DialogTrigger asChild>
                    <button
                      type="button"
                      className="w-12 h-10 flex items-center justify-center rounded-md border border-gray-300 dark:border-gray-600 overflow-hidden cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors flex-shrink-0"
                    >
                      {(stagedImageUrl || article?.article.draft_image_url) ? (
                        <img 
                          src={stagedImageUrl || article?.article.draft_image_url} 
                          alt="Article header" 
                          className="w-full h-full object-cover"
                        />
                      ) : (
                        <ImageIcon className="w-5 h-5 text-muted-foreground" />
                      )}
                    </button>
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
                                const result = await generateArticleImage(article?.article.draft_title || '', article?.article.id || '');
                                if (result.success) {
                                  setNewImageGenerationRequestId(result.generationRequestId);
                                  // Add to versions when image is generated
                                  if (result.generationRequestId) {
                                    setTimeout(async () => {
                                      const status = await getImageGenerationStatus(result.generationRequestId);
                                      if (status.outputUrl) {
                                        addImageVersion(status.outputUrl, article?.article.draft_title || '');
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

                {/* Title Input */}
                <div className="flex-1">
                  <Input
                    {...register('title')}
                    placeholder="Article Title"
                    className="w-full text-lg font-medium"
                  />
                  {errors.title && <p className="text-red-500 text-sm mt-1">{errors.title.message}</p>}
                </div>

                {/* Save Button */}
                <Button
                  type="button"
                  onClick={() => {
                    if (diffing) {
                      rejectDiff();
                    }
                    handleSubmit((data) => onSubmit(data, false))();
                  }}
                  disabled={createArticleMutation.isPending || updateArticleMutation.isPending}
                  className="flex-shrink-0"
                >
                  {(createArticleMutation.isPending || updateArticleMutation.isPending) ? 
                    (isNew ? 'Creating...' : 'Saving...') : 
                    'Save'
                  }
                </Button>
              </div>
            </div>

            {/* Article Tools Section */}
            <div className="flex flex-wrap items-center gap-2 mb-4">
              {/* Sources Button */}
              <SourcesPreview
                articleId={article?.article.id}
                onOpenDrawer={() => setSourcesManagerOpen(true)}
                disabled={!article && isNew}
                refreshTrigger={sourcesRefreshTrigger}
              />

              {/* Tags Button */}
              <Drawer direction="right">
                <DrawerTrigger asChild>
                  <Button variant="outline" size="sm">
                    <Tag className="h-4 w-4" />
                    Tags
                    {watchedTags && watchedTags.length > 0 && (
                      <Badge variant="secondary" className="ml-1">
                        {watchedTags.length}
                      </Badge>
                    )}
                  </Button>
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

              {/* Publish Button */}
              <Drawer direction="right">
                <DrawerTrigger asChild>
                  <Button variant="outline" size="sm">
                    {isPublished(article?.article) ? (
                      <Globe className="h-4 w-4" />
                    ) : (
                      <EyeOff className="h-4 w-4" />
                    )}
                    {isPublished(article?.article) ? "Published" : "Draft"}
                    {isPublished(article?.article) && hasDraftChanges(article?.article) && (
                      <Badge variant="secondary" className="ml-1">*</Badge>
                    )}
                  </Button>
                </DrawerTrigger>

                {/* Drawer content for publishing settings */}
                <DrawerContent className="w-full sm:max-w-sm ml-auto">
                  <DrawerHeader>
                    <DrawerTitle>Publishing Settings</DrawerTitle>
                    <DrawerDescription>Manage article publication status.</DrawerDescription>
                  </DrawerHeader>
                  <div className="space-y-4 px-4">
                    {/* Status Display */}
                    <div className="flex items-center gap-2 mb-4">
                      <Badge variant={isPublished(article?.article) ? "default" : "secondary"}>
                        {isPublished(article?.article) ? "Published" : "Draft Only"}
                      </Badge>
                      {article?.article.published_at && (
                        <span className="text-xs text-muted-foreground">
                          Published {format(new Date(article.article.published_at), 'PPP')}
                        </span>
                      )}
                    </div>

                    {/* Show if draft differs from published */}
                    {isPublished(article?.article) && hasDraftChanges(article?.article) && (
                      <div className="p-2 bg-amber-50 dark:bg-amber-900/20 rounded text-sm text-amber-800 dark:text-amber-200">
                        Draft has unpublished changes
                      </div>
                    )}

                    {/* Action Buttons */}
                    <div className="space-y-2">
                      {!isPublished(article?.article) ? (
                        <Button 
                          onClick={() => publishMutation.mutate()} 
                          disabled={publishMutation.isPending || isNew}
                          className="w-full"
                        >
                          <Globe className="mr-2 h-4 w-4" />
                          {publishMutation.isPending ? 'Publishing...' : 'Publish'}
                        </Button>
                      ) : (
                        <>
                          <Button 
                            onClick={() => publishMutation.mutate()} 
                            variant="outline" 
                            disabled={publishMutation.isPending}
                            className="w-full"
                          >
                            <RefreshCw className={cn("mr-2 h-4 w-4", publishMutation.isPending && "animate-spin")} />
                            {publishMutation.isPending ? 'Updating...' : 'Update Published'}
                          </Button>
                          <Button 
                            onClick={() => unpublishMutation.mutate()} 
                            variant="destructive" 
                            disabled={unpublishMutation.isPending}
                            className="w-full"
                          >
                            <EyeOff className="mr-2 h-4 w-4" />
                            {unpublishMutation.isPending ? 'Unpublishing...' : 'Unpublish'}
                          </Button>
                        </>
                      )}
                    </div>

                    {isNew && (
                      <p className="text-xs text-muted-foreground">
                        Save the article first before publishing.
                      </p>
                    )}
                  </div>
                  <DrawerFooter>
                    <DrawerClose asChild>
                      <Button variant="outline" className="w-full">Done</Button>
                    </DrawerClose>
                  </DrawerFooter>
                </DrawerContent>
              </Drawer>

              {/* View Article Button */}
              {!isNew && (
                <Button variant="outline" size="sm" asChild>
                  <Link
                    to="/blog"
                    params={{ slug: article?.article.slug || '' }}
                    search={{ page: undefined, tag: undefined, search: undefined }}
                    target="_blank"
                  >
                    <ExternalLinkIcon className="h-4 w-4" />
                    View
                  </Link>
                </Button>
              )}

              {/* Version History Button */}
              {!isNew && (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setShowVersions(true)}
                >
                  <History className="h-4 w-4" />
                  History
                </Button>
              )}

              {/* Regenerate Button */}
              {!isNew && (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={rewriteArticle}
                  disabled={generatingRewrite}
                >
                  <RefreshCw className={cn('h-4 w-4', generatingRewrite && 'animate-spin')} />
                  Regenerate
                </Button>
              )}

              {/* Settings Button */}
              {!isNew && (
                <Drawer open={showSettingsDrawer} onOpenChange={setShowSettingsDrawer} direction="right">
                  <DrawerTrigger asChild>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                    >
                      <Settings className="h-4 w-4" />
                    </Button>
                  </DrawerTrigger>
                  <DrawerContent>
                    <DrawerHeader>
                      <DrawerTitle>Chat Settings</DrawerTitle>
                      <DrawerDescription>
                        Manage your chat assistant settings
                      </DrawerDescription>
                    </DrawerHeader>
                    <div className="p-4 space-y-4">
                      <div className="flex items-center justify-between p-4 border rounded-lg">
                        <div className="space-y-1">
                          <p className="font-medium">Clear Chat History</p>
                          <p className="text-sm text-muted-foreground">
                            Remove all messages and start fresh with a new conversation
                          </p>
                        </div>
                        <Button
                          variant="destructive"
                          size="sm"
                          disabled={clearingChat}
                          onClick={async () => {
                            if (!article?.article?.id) return;
                            setClearingChat(true);
                            try {
                              await clearConversationHistory(article.article.id);
                              // Reload conversation to get the fresh initial greeting
                              const result = await getConversationHistory(article.article.id);
                              setChatMessages(result.messages?.map((msg: any) => ({
                                id: msg.id,
                                role: msg.role,
                                content: msg.content,
                              })) || []);
                              toast({
                                title: "Chat cleared",
                                description: "Your conversation history has been reset.",
                              });
                              setShowSettingsDrawer(false);
                            } catch (error) {
                              console.error('Failed to clear chat history:', error);
                              toast({
                                title: "Error",
                                description: "Failed to clear chat history. Please try again.",
                                variant: "destructive",
                              });
                            } finally {
                              setClearingChat(false);
                            }
                          }}
                        >
                          <Trash2 className="h-4 w-4 mr-2" />
                          {clearingChat ? 'Clearing...' : 'Clear'}
                        </Button>
                      </div>
                    </div>
                    <DrawerFooter>
                      <DrawerClose asChild>
                        <Button variant="outline">Close</Button>
                      </DrawerClose>
                    </DrawerFooter>
                  </DrawerContent>
                </Drawer>
              )}
            </div>

          <form className="flex-1 flex flex-col min-h-0">

              <div className="flex-1 flex flex-col border border-gray-300 dark:border-gray-600 rounded-md min-h-0">
                <FormattingToolbar editor={editor} />
                <EditorContent
                  editor={editor}
                  className="tiptap w-full border-none rounded-b-md flex-1 overflow-y-auto focus:outline-none"
                />
                {errors.content && <p className="text-red-500">{errors.content.message}</p>}
              </div>

          </form>

      </div>

      {/* Chat side-panel */}
      <div className="hidden xl:flex flex-col w-[26rem] border rounded-md">
        <div ref={chatMessagesRef} className="flex-1 overflow-y-auto p-2 space-y-3">
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
                          const oldContent = (result.old_str || result.original_text || result.original_content) as string;
                          const newContent = (result.new_str || result.new_text || result.new_content) as string;
                          const newMarkdown = (result.new_markdown) as string | undefined;
                          const reason = (result.reason as string) || '';
                          
                          if (newContent) {
                            if (call.name === 'edit_text' && oldContent) {
                              // edit_text: markdown str_replace
                              applyTextEdit(oldContent, newContent, reason, newMarkdown);
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
                  old_str?: string;          // new edit_text format
                  new_str?: string;          // new edit_text format
                  new_markdown?: string;     // new edit_text format
                  original_text?: string;    // legacy edit_text format
                  new_text?: string;         // legacy edit_text format
                  original_content?: string; // rewrite_document format
                  new_content?: string;      // rewrite_document format
                } | undefined;
                
                // edit_text uses old_str/new_str (or legacy original_text/new_text), rewrite_document uses original_content/new_content
                const oldText = toolOutput?.old_str || toolOutput?.original_text || toolOutput?.original_content || '';
                const newText = toolOutput?.new_str || toolOutput?.new_text || toolOutput?.new_content || '';
                const newMarkdown = toolOutput?.new_markdown;
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
                        // edit_text: markdown str_replace
                        applyTextEdit(oldText, newText, reason || '', newMarkdown);
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
                    old_str?: string;          // new edit_text format
                    new_str?: string;          // new edit_text format
                    new_markdown?: string;     // new edit_text format
                    original_text?: string;    // legacy edit_text format
                    new_text?: string;         // legacy edit_text format
                    original_content?: string; // rewrite_document format
                    new_content?: string;      // rewrite_document format
                  } | undefined;
                  
                  // Get diff data - handle both edit_text and rewrite_document formats
                  const oldText = toolOutput?.old_str || toolOutput?.original_text || toolOutput?.original_content || '';
                  const newText = toolOutput?.new_str || toolOutput?.new_text || toolOutput?.new_content || artifact.content || '';
                  const newMarkdown = toolOutput?.new_markdown;
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
                          // edit_text: markdown str_replace
                          applyTextEdit(oldText, newText, artifact.description || '', newMarkdown);
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
                
                // NEW: Use chain of thought steps if available
                if (m.steps && m.steps.length > 0) {
                  return (
                    <div key={i} className="w-full space-y-2">
                      <ChainOfThought>
                        {m.steps.map((step, stepIdx) => {
                          const isLastStep = stepIdx === m.steps!.length - 1;
                          
                          if (step.type === 'reasoning' && step.thinking?.content) {
                            return (
                              <ChainOfThoughtStep 
                                key={stepIdx}
                                type="reasoning" 
                                status={step.isStreaming ? 'running' : 'completed'}
                                isStreaming={step.isStreaming}
                                isLast={isLastStep}
                              >
                                <ChainOfThoughtTrigger>
                                  {step.isStreaming ? "Reasoning..." : "Reasoning"}
                                </ChainOfThoughtTrigger>
                                <ChainOfThoughtContent>
                                  {step.thinking.content}
                                </ChainOfThoughtContent>
                              </ChainOfThoughtStep>
                            );
                          }
                          
                          if (step.type === 'tool_group' && step.toolGroup) {
                            return (
                              <ChainOfThoughtStep 
                                key={stepIdx} 
                                type="tool" 
                                status="completed" 
                                isLast={isLastStep}
                              >
                                <ToolGroupDisplay 
                                  group={step.toolGroup}
                                  onArtifactAction={(toolId, action) => {
                                    const call = step.toolGroup?.calls.find(c => c.id === toolId);
                                    if (call && (call.name === 'edit_text' || call.name === 'rewrite_document')) {
                                      const result = call.result;
                                      if (action === 'accept' && result && editor) {
                                        const oldContent = (result.old_str || result.original_text || result.original_content) as string;
                                        const newContent = (result.new_str || result.new_text || result.new_content) as string;
                                        const newMarkdown = (result.new_markdown) as string | undefined;
                                        const reason = (result.reason as string) || '';
                                        
                                        if (newContent) {
                                          if (call.name === 'edit_text' && oldContent) {
                                            applyTextEdit(oldContent, newContent, reason, newMarkdown);
                                          } else {
                                            const currentHtml = editor.getHTML();
                                            const newHtml = mdParser.render(newContent);
                                            enterDiffPreview(currentHtml, newHtml, reason);
                                          }
                                        }
                                      }
                                    }
                                  }}
                                />
                              </ChainOfThoughtStep>
                            );
                          }
                          
                          if (step.type === 'content' && step.content) {
                            return (
                              <ChainOfThoughtItem key={stepIdx}>
                                <div className="prose prose-sm max-w-none dark:prose-invert text-sm">
                                  <Markdown components={chatMarkdownComponents}>{step.content}</Markdown>
                                </div>
                              </ChainOfThoughtItem>
                            );
                          }
                          
                          return null;
                        })}
                      </ChainOfThought>
                      
                      {/* Render content after steps if there's additional content */}
                      {m.content && !m.steps.some(s => s.type === 'content' && s.content === m.content) && (
                        <div className="prose prose-sm dark:prose-invert max-w-none text-sm">
                          <Markdown components={chatMarkdownComponents}>{m.content}</Markdown>
                        </div>
                      )}
                    </div>
                  );
                }
                
                // LEGACY: Fall back to old rendering if no steps
                // Check if there's reasoning content to display
                const thinkingContent = m.meta_data?.thinking?.content;
                const isReasoningStreaming = m.isReasoningStreaming;
                const hasReasoning = thinkingContent && thinkingContent.length > 0;
                
                // If only reasoning with no content, show reasoning
                if (hasReasoning && (!m.content || m.content === '')) {
                  return (
                    <div key={i} className="w-full space-y-2">
                      <ReasoningStep 
                        content={thinkingContent}
                        isStreaming={isReasoningStreaming}
                        durationMs={m.meta_data?.thinking?.duration_ms}
                        isLast={true}
                      />
                    </div>
                  );
                }
                  
                // Regular assistant message
                if (!m.content || m.content === '') {
                  return null; // Don't render empty messages
                }
                
                // If there's reasoning AND content, show both
                if (hasReasoning) {
                  return (
                    <div key={i} className="w-full space-y-2">
                      <ReasoningStep 
                        content={thinkingContent}
                        isStreaming={false}
                        durationMs={m.meta_data?.thinking?.duration_ms}
                        isLast={false}
                      />
                      <div className="prose prose-sm dark:prose-invert max-w-none text-sm">
                        <Markdown components={chatMarkdownComponents}>{m.content}</Markdown>
                      </div>
                    </div>
                  );
                }
                
                return (
                  <div key={i} className="w-full">
                    <div className="prose prose-sm dark:prose-invert max-w-none text-sm">
                      <Markdown components={chatMarkdownComponents}>{m.content}</Markdown>
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

      {/* Version History Drawer */}
      <Drawer open={showVersions} onOpenChange={setShowVersions} direction="right">
        <DrawerContent className="w-full sm:max-w-md ml-auto h-full">
          <DrawerHeader>
            <DrawerTitle>Version History</DrawerTitle>
            <DrawerDescription>
              {versionsData && (
                <span>{versionsData.draft_count} drafts, {versionsData.published_count} published</span>
              )}
            </DrawerDescription>
          </DrawerHeader>
          <div className="px-4 flex-1 overflow-y-auto">
            {versionsLoading ? (
              <div className="flex items-center justify-center py-8">
                <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : versionsData?.versions.length === 0 ? (
              <div className="text-center text-muted-foreground py-8">
                No versions yet. Save the article to create versions.
              </div>
            ) : (
              <div className="space-y-2">
                {versionsData?.versions.map((version) => (
                  <div 
                    key={version.id} 
                    className="flex items-center justify-between py-3 px-3 border rounded-md hover:bg-gray-50 dark:hover:bg-gray-800"
                  >
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-sm">v{version.version_number}</span>
                        <Badge variant={version.status === 'published' ? 'default' : 'secondary'} className="text-xs">
                          {version.status}
                        </Badge>
                      </div>
                      <div className="text-xs text-muted-foreground truncate mt-1">
                        {version.title}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {format(new Date(version.created_at), 'PPP p')}
                      </div>
                    </div>
                    <div className="flex gap-1 ml-2">
                      <Button 
                        size="sm" 
                        variant="ghost" 
                        onClick={() => setSelectedVersion(version)}
                        className="h-7 text-xs"
                      >
                        View
                      </Button>
                      <Button 
                        size="sm" 
                        variant="outline" 
                        onClick={() => revertMutation.mutate(version.id)}
                        disabled={revertMutation.isPending}
                        className="h-7 text-xs"
                      >
                        {revertMutation.isPending ? '...' : 'Revert'}
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
          <DrawerFooter>
            <DrawerClose asChild>
              <Button variant="outline" className="w-full">Close</Button>
            </DrawerClose>
          </DrawerFooter>
        </DrawerContent>
      </Drawer>

      {/* Version Preview Dialog */}
      <Dialog open={!!selectedVersion} onOpenChange={(open) => !open && setSelectedVersion(null)}>
        <DialogContent className="max-w-4xl max-h-[80vh] overflow-hidden flex flex-col">
          <DialogHeader>
            <DialogTitle>Version {selectedVersion?.version_number}</DialogTitle>
            <DialogDescription>
              <Badge variant={selectedVersion?.status === 'published' ? 'default' : 'secondary'} className="mr-2">
                {selectedVersion?.status}
              </Badge>
              {selectedVersion && format(new Date(selectedVersion.created_at), 'PPP p')}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 flex-1 overflow-y-auto">
            <div>
              <h4 className="font-medium mb-1 text-sm">Title</h4>
              <p className="text-sm text-muted-foreground">{selectedVersion?.title}</p>
            </div>
            <div>
              <h4 className="font-medium mb-1 text-sm">Content Preview</h4>
              <div 
                className="prose prose-sm dark:prose-invert max-h-64 overflow-y-auto border rounded p-3 text-sm"
                dangerouslySetInnerHTML={{ __html: selectedVersion?.content || '' }} 
              />
            </div>
            {selectedVersion?.image_url && (
              <div>
                <h4 className="font-medium mb-1 text-sm">Image</h4>
                <img 
                  src={selectedVersion.image_url} 
                  alt="Version image" 
                  className="max-h-32 rounded border"
                />
              </div>
            )}
          </div>
          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={() => setSelectedVersion(null)}>
              Close
            </Button>
            <Button 
              onClick={() => selectedVersion && revertMutation.mutate(selectedVersion.id)}
              disabled={revertMutation.isPending}
            >
              {revertMutation.isPending ? 'Reverting...' : 'Revert to This Version'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Expanded Table Dialog */}
      <Dialog open={!!expandedTable} onOpenChange={(open) => !open && setExpandedTable(null)}>
        <DialogContent className="!w-[90vw] !max-w-[90vw] sm:!max-w-[90vw] max-h-[80vh] overflow-hidden flex flex-col">
          <DialogHeader>
            <DialogTitle>Table View</DialogTitle>
            <DialogDescription>Full table view</DialogDescription>
          </DialogHeader>
          <div className="flex-1 overflow-auto w-full">
            {expandedTable}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setExpandedTable(null)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}
