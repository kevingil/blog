import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate, useSearch } from '@tanstack/react-router';
import { useAuth } from '@/services/auth/auth';
import { useForm, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { format } from "date-fns"
import { Calendar as CalendarIcon, PencilIcon, SparklesIcon, RefreshCw, Trash2, Mic, MessageSquare, X } from "lucide-react"
import { AnimatePresence, motion } from "framer-motion"
import { ExternalLinkIcon, UploadIcon } from '@radix-ui/react-icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { VITE_API_BASE_URL, VITE_PUBLIC_S3_URL_PREFIX } from "@/services/constants";
import { isAuthError } from '@/services/authenticatedFetch';
import { submitAgentRequest } from '@/services/agent';
import { generateArticle } from '@/services/llm/articles';
import { getArticleSources, scrapeAndCreateSource } from '@/services/sources';
import { uploadFile } from '@/services/storage';
import type { ArticleCitation } from './ArticleSources';
import { addedTextRanges, type TextRange } from '@/lib/added-text';
import { useConversation } from '@/hooks/use-conversation';

// Editor modules
import { TipTapEditor } from './editor/TipTapEditor';
import { SourcesManager } from './SourcesManager';
import { ImageLoader } from './editor/ImageLoader';
import { ImagePickerFromUploads } from './editor/ImagePickerFromUploads';
import { BlurhashImage } from '@/components/media/BlurhashImage';
import { GenerateBlurhash } from '@/components/media/GenerateBlurhash';
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
import { AttachedSource, ChatComposer, WritingSuggestions } from "@/components/chat/ChatComposer";
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
import { Wrench, BookOpen, FileSearch, PlusCircle, FileText, MoreHorizontal } from "lucide-react";
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
import { Eye, Globe, EyeOff, History, Save, Tag } from 'lucide-react';
import { Dialog, DialogTitle, DialogContent, DialogTrigger, DialogDescription, DialogFooter, DialogHeader, DialogClose } from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { Drawer, DrawerContent, DrawerHeader, DrawerTitle, DrawerDescription, DrawerFooter, DrawerClose } from '@/components/ui/drawer';
import type { UseMutationResult } from '@tanstack/react-query';

const INITIAL_CHAT_GREETING = 'Hi! I can help you improve your article. Try asking me to "rewrite the introduction" or "make the content more engaging".';

function mapConversationMessages(messages: any[]): ChatMessage[] {
  return messages.map((msg: any) => {
    const chatMsg: ChatMessage = {
      id: msg.id,
      role: msg.role,
      content: msg.content,
      channel: msg.meta_data?.input_channel === 'voice' ? 'voice' : 'text',
      meta_data: msg.meta_data,
      created_at: msg.created_at,
    };

    if (msg.meta_data?.tool_group) {
      chatMsg.tool_group = msg.meta_data.tool_group;
    } else if (msg.meta_data?.tool_execution) {
      const toolExec = msg.meta_data.tool_execution;
      const output = toolExec.output;
      const toolName = toolExec.tool_name;

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

    return chatMsg;
  }) as ChatMessage[];
}

function getInitialGreetingMessage(): ChatMessage {
  return { role: 'assistant', content: INITIAL_CHAT_GREETING } as ChatMessage;
}

function PublishDrawerContent({
  article,
  isNew = false,
  publishMutation,
  unpublishMutation,
}: {
  article: ArticleListItem | null | undefined;
  isNew?: boolean;
  publishMutation: UseMutationResult<ArticleListItem, Error, Date | undefined, unknown>;
  unpublishMutation: UseMutationResult<ArticleListItem, Error, void, unknown>;
}) {
  const currentPublishedAt = article?.article.published_at
    ? new Date(article.article.published_at)
    : undefined;
  const [selectedDate, setSelectedDate] = useState<Date | undefined>(currentPublishedAt);
  const [showCalendar, setShowCalendar] = useState(false);

  // Sync when the article's published_at changes externally
  useEffect(() => {
    if (article?.article.published_at) {
      setSelectedDate(new Date(article.article.published_at));
    }
  }, [article?.article.published_at]);

  const handlePublish = () => {
    // When already published: always send the date so backend preserves it (it defaults to "now" when omitted).
    // When not published: only send if user picked a custom date.
    const dateToSend = isPublished(article?.article)
      ? (selectedDate ?? currentPublishedAt ?? undefined)
      : (selectedDate ?? undefined);
    publishMutation.mutate(dateToSend);
  };

  return (
    <DrawerContent className="ml-auto w-full data-[vaul-drawer-direction=right]:w-full sm:max-w-sm sm:data-[vaul-drawer-direction=right]:max-w-sm">
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
        </div>

        {/* Published At Date Picker */}
        <div className="space-y-1.5">
          <label className="text-sm font-medium">Published date</label>
          <Button
            variant="outline"
            className={cn(
              "w-full justify-start text-left font-normal",
              !selectedDate && "text-muted-foreground"
            )}
            onClick={() => setShowCalendar((v) => !v)}
          >
            <CalendarIcon className="mr-2 h-4 w-4" />
            {selectedDate ? format(selectedDate, 'PPP') : 'Pick a date'}
          </Button>
          {showCalendar && (
            <div className="rounded-md border p-0">
              <Calendar
                mode="single"
                selected={selectedDate}
                onSelect={(day) => {
                  if (day) {
                    const base = selectedDate || new Date();
                    day.setHours(base.getHours(), base.getMinutes(), base.getSeconds());
                  }
                  setSelectedDate(day);
                  setShowCalendar(false);
                }}
              />
            </div>
          )}
          <p className="text-xs text-muted-foreground">
            {isPublished(article?.article)
              ? "Change the published date for this article."
              : "Optionally set a custom publish date."}
          </p>
        </div>

        {/* Created At (read-only) */}
        {article?.article.created_at && (
          <div className="space-y-1.5">
            <label className="text-sm font-medium">Created</label>
            <p className="text-sm text-muted-foreground">
              {format(new Date(article.article.created_at), 'PPP p')}
            </p>
          </div>
        )}

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
              onClick={handlePublish}
              disabled={publishMutation.isPending || isNew}
              className="w-full"
            >
              <Globe className="mr-2 h-4 w-4" />
              {publishMutation.isPending ? 'Publishing...' : 'Publish'}
            </Button>
          ) : (
            <>
              <Button
                onClick={handlePublish}
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
  );
}

const AUTOSAVE_DELAY_MS = 800;
const PENDING_DRAFT_KEY = 'blog-editor-pending-draft';

function draftSnapshot(value: Partial<ArticleFormData> | undefined): string {
  return JSON.stringify({
    title: value?.title ?? '',
    content: value?.content ?? '',
    image_url: value?.image_url ?? '',
    tags: value?.tags ?? [],
    external_url: value?.external_url ?? '',
  });
}

function rememberPendingDraft(slug: string, value: ArticleFormData) {
  try {
    sessionStorage.setItem(
      PENDING_DRAFT_KEY,
      JSON.stringify({ slug, ...JSON.parse(draftSnapshot(value)) }),
    );
  } catch {
    // Autosave still navigates; the edit page reloads the saved draft.
  }
}

function takePendingDraft(slug: string): ArticleFormData | null {
  try {
    const raw = sessionStorage.getItem(PENDING_DRAFT_KEY);
    if (!raw) return null;
    sessionStorage.removeItem(PENDING_DRAFT_KEY);
    const parsed = JSON.parse(raw) as ArticleFormData & { slug?: string };
    if (parsed.slug !== slug) return null;
    return {
      title: parsed.title ?? '',
      content: parsed.content ?? '',
      image_url: parsed.image_url ?? '',
      tags: parsed.tags ?? [],
      external_url: parsed.external_url ?? '',
    };
  } catch {
    return null;
  }
}

function storageKeyFromUrl(url: string): string | undefined {
  const prefix = (VITE_PUBLIC_S3_URL_PREFIX || "").replace(/\/$/, "");
  if (!prefix || !url.startsWith(`${prefix}/`)) return undefined;
  return decodeURIComponent(url.slice(prefix.length + 1));
}

export default function ArticleEditor({ isNew, launchpad = false }: { isNew?: boolean; launchpad?: boolean }) {
  const { toast } = useToast()
  const navigate = useNavigate();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  
  // Route params only exist on the edit page. The dashboard launchpad stays
  // mounted and fills these in after the first send or voice start.
  const routed = !isNew && !launchpad;
  const params = routed ? useParams({ from: '/dashboard/blog/edit/$blogSlug' }) : null;
  const search = routed ? useSearch({ from: '/dashboard/blog/edit/$blogSlug' }) : ({} as { requestId?: string });
  const [launchedSlug, setLaunchedSlug] = useState<string | null>(null);
  const [launchedRequestId, setLaunchedRequestId] = useState<string | null>(null);
  const [launchPhase, setLaunchPhase] = useState<'landing' | 'editor'>(launchpad ? 'landing' : 'editor');
  const [launching, setLaunching] = useState(false);
  const blogSlug = params?.blogSlug ?? launchedSlug ?? undefined;
  const initialRequestId = (search as { requestId?: string }).requestId ?? launchedRequestId ?? undefined;
  const landing = launchpad && launchPhase === 'landing';
  
  // Loading states are now handled by React Query mutations
  // const [isLoading, setIsLoading] = useState(false);
  // const [isSaving, setIsSaving] = useState(false);
  const [generatingImage, setGeneratingImage] = useState(false);
  const [newImageGenerationRequestId, setNewImageGenerationRequestId] = useState<string | null>(null);
  const [stagedImageUrl, setStagedImageUrl] = useState<string | null | undefined>(undefined);
  const [generateImageOpen, setGenerateImageOpen] = useState(false);
  const [imageModalOpen, setImageModalOpen] = useState(false);
  const [generatingRewrite, setGeneratingRewrite] = useState(false);
  const [publishDrawerOpen, setPublishDrawerOpen] = useState(false);
  const [resourcesOpen, setResourcesOpen] = useState(false);
  const [articleSources, setArticleSources] = useState<ArticleCitation[]>([]);
  const [sourcesReloadToken, setSourcesReloadToken] = useState(0);
  const [tagsOpen, setTagsOpen] = useState(false);
  const [externalOpen, setExternalOpen] = useState(false);
  const [autosaveStatus, setAutosaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const [draftEpoch, setDraftEpoch] = useState(0);
  const savedSnapshotRef = useRef<string | null>(null);
  const catchUpSnapshotRef = useRef<string | null>(null);
  const catchUpSeenRef = useRef<string | null>(null);
  const failedSnapshotRef = useRef<string | null>(null);
  const hydratedArticleIdRef = useRef<string | null>(null);
  const getFormValuesRef = useRef<(() => ArticleFormData) | null>(null);
  
  // Image versioning state
  const [imageVersions, setImageVersions] = useState<Array<{ url: string; prompt?: string; timestamp: number; uploadId?: string; blurhash?: string | null }>>([]);
  const [currentVersionIndex, setCurrentVersionIndex] = useState(-1);
  const [previewImageUrl, setPreviewImageUrl] = useState<string>('');
  const currentHeader = currentVersionIndex >= 0 ? imageVersions[currentVersionIndex] : undefined;

  // Image versioning functions
  const uploadDroppedImage = async (file: File) => {
    const safeName = file.name.replace(/[^\w.\-]+/g, '-').replace(/^-+/, '') || 'image';
    const uploaded = await uploadFile(`articles/${Date.now()}-${safeName}`, file);
    return {
      url: uploaded.url,
      alt: file.name.replace(/\.[^.]+$/, '') || 'Image',
      uploadId: uploaded.id ?? undefined,
      blurhash: uploaded.blurhash,
    };
  };

  const dropCoverImage = async (file: File) => {
    try {
      const uploaded = await uploadDroppedImage(file);
      addImageVersion(uploaded.url, undefined, {
        uploadId: uploaded.uploadId,
        blurhash: uploaded.blurhash,
      });
    } catch (error) {
      const description = error instanceof Error ? error.message : 'Could not upload that image.';
      toast({ title: 'Upload failed', description, variant: 'destructive' });
    }
  };

  const uploadBodyImage = async (file: File) => {
    try {
      const uploaded = await uploadDroppedImage(file);
      return { url: uploaded.url, alt: uploaded.alt };
    } catch (error) {
      const description = error instanceof Error ? error.message : 'Could not upload that image.';
      toast({ title: 'Upload failed', description, variant: 'destructive' });
      throw error;
    }
  };

  const addImageVersion = (url: string, prompt?: string, asset?: { uploadId?: string; blurhash?: string | null }) => {
    const newVersion = { url, prompt, timestamp: Date.now(), uploadId: asset?.uploadId, blurhash: asset?.blurhash };
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
  const launchTokenRef = useRef(0);
  const launchMessagesRef = useRef<ChatMessage[] | null>(null);
  const launchpadSessionRef = useRef(false);
  const [mobileChatOpen, setMobileChatOpen] = useState(false);
  const [chatInput, setChatInput] = useState('');
  const [inputMode, setInputMode] = useState<'text' | 'conversation'>('text');
  const playSpeechRef = useRef<(audioBase64: string, mimeType?: string) => Promise<void>>(async () => {});
  const sendLiveTextRef = useRef<(text: string) => void>(() => {});
  const liveUserIndexRef = useRef<number | null>(null);
  const pendingLiveStreamRef = useRef<{ requestId: string; assistantIndex: number } | null>(null);
  const startedLiveRequestsRef = useRef(new Set<string>());
  const [isThinking, setIsThinking] = useState(false);
  const [thinkingMessage, setThinkingMessage] = useState<string>('Thinking...');
  const chatMessagesRef = useRef<HTMLDivElement>(null);
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
    ul: function ChatUlComponent({ children, ...props }: any) {
      return <ul className="my-2 space-y-1.5 list-disc pl-6" {...props}>{children}</ul>;
    },
    ol: function ChatOlComponent({ children, ...props }: any) {
      return <ol className="my-2 space-y-1.5 list-decimal pl-6" {...props}>{children}</ol>;
    },
    li: function ChatLiComponent({ children, ...props }: any) {
      return <li className="leading-relaxed" {...props}>{children}</li>;
    },
  };
  
  // Article edits are applied on the backend. The editor only applies
  // document_update events that carry the saved draft.

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
      image_upload_id?: string;
      tags: string[];
      publish: boolean;
      authorId: string;
      external_url?: string | null;
      autosave?: boolean;
      snapshot?: string;
    }) => createArticle({
      title: data.title,
      content: data.content,
      image_url: data.image_url,
      image_upload_id: data.image_upload_id,
      tags: data.tags,
      publish: data.publish,
      authorId: data.authorId,
      external_url: data.external_url,
    }),
    onSuccess: (response, variables) => {
      queryClient.invalidateQueries({ queryKey: ['articles'] });
      if (variables.autosave && response.article.slug) {
        const latest = getFormValuesRef.current?.();
        if (latest && variables.snapshot && draftSnapshot(latest) !== variables.snapshot) {
          rememberPendingDraft(response.article.slug, latest);
        }
        try {
          sessionStorage.setItem('blog-editor-autosaved', response.article.slug);
        } catch {
          // The edit page still loads the saved draft.
        }
        navigate({
          to: `/dashboard/blog/edit/${response.article.slug}`,
          replace: true,
        });
        return;
      }
      toast({ title: "Success", description: "Article created successfully." });
      if (response.article.external_url && response.article.slug) {
        navigate({ to: `/dashboard/blog/edit/${response.article.slug}` });
      } else {
        navigate({ to: '/dashboard/blog' });
      }
    },
    onError: (error, variables) => {
      console.error('Error creating article:', error);
      if (variables.autosave && variables.snapshot) {
        failedSnapshotRef.current = variables.snapshot;
        setAutosaveStatus('error');
      }
      const errorMessage = error instanceof Error ? error.message : "Failed to create article. Please try again.";
      toast({ title: "Error", description: errorMessage, variant: "destructive" });
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
        image_upload_id?: string;
        tags: string[];
        external_url?: string | null;
      };
      returnToDashboard?: boolean;
      autosave?: boolean;
      snapshot?: string;
    }) => updateArticle(data.slug, data.updateData),
    onSuccess: (response, variables) => {
      if (variables.autosave) {
        if (variables.snapshot) {
          savedSnapshotRef.current = variables.snapshot;
        }
        failedSnapshotRef.current = null;
        setAutosaveStatus('saved');
      } else {
        toast({ title: "Success", description: "Draft saved successfully." });
      }
      
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
      } else if (!variables.autosave) {
        const saved = response.article;
        const next = {
          title: saved.draft_title || '',
          content: saved.draft_content || '',
          image_url: saved.draft_image_url || '',
          external_url: saved.external_url || '',
          tags: variables.updateData.tags,
        };
        setValue('title', next.title);
        setValue('content', next.content);
        setValue('image_url', next.image_url);
        setValue('external_url', next.external_url);
        setValue('tags', next.tags);
        const snap = draftSnapshot(next);
        savedSnapshotRef.current = snap;
        catchUpSnapshotRef.current = snap;
        catchUpSeenRef.current = null;
        setDraftEpoch((epoch) => epoch + 1);
        if (saved.draft_image_url) {
          setStagedImageUrl(saved.draft_image_url);
          setPreviewImageUrl(saved.draft_image_url);
        }
      }
    },
    onError: (error, variables) => {
      console.error('Error updating article:', error);
      if (variables.autosave && variables.snapshot) {
        failedSnapshotRef.current = variables.snapshot;
        setAutosaveStatus('error');
      }
      const errorMessage = error instanceof Error ? error.message : "Failed to save draft. Please try again.";
      toast({ title: "Error", description: errorMessage, variant: "destructive" });
    }
  });

  // Mutation for publishing an article (copies draft to published)
  const publishMutation = useMutation({
    mutationFn: (publishedAt?: Date) => publishArticle(blogSlug as string, publishedAt),
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
      const errorMessage = error instanceof Error ? error.message : "Failed to unpublish article. Please try again.";
      toast({ title: "Error", description: errorMessage, variant: "destructive" });
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
      const revertedContent = response.article.draft_content || '';
      const reverted = {
        title: response.article.draft_title,
        content: revertedContent,
        image_url: response.article.draft_image_url || '',
        tags: tagNames,
        external_url: response.article.external_url || '',
      };
      reset(reverted);
      const revertedSnapshot = draftSnapshot(reverted);
      savedSnapshotRef.current = revertedSnapshot;
      catchUpSnapshotRef.current = revertedSnapshot;
      catchUpSeenRef.current = null;
      setDraftEpoch((epoch) => epoch + 1);
      
      setSelectedVersion(null);
      setShowVersions(false);
      setAddedRanges([]);
    },
    onError: (error) => {
      console.error('Error reverting to version:', error);
      const errorMessage = error instanceof Error ? error.message : "Failed to revert to version. Please try again.";
      toast({ title: "Error", description: errorMessage, variant: "destructive" });
    }
  });

  const { register, handleSubmit, setValue, getValues, formState: { errors }, control, reset } = useForm<ArticleFormData>({
    resolver: zodResolver(articleSchema),
    defaultValues: {
      title: '',
      content: '',
      image_url: '',
      tags: [],
      external_url: '',
    }
  });

  // Watch reactive form fields for UI updates
  const watchedTags = useWatch({ control, name: 'tags' });
  const watchedContent = useWatch({ control, name: 'content' });
  const watchedTitle = useWatch({ control, name: 'title' });
  const watchedExternalUrl = useWatch({ control, name: 'external_url' });
  const articleIsLive = isPublished(article?.article);
  const articleWordCount = (watchedContent ?? "")
    .trim()
    .split(/\s+/)
    .filter(Boolean).length;
  const articleWhen = articleIsLive ? article?.article.published_at : article?.article.updated_at;
  const articleWhenDate = articleWhen ? new Date(articleWhen) : null;
  const articleWhenLabel = articleWhenDate && !Number.isNaN(articleWhenDate.getTime())
    ? `${articleIsLive ? "Published" : "Updated"} ${format(articleWhenDate, "MMM d, yyyy")}`
    : null;
  const savePending = createArticleMutation.isPending || updateArticleMutation.isPending;
  const saveLabel = savePending ? (isNew ? 'Creating...' : 'Saving...') : 'Save';
  const autosaveLabel = autosaveStatus === 'saving' || savePending
    ? 'Saving…'
    : autosaveStatus === 'saved'
      ? 'Saved'
      : autosaveStatus === 'error'
        ? 'Not saved'
        : '';
  const watchedImageUrl = useWatch({ control, name: 'image_url' });
  const draftKey = draftSnapshot({
    title: watchedTitle,
    content: watchedContent,
    image_url: watchedImageUrl,
    tags: watchedTags,
    external_url: watchedExternalUrl,
  });
  getFormValuesRef.current = getValues;

  const [imagePrompt, setImagePrompt] = useState<string | null>(DEFAULT_IMAGE_PROMPT[Math.floor(Math.random() * DEFAULT_IMAGE_PROMPT.length)]);

  /* Highlights stay until the next agent run. */
  const [addedRanges, setAddedRanges] = useState<TextRange[]>([]);

  const onContentChange = (md: string) => {
    setValue('content', md);
  };

  useEffect(() => {
    const articleId = article?.article.id;
    if (!articleId) {
      setArticleSources([]);
      return;
    }
    let cancelled = false;
    getArticleSources(articleId)
      .then((sources) => {
        if (cancelled) return;
        setArticleSources(sources.map((source) => ({
          id: source.id,
          title: source.title,
          url: source.url,
        })));
      })
      .catch(() => {
        if (!cancelled) setArticleSources([]);
      });
    return () => {
      cancelled = true;
    };
  }, [article?.article.id, sourcesReloadToken]);

  const applyAgentMarkdown = (next: string) => {
    const previous = getValues('content') || '';
    setValue('content', next);
    setAddedRanges(addedTextRanges(previous, next));
  };

  if (!user) {
    return <div>Please log in to edit articles.</div>;
  }

  // Sync stagedImageUrl to form; add new URLs to versions
  useEffect(() => {
    if (stagedImageUrl !== undefined) {
      setValue('image_url', stagedImageUrl ?? '');
      setPreviewImageUrl(stagedImageUrl ?? '');
      if (stagedImageUrl && !imageVersions.some(v => v.url === stagedImageUrl)) {
        addImageVersion(stagedImageUrl);
      }
    }
  }, [stagedImageUrl, setValue, imageVersions]);

  // Populate form when article data is loaded (always load draft_* fields for editing)
  useEffect(() => {
    if (article && !isNew) {
      if (hydratedArticleIdRef.current === article.article.id) {
        return;
      }
      hydratedArticleIdRef.current = article.article.id;
      try {
        if (sessionStorage.getItem('blog-editor-autosaved') === article.article.slug) {
          sessionStorage.removeItem('blog-editor-autosaved');
          setAutosaveStatus('saved');
        }
      } catch {
        // Status text is optional.
      }
      // Extract tag names from the server response format
      const tagNames = article.tags ? article.tags
        .map((tag: any) => tag?.name?.toUpperCase())
        .filter((name: string | undefined) => !!name && name !== '') : [];
      const loadedContent = article.article.draft_content || '';
      const newValues = {
        title: article.article.draft_title || '',
        content: loadedContent,
        image_url: article.article.draft_image_url || '',
        tags: tagNames,
        external_url: article.article.external_url || '',
      } as ArticleFormData;
      const pending = takePendingDraft(article.article.slug);
      const formValues = pending ?? newValues;
      reset(formValues);
      savedSnapshotRef.current = draftSnapshot(newValues);
      catchUpSnapshotRef.current = draftSnapshot(formValues);
      catchUpSeenRef.current = null;
      setDraftEpoch((epoch) => epoch + 1);
      
      // Initialize image versions and staged state if there's an existing image
      if (article.article.draft_image_url) {
        setImageVersions([{
          url: article.article.draft_image_url,
          timestamp: Date.now(),
          uploadId: article.article.draft_image?.id ?? article.article.draft_upload_file_id ?? undefined,
          blurhash: article.article.draft_image?.blurhash,
        }]);
        setCurrentVersionIndex(0);
        setPreviewImageUrl(article.article.draft_image_url);
        setStagedImageUrl(article.article.draft_image_url);
      } else {
        setImageVersions([]);
        setCurrentVersionIndex(-1);
        setPreviewImageUrl('');
        setStagedImageUrl(undefined);
      }
      
      // Content is now markdown -- form value is set via reset() above
    } else if (isNew) {
      hydratedArticleIdRef.current = null;
      const blank: ArticleFormData = {
        title: '',
        content: '',
        image_url: '',
        tags: [],
        external_url: '',
      };
      reset(blank);
      const blankSnapshot = draftSnapshot(blank);
      savedSnapshotRef.current = blankSnapshot;
      catchUpSnapshotRef.current = blankSnapshot;
      catchUpSeenRef.current = null;
      setDraftEpoch((epoch) => epoch + 1);
      setImageVersions([]);
      setCurrentVersionIndex(-1);
      setPreviewImageUrl('');
    }
  }, [article, isNew, reset]);

  // Older articles gain their blurhash on the upload row after generation.
  // Keep the open header in sync when that payload arrives after first paint.
  useEffect(() => {
    const savedHash = article?.article.draft_image?.blurhash;
    const savedId = article?.article.draft_image?.id ?? article?.article.draft_upload_file_id ?? undefined;
    const savedUrl = article?.article.draft_image?.url || article?.article.draft_image_url;
    if (!savedUrl || (!savedHash && !savedId)) {
      return;
    }
    setImageVersions((prev) => {
      let changed = false;
      const next = prev.map((version) => {
        if (version.url !== savedUrl && version.url !== article?.article.draft_image_url) {
          return version;
        }
        if (version.blurhash && version.uploadId) {
          return version;
        }
        changed = true;
        return {
          ...version,
          blurhash: version.blurhash ?? savedHash,
          uploadId: version.uploadId ?? savedId,
        };
      });
      return changed ? next : prev;
    });
  }, [article]);

  // Load conversation history with artifacts when article is loaded.
  // Generation handoff owns the initial requestId load so it can append the
  // streaming assistant placeholder without a competing history refresh.
  useEffect(() => {
    if (initialRequestId) {
      return;
    }

    if (article?.article?.id && !isNew) {
      const loadConversations = async () => {
        try {
          const result = await getConversationHistory(article.article.id);
          const loadedMessages = mapConversationMessages(result.messages || []);
          if (launchpadSessionRef.current && loadedMessages.length === 0) {
            return;
          }
          setChatMessages(
            loadedMessages.length > 0 || initialRequestId
              ? loadedMessages
              : [getInitialGreetingMessage()]
          );
        } catch (error) {
          console.error('[Editor] Failed to load conversation history:', error);
          setChatMessages(initialRequestId ? [] : [getInitialGreetingMessage()]);
        }
      };
      
      loadConversations();
    } else if (isNew) {
      setChatMessages([]);
    }
  }, [article?.article?.id, isNew, initialRequestId]);

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
        if (createArticleMutation.isPending || updateArticleMutation.isPending) {
          return;
        }
        handleSubmit((data) => onSubmit(data, false))();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleSubmit, createArticleMutation.isPending, updateArticleMutation.isPending]);

  const onSubmit = async (data: ArticleFormData, returnToDashboard: boolean = true, autosave = false) => {
    if (!user) {
      toast({ title: "Error", description: "You must be logged in to edit an article." });
      return;
    }

    // Ensure image_url is captured at submit time - prefer stagedImageUrl when set, else form value
    const formImageUrl = getValues('image_url') ?? data.image_url ?? '';
    const finalImageUrl = (stagedImageUrl !== undefined && stagedImageUrl !== null)
      ? stagedImageUrl
      : formImageUrl;

    const externalUrl = data.external_url?.trim() || null;
    const snapshot = draftSnapshot({
      ...data,
      image_url: finalImageUrl || '',
    });

    if (isNew) {
      // New articles are created as drafts by default
      // Use publishArticle() separately to publish
      createArticleMutation.mutate({
        title: data.title,
        content: data.content,
        image_url: finalImageUrl || undefined,
        image_upload_id: currentHeader?.uploadId,
        tags: data.tags,
        publish: false, // Save as draft, publish is a separate action
        authorId: String(user.id),
        external_url: externalUrl,
        autosave,
        snapshot,
      });
    } else {
      // Updates always go to draft_* fields
      // Use publishArticle() separately to publish
      const updateData = {
        title: data.title,
        content: data.content, // Markdown content
        image_url: finalImageUrl || undefined,
        image_upload_id: currentHeader?.uploadId,
        tags: data.tags,
        external_url: externalUrl,
      };
      
      updateArticleMutation.mutate({
        slug: blogSlug as string,
        updateData,
        returnToDashboard: autosave ? false : returnToDashboard,
        autosave,
        snapshot,
      });
    }
  };

  const onSubmitRef = useRef(onSubmit);
  onSubmitRef.current = onSubmit;

  useEffect(() => {
    if (catchUpSnapshotRef.current !== null && draftKey !== catchUpSnapshotRef.current) {
      if (catchUpSeenRef.current === null || catchUpSeenRef.current === draftKey) {
        catchUpSeenRef.current = draftKey;
        return;
      }
    }
    catchUpSnapshotRef.current = null;
    catchUpSeenRef.current = null;
    if (savedSnapshotRef.current === null) {
      return;
    }
    if (draftKey === savedSnapshotRef.current || draftKey === failedSnapshotRef.current) {
      return;
    }
    if (createArticleMutation.isPending || updateArticleMutation.isPending) {
      return;
    }

    setAutosaveStatus('idle');
    const timer = window.setTimeout(() => {
      let submitted = false;
      setAutosaveStatus('saving');
      void handleSubmit((data) => {
        submitted = true;
        return onSubmitRef.current(data, false, true);
      })().finally(() => {
        if (!submitted) {
          setAutosaveStatus('idle');
        }
      });
    }, AUTOSAVE_DELAY_MS);
    return () => window.clearTimeout(timer);
  }, [
    draftKey,
    draftEpoch,
    handleSubmit,
    createArticleMutation.isPending,
    updateArticleMutation.isPending,
  ]);

  const rewriteArticle = async () => {
    if (!article?.article.id) return;
    setAddedRanges([]);
    setGeneratingRewrite(true);
    try {
      const result = await updateArticleWithContext(article.article.id);
      
      if (result.success && result.content) {
        applyAgentMarkdown(result.content);
        setChatMessages((prev) => [
          ...prev,
          { role: 'assistant', content: 'I updated the draft. New text is highlighted in the editor. Use History to restore an earlier version.' }
        ]);
      }
    } catch (error) {
      console.error('Error rewriting article:', error);
      toast({ title: 'Error', description: 'Failed to rewrite article. Please try again.' });
    } finally {
      setGeneratingRewrite(false);
    }
  };

  const clearChat = async () => {
    if (!article?.article?.id) return;
    setClearingChat(true);
    try {
      await clearConversationHistory(article.article.id);
      setChatMessages([getInitialGreetingMessage()]);
      toast({
        title: "Chat cleared",
        description: "Your conversation history has been reset.",
      });
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
  };

  const sendChatWithMessage = async (message: string) => {
    const text = message.trim();
    
    if (!text) {
      return;
    }

    // Get current document content in both HTML and markdown formats
    // Content is now markdown -- no HTML-to-markdown conversion needed
    const currentContent = getValues('content') || '';
    const currentMarkdown = currentContent;

    // Check if this looks like an edit request
    const isEditRequest = /\b(rewrite|edit|improve|change|update|fix|enhance|modify)\b/i.test(text);

    // Show original user message in UI
    const baseMessages = [...chatMessages, { role: 'user', content: text, channel: inputMode } as ChatMessage];
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
    
    if (!text || (launchpad && !article?.article?.id)) {
      return;
    }

    if (inputMode === 'conversation') {
      setChatInput('');
      sendLiveTextRef.current(text);
      return;
    }

    await sendChatWithMessage(text);
  };

  const beginFromLanding = async (mode: 'send' | 'voice', sources: AttachedSource[] = []) => {
    if (!launchpad || launching) return;
    const text = chatInput.trim();
    if (mode === 'send' && !text) return;
    if (!user?.id) {
      toast({
        title: "Error",
        description: "User not found. Please log in again.",
        variant: "destructive",
      });
      return;
    }

    const token = ++launchTokenRef.current;
    launchpadSessionRef.current = true;
    setLaunching(true);
    setMobileChatOpen(true);

    if (mode === 'voice') {
      setInputMode('conversation');
    } else {
      const seed = [{ role: 'user', content: text, channel: 'text' } as ChatMessage];
      launchMessagesRef.current = seed;
      setChatMessages(seed);
      setChatInput('');
      setIsThinking(true);
      setThinkingMessage('Starting…');
    }

    setLaunchPhase('editor');

    const attachSources = async (articleId: string) => {
      if (sources.length === 0) return;
      await Promise.all(sources.map((source) =>
        scrapeAndCreateSource({
          article_id: articleId,
          url: source.url,
        }).catch((err) => {
          console.error(`Failed to scrape source ${source.url}:`, err);
          return null;
        })
      ));
    };

    try {
      if (mode === 'voice') {
        const created = await createArticle({
          title: 'Untitled',
          content: '',
          tags: [],
          publish: false,
          authorId: String(user.id),
        });
        if (launchTokenRef.current !== token) return;
        await attachSources(String(created.article.id));
        if (launchTokenRef.current !== token) return;
        queryClient.setQueryData(['article', created.article.slug], created);
        setLaunchedSlug(created.article.slug);
        return;
      }

      const { article, request_id } = await generateArticle(text);
      if (launchTokenRef.current !== token) return;
      await attachSources(String(article.id));
      if (launchTokenRef.current !== token) return;
      setLaunchedRequestId(request_id);
      setLaunchedSlug(article.slug);
    } catch (err) {
      if (launchTokenRef.current !== token) return;
      console.error('Failed to start writing:', err);
      launchpadSessionRef.current = false;
      launchMessagesRef.current = null;
      setLaunchPhase('landing');
      setInputMode('text');
      setIsThinking(false);
      setChatMessages([]);
      if (mode === 'send') setChatInput(text);
      toast({
        title: "Error",
        description: mode === 'voice'
          ? "Could not start voice. Please try again."
          : "Failed to start the article. Please try again.",
        variant: "destructive",
      });
    } finally {
      if (launchTokenRef.current === token) setLaunching(false);
    }
  };

  const performChatRequest = async (messageText: string, assistantIndex: number, isEditRequest: boolean, documentContent: string, documentMarkdown?: string) => {
    setChatLoading(true);
    try {
      if (!article?.article?.id) {
        throw new Error('Article ID is required');
      }
      
      const result = await submitAgentRequest({
            message: messageText,
            documentContent: documentContent,
            documentMarkdown: documentMarkdown || '',
            documentTitle: getValues('title') || '',
            articleId: article.article.id,
            channel: 'text',
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
    setAddedRanges([]);
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
              case 'turn_started':
                break;

              case 'transcript':
                if (msg.content) {
                  setChatMessages((prev) => {
                    const lastUser = [...prev].reverse().find((item) => item.role === 'user');
                    if (lastUser && lastUser.content === msg.content) {
                      return prev;
                    }
                    return prev;
                  });
                }
                break;

              case 'speech':
                if (msg.data?.audioBase64) {
                  void playSpeechRef.current(msg.data.audioBase64, msg.data.mimeType);
                }
                break;

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
                setIsThinking(false);
                if (msg.tool_name) {
                  const toolId = msg.tool_id || `tool-${Date.now()}`;
                  console.debug('[Agent] tool_use:', msg.tool_name, toolId);
                  setChatMessages((prev) => {
                    const updated = [...prev];
                    if (!updated[assistantIndex]) return updated;
                    const steps = [...(updated[assistantIndex].steps || [])];
                    // Mark prior reasoning as done
                    for (let i = 0; i < steps.length; i++) {
                      if (steps[i].type === 'reasoning' && steps[i].isStreaming) {
                        steps[i] = { ...steps[i], isStreaming: false };
                      }
                    }
                    // Push new tool step with tool_id for later matching
                    steps.push({
                      type: 'tool_group',
                      toolGroup: {
                        group_id: toolId,
                        status: 'running',
                        calls: [{
                          id: toolId,
                          name: msg.tool_name,
                          input: msg.tool_input as Record<string, unknown> || {},
                          status: 'running',
                          started_at: new Date().toISOString(),
                        }],
                      },
                    });
                    updated[assistantIndex] = { ...updated[assistantIndex], steps, isReasoningStreaming: false };
                    return updated;
                  });
                }
                break;
                
              case 'document_update':
                if (typeof msg.content === 'string') {
                  applyAgentMarkdown(msg.content);
                }
                break;

              case 'title_update':
                if (typeof msg.content === 'string') {
                  setValue('title', msg.content, { shouldDirty: true, shouldValidate: true });
                }
                break;

              case 'sources_update': {
                const next = Array.isArray(msg.data?.sources) ? msg.data.sources : [];
                setArticleSources(next.map((source: { id?: string; title?: string; url?: string }) => ({
                  id: String(source.id || source.url || source.title || Math.random()),
                  title: source.title || source.url || 'Source',
                  url: source.url || '',
                })));
                setSourcesReloadToken((token) => token + 1);
                break;
              }

              case 'tool_result':
                setIsThinking(false);
                if (msg.tool_result) {
                  const toolId = msg.tool_id;

                  try {
                    const toolResult = msg.tool_result.content ? JSON.parse(msg.tool_result.content) : {};
                    const toolName = msg.tool_name || toolResult.tool_name || '';
                    const isError = toolResult.is_error || msg.tool_result.is_error;

                    console.debug('[Agent] tool_result:', toolName, toolId, isError ? 'ERROR' : 'OK');

                    // Match by tool_id (not name), scan ALL steps, no status filter
                    setChatMessages((prev) => {
                      const updated = [...prev];
                      if (!updated[assistantIndex]?.steps) return updated;
                      const steps = [...updated[assistantIndex].steps!];
                      for (let i = 0; i < steps.length; i++) {
                        if (steps[i].type !== 'tool_group' || !steps[i].toolGroup) continue;
                        const calls = [...steps[i].toolGroup!.calls];
                        let found = false;
                        for (let j = 0; j < calls.length; j++) {
                          if (calls[j].id === toolId) {
                            calls[j] = {
                              ...calls[j],
                              status: isError ? 'error' : 'completed',
                              result: toolResult,
                              error: isError ? (toolResult.error || 'Failed') : undefined,
                              completed_at: new Date().toISOString(),
                            };
                            found = true;
                            break;
                          }
                        }
                        if (found) {
                          const groupStatus = calls.some((c: any) => c.status === 'error') ? 'error' :
                                              calls.every((c: any) => c.status === 'completed' || c.status === 'error') ? 'completed' : 'running';
                          steps[i] = {
                            ...steps[i],
                            toolGroup: { ...steps[i].toolGroup!, status: groupStatus, calls },
                          };
                          break;
                        }
                      }
                      updated[assistantIndex] = { ...updated[assistantIndex], steps };
                      return updated;
                    });
                  } catch (e) {
                    console.error('[Editor] Failed to parse tool result:', e);
                  }
                }
                break;

              case 'tool_group_complete':
                // No-op: tool_result is the source of truth for call status.
                // This event arrives BEFORE tool_result from the backend,
                // so we must NOT mark calls as completed here (that would
                // prevent tool_result from attaching result data).
                setIsThinking(false);
                break;

              case 'full_message':
                // Only merge metadata -- never overwrite streaming steps
                setIsThinking(false);
                if (msg.full_message) {
                  setChatMessages((prev) => {
                    const updated = [...prev];
                    if (!updated[assistantIndex]) return updated;
                    const existing = updated[assistantIndex];
                    updated[assistantIndex] = {
                      ...existing,
                      id: msg.full_message.id,
                      meta_data: msg.full_message.meta_data,
                      created_at: msg.full_message.created_at,
                      // Keep streaming steps -- do NOT overwrite
                      isReasoningStreaming: false,
                    };
                    return updated;
                  });
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
            
            // Legacy done signal -- no diff handling needed
            
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

  const conversation = useConversation({
    enabled: inputMode === 'conversation' && !isNew && Boolean(article?.article?.id),
    articleId: article?.article?.id,
    getDocument: () => {
      const content = getValues('content') || '';
      return { content, markdown: content };
    },
    onUserTranscript: (text) => {
      setChatMessages((prev) => {
        const index = liveUserIndexRef.current;
        if (index != null && prev[index]?.role === 'user') {
          const next = [...prev];
          next[index] = { ...next[index], content: text, channel: 'voice' };
          return next;
        }
        liveUserIndexRef.current = prev.length;
        return [...prev, { role: 'user', content: text, channel: 'voice' } as ChatMessage];
      });
    },
    onDelegation: (requestId, message) => {
      setChatMessages((prev) => {
        const index = liveUserIndexRef.current;
        let next = [...prev];
        if (index != null && next[index]?.role === 'user') {
          next[index] = { ...next[index], content: message, channel: 'voice' };
        } else {
          next = [...next, { role: 'user', content: message, channel: 'voice' } as ChatMessage];
        }
        liveUserIndexRef.current = null;
        pendingLiveStreamRef.current = { requestId, assistantIndex: next.length };
        return [...next, { role: 'assistant', content: '' } as ChatMessage];
      });
    },
  });
  sendLiveTextRef.current = conversation.sendText;

  useEffect(() => {
    const pending = pendingLiveStreamRef.current;
    if (!pending || startedLiveRequestsRef.current.has(pending.requestId)) {
      return;
    }
    pendingLiveStreamRef.current = null;
    startedLiveRequestsRef.current.add(pending.requestId);
    setChatLoading(true);
    void streamChatResponse(pending.requestId, pending.assistantIndex, false)
      .catch((streamError: unknown) => {
        toast({
          title: "Conversation error",
          description: streamError instanceof Error ? streamError.message : "Could not run the live turn",
          variant: "destructive",
        });
      })
      .finally(() => setChatLoading(false));
  });

  // Auto-subscribe to an in-flight generation session when arriving from /blog/generate.
  const consumedRequestIdRef = useRef<string | null>(null);
  useEffect(() => {
    if (!initialRequestId || isNew) return;
    if (!article?.article?.id) return;
    if (consumedRequestIdRef.current === initialRequestId) return;
    consumedRequestIdRef.current = initialRequestId;

    setMobileChatOpen(true);
    setChatLoading(true);

    (async () => {
      try {
        let loadedMessages: ChatMessage[] = [];
        try {
          const result = await getConversationHistory(article.article.id);
          loadedMessages = mapConversationMessages(result.messages || []);
        } catch (historyErr) {
          console.error('[Editor] Failed to load generation conversation history:', historyErr);
        }

        const seeded = loadedMessages.length > 0
          ? loadedMessages
          : (launchMessagesRef.current ?? []);
        const last = seeded[seeded.length - 1];
        if (last?.role === 'assistant' && last.content.trim()) {
          setChatMessages(seeded);
          setIsThinking(false);
          return;
        }
        const assistantIndex = last?.role === 'assistant' ? seeded.length - 1 : seeded.length;
        setChatMessages(
          last?.role === 'assistant'
            ? seeded
            : [...seeded, { role: 'assistant', content: '' } as ChatMessage],
        );
        await streamChatResponse(initialRequestId, assistantIndex, false);
      } catch (err) {
        console.error('[Editor] Failed to attach to generation session:', err);
      } finally {
        setChatLoading(false);
        // Strip requestId from URL so refresh doesn't re-attach.
        // The launchpad stays on this screen so the chat never remounts.
        if (!launchpad) {
          navigate({
            to: `/dashboard/blog/edit/${blogSlug}`,
            replace: true,
          });
        }
      }
    })();
  }, [initialRequestId, article?.article?.id, isNew]);

  // Show loading state while fetching article
  if (articleLoading && !isNew && !launchpad) {
    return (
      <section className="flex-1 p-0 md:p-4">
        <div className="flex items-center justify-center h-64">
          <div>Loading article...</div>
        </div>
      </section>
    );
  }

  // Show error state if fetch failed
  if (error && !isNew && !launchpad) {
    return (
      <section className="flex-1 p-0 md:p-4">
        <div className="flex items-center justify-center h-64">
          <div>Error loading article. Please try again.</div>
        </div>
      </section>
    );
  }

  const voiceStatus = conversation.error
    ? conversation.error
    : conversation.caption
      ? conversation.caption
      : conversation.state === 'connecting'
        ? 'Connecting to GPT-Live…'
        : conversation.state === 'speaking'
          ? 'Speaking…'
          : 'Voice on · talk, or paste a link';

  return (
      <motion.section
        layout
        data-writing-phase={launchpad ? launchPhase : 'editor'}
        transition={{ layout: { duration: 0.55, ease: [0.22, 1, 0.36, 1] } }}
        className={cn(
          landing ? "writing-landing" : "article-editor-shell",
          launchpad && !landing && "writing-launch-editor",
        )}
      >
        <AnimatePresence>
          {landing && (
            <motion.div
              key="writing-intro"
              className="writing-landing-intro"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.35 }}
            >
              <h1 className="font-wordmark text-3xl font-medium tracking-tight text-balance md:text-4xl">
                What are we writing today?
              </h1>
            </motion.div>
          )}
        </AnimatePresence>
        {!landing && (
        <motion.div
          className="article-editor-main px-2 pt-2 md:px-0 md:pt-0"
          initial={launchpad ? { opacity: 0 } : false}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.4, delay: launchpad ? 0.08 : 0 }}
        >
        {/* Article Metadata Card */}
        
                <Dialog open={imageModalOpen} onOpenChange={setImageModalOpen}>
                  <DialogContent className="sm:max-w-4xl">
                    <DialogHeader>
                      <DialogTitle>Edit Header Image</DialogTitle>
                      <DialogDescription>Update or generate a header image for your article.</DialogDescription>
                    </DialogHeader>
                    
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                      {/* Image Preview Section */}
                      <div className="space-y-4">
                        <div className="text-sm font-medium">Preview</div>
                        <div className="aspect-video rounded-lg border-2 border-dashed border-border overflow-hidden bg-muted/40">
                          {previewImageUrl ? (
                            <BlurhashImage
                              src={previewImageUrl}
                              alt="Image preview"
                              blurhash={currentHeader?.blurhash}
                              className="h-full w-full"
                              imgClassName="h-full w-full object-cover"
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
                        {previewImageUrl ? (
                          <GenerateBlurhash
                            fileKey={storageKeyFromUrl(previewImageUrl)}
                            uploadId={currentHeader?.uploadId}
                            blurhash={currentHeader?.blurhash}
                            testId="header-blurhash"
                            onGenerated={(result) => {
                              setImageVersions((prev) => prev.map((version, index) => (
                                index === currentVersionIndex
                                  ? {
                                      ...version,
                                      blurhash: result.blurhash,
                                      uploadId: result.id ?? version.uploadId,
                                    }
                                  : version
                              )));
                              if (blogSlug) {
                                void queryClient.invalidateQueries({ queryKey: ['article', blogSlug] });
                                void queryClient.invalidateQueries({ queryKey: ['articles'] });
                              }
                            }}
                          />
                        ) : null}
                        
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
                                      ? "border-primary ring-2 ring-primary/30" 
                                      : "border-border hover:border-primary/50"
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
                        <Tabs defaultValue="uploads" className="w-full">
                          <TabsList className="grid w-full grid-cols-2">
                            <TabsTrigger value="uploads">From Uploads</TabsTrigger>
                            <TabsTrigger value="generate">Generate</TabsTrigger>
                          </TabsList>
                          <TabsContent value="generate" className="space-y-3 mt-3">
                            <div className="text-sm font-medium">Generate New Image</div>
                            <div className="flex items-center gap-2">
                              <Dialog open={generateImageOpen} onOpenChange={setGenerateImageOpen}>
                                <DialogTrigger asChild>
                                  <Button variant="outline" className="flex-1">
                                    <PencilIcon className="w-4 h-4 mr-2 text-primary" /> 
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
                                <SparklesIcon className={cn('w-4 h-4 text-primary', generatingImage && 'animate-spin')} />
                              </Button>
                            </div>
                          </TabsContent>
                          <TabsContent value="uploads" className="mt-3">
                            <ImagePickerFromUploads
                              onSelect={(file) => {
                                addImageVersion(file.url, undefined, { uploadId: file.id, blurhash: file.blurhash });
                                setImageModalOpen(false);
                              }}
                            />
                          </TabsContent>
                        </Tabs>
                        
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
            {article?.article.id && (
              <SourcesManager
                articleId={article.article.id}
                isOpen={resourcesOpen}
                onOpenChange={setResourcesOpen}
                reloadToken={sourcesReloadToken}
              />
            )}
            <Drawer direction="right" open={tagsOpen} onOpenChange={setTagsOpen}>
              <DrawerContent className="ml-auto w-full data-[vaul-drawer-direction=right]:w-full sm:max-w-sm sm:data-[vaul-drawer-direction=right]:max-w-sm">
                <DrawerHeader>
                  <DrawerTitle>Edit Tags</DrawerTitle>
                  <DrawerDescription>Add or remove tags for your article.</DrawerDescription>
                </DrawerHeader>
                <div className="space-y-4 px-4">
                  <div className="space-y-2">
                    <label className="block text-md font-medium leading-6 text-foreground">Article Tags</label>
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
            <Drawer direction="right" open={externalOpen} onOpenChange={setExternalOpen}>
              <DrawerContent className="ml-auto w-full data-[vaul-drawer-direction=right]:w-full sm:max-w-sm sm:data-[vaul-drawer-direction=right]:max-w-sm">
                <DrawerHeader>
                  <DrawerTitle>External link</DrawerTitle>
                  <DrawerDescription>
                    Point this article at a post published somewhere else. Readers open that link in a new tab. Clear the link and save to keep the article on this site.
                  </DrawerDescription>
                </DrawerHeader>
                <div className="space-y-2 px-4">
                  <label htmlFor="external-article-url" className="text-sm font-medium">
                    Article link
                  </label>
                  <Input
                    id="external-article-url"
                    type="url"
                    placeholder="https://example.com/blog/post"
                    {...register('external_url')}
                  />
                  {errors.external_url && (
                    <p className="text-sm text-destructive">{errors.external_url.message}</p>
                  )}
                  <p className="text-xs text-muted-foreground">
                    Saving with a new link stores it, and fills in a title, preview, and cover when those are still empty.
                  </p>
                </div>
                <DrawerFooter>
                  <DrawerClose asChild>
                    <Button variant="outline" className="w-full">Done</Button>
                  </DrawerClose>
                </DrawerFooter>
              </DrawerContent>
            </Drawer>
            <Drawer direction="right" open={publishDrawerOpen} onOpenChange={setPublishDrawerOpen}>
              <PublishDrawerContent
                article={article}
                isNew={isNew}
                publishMutation={publishMutation}
                unpublishMutation={unpublishMutation}
              />
            </Drawer>

          <form
            className="flex-1 flex flex-col min-h-0 min-w-0"
            onSubmit={(event) => event.preventDefault()}
          >
              <div className="flex-1 flex flex-col border border-border rounded-sm min-h-0 min-w-0">
                <TipTapEditor
                  content={watchedContent || ''}
                  onChange={onContentChange}
                  highlights={addedRanges}
                  title={watchedTitle}
                  onTitleChange={(value) => setValue('title', value, { shouldDirty: true, shouldValidate: true })}
                  titleError={errors.title?.message}
                  authorName={user?.name}
                  imageUrl={previewImageUrl}
                  imageBlurhash={currentHeader?.blurhash ?? article?.article.draft_image?.blurhash}
                  onEditImage={() => setImageModalOpen(true)}
                  onDropCover={dropCoverImage}
                  onUploadBodyImage={uploadBodyImage}
                  sources={articleSources}
                  tags={watchedTags}
                  meta={(
                    <div className="article-editor-meta">
                      <span
                        className={cn(
                          "inline-flex h-5 shrink-0 items-center rounded-none border px-1.5 text-[10px] font-semibold uppercase tracking-wide",
                          articleIsLive
                            ? "border-primary/50 bg-primary/10 text-primary"
                            : "border-border bg-muted text-foreground",
                        )}
                        data-testid="article-status-badge"
                      >
                        {articleIsLive ? "Live" : "Draft"}
                      </span>
                      <span className="truncate font-mono">
                        {article?.article.slug ? `/${article.article.slug}` : "Not saved"}
                      </span>
                      {articleWordCount > 0 && (
                        <span className="shrink-0">{articleWordCount.toLocaleString()} words</span>
                      )}
                      {articleWhenLabel && (
                        <span className="shrink-0">{articleWhenLabel}</span>
                      )}
                      {articleIsLive && hasDraftChanges(article?.article) && (
                        <span className="shrink-0 text-primary">Unpublished edits</span>
                      )}
                      {watchedExternalUrl?.trim() && (
                        <span className="shrink-0">External</span>
                      )}
                    </div>
                  )}
                  mobileControls={(
                    <>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="size-8 text-primary"
                        aria-label={saveLabel}
                        title={autosaveLabel || saveLabel}
                        onClick={() => {
                          handleSubmit((data) => onSubmit(data, false))();
                        }}
                        disabled={savePending}
                      >
                        <Save className="h-4 w-4" />
                      </Button>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            type="button"
                            variant="outline"
                            size="icon"
                            className="size-8"
                            aria-label="Article options"
                          >
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-52">
                          <DropdownMenuItem
                            disabled={!article?.article.id}
                            onSelect={() => window.setTimeout(() => setResourcesOpen(true), 0)}
                          >
                            <BookOpen />
                            Resources
                          </DropdownMenuItem>
                          <DropdownMenuItem onSelect={() => window.setTimeout(() => setTagsOpen(true), 0)}>
                            <Tag />
                            Tags
                            {watchedTags && watchedTags.length > 0 && (
                              <span className="ml-auto text-xs text-muted-foreground">{watchedTags.length}</span>
                            )}
                          </DropdownMenuItem>
                          <DropdownMenuItem onSelect={() => window.setTimeout(() => setExternalOpen(true), 0)}>
                            <ExternalLinkIcon />
                            External
                            {watchedExternalUrl?.trim() && (
                              <span className="ml-auto text-xs text-muted-foreground">on</span>
                            )}
                          </DropdownMenuItem>
                          <DropdownMenuItem onSelect={() => window.setTimeout(() => setPublishDrawerOpen(true), 0)}>
                            {isPublished(article?.article) ? <Globe /> : <EyeOff />}
                            {isPublished(article?.article) ? "Published" : "Draft"}
                          </DropdownMenuItem>
                          {!isNew && isPublished(article?.article) && hasDraftChanges(article?.article) && (
                            <DropdownMenuItem onSelect={() => window.setTimeout(() => setPublishDrawerOpen(true), 0)}>
                              <RefreshCw />
                              Update published
                            </DropdownMenuItem>
                          )}
                          {!isNew && (
                            <DropdownMenuItem asChild>
                              <Link
                                to="/blog"
                                params={{ slug: article?.article.slug || '' }}
                                search={{ page: undefined, tag: undefined, search: undefined }}
                                target="_blank"
                              >
                                <ExternalLinkIcon />
                                View
                              </Link>
                            </DropdownMenuItem>
                          )}
                          {!isNew && (
                            <DropdownMenuItem onSelect={() => window.setTimeout(() => setShowVersions(true), 0)}>
                              <History />
                              History
                            </DropdownMenuItem>
                          )}
                          {!isNew && (
                            <DropdownMenuItem
                              disabled={generatingRewrite}
                              onSelect={() => rewriteArticle()}
                            >
                              <RefreshCw />
                              Regenerate
                            </DropdownMenuItem>
                          )}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </>
                  )}
                  sideControls={(
                    <>
                      <div className="article-editor-rail article-editor-rail-left" aria-label="Article actions">
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="relative"
                          aria-label="Resources"
                          title="Resources"
                          onClick={() => setResourcesOpen(true)}
                          disabled={!article?.article.id}
                        >
                          <BookOpen className="h-4 w-4" />
                        </Button>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="relative"
                          aria-label="Tags"
                          title="Tags"
                          onClick={() => setTagsOpen(true)}
                        >
                          <Tag className="h-4 w-4" />
                          {watchedTags && watchedTags.length > 0 ? <span className="article-editor-mark" /> : null}
                        </Button>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="relative"
                          aria-label="External link"
                          title="External link"
                          onClick={() => setExternalOpen(true)}
                        >
                          <ExternalLinkIcon className="h-4 w-4" />
                          {watchedExternalUrl?.trim() ? <span className="article-editor-mark" /> : null}
                        </Button>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="relative"
                          aria-label={isPublished(article?.article) ? "Published" : "Draft"}
                          title={isPublished(article?.article) ? "Published" : "Draft"}
                          onClick={() => setPublishDrawerOpen(true)}
                        >
                          {isPublished(article?.article) ? (
                            <Globe className="h-4 w-4" />
                          ) : (
                            <EyeOff className="h-4 w-4" />
                          )}
                          {isPublished(article?.article) && hasDraftChanges(article?.article) ? (
                            <span className="article-editor-mark" />
                          ) : null}
                        </Button>
                      </div>
                      <div className="article-editor-rail article-editor-rail-right" aria-label="Article actions">
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="article-editor-save"
                          aria-label={saveLabel}
                          title={autosaveLabel || saveLabel}
                          onClick={() => {
                            handleSubmit((data) => onSubmit(data, false))();
                          }}
                          disabled={savePending}
                        >
                          <Save className={cn('h-4 w-4', savePending && 'animate-pulse')} />
                        </Button>
                        {!isNew && (
                          <Button variant="ghost" size="icon" asChild>
                            <Link
                              to="/blog"
                              params={{ slug: article?.article.slug || '' }}
                              search={{ page: undefined, tag: undefined, search: undefined }}
                              target="_blank"
                              aria-label="View"
                              title="View"
                            >
                              <Eye className="h-4 w-4" />
                            </Link>
                          </Button>
                        )}
                        {!isNew && (
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            aria-label="History"
                            title="History"
                            onClick={() => setShowVersions(true)}
                          >
                            <History className="h-4 w-4" />
                          </Button>
                        )}
                        {!isNew && (
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            aria-label="Regenerate"
                            title="Regenerate"
                            onClick={rewriteArticle}
                            disabled={generatingRewrite}
                          >
                            <RefreshCw className={cn('h-4 w-4', generatingRewrite && 'animate-spin')} />
                          </Button>
                        )}
                      </div>
                    </>
                  )}
                />
                {errors.content && <p className="text-red-500">{errors.content.message}</p>}
              </div>
          </form>

      </motion.div>
        )}

      {/* Same chat surface: centered on the dashboard, docked beside the draft after send. */}
        <motion.div
          layout
          transition={{ layout: { duration: 0.55, ease: [0.22, 1, 0.36, 1] } }}
          className={cn(
            landing
              ? "writing-landing-chat"
              : cn("article-editor-chat border rounded-sm", mobileChatOpen && "article-editor-chat-open"),
          )}
        >
        {!landing && <div className="article-editor-chat-mobile-bar">
          <span className="text-sm font-medium">Assistant</span>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-8"
            onClick={() => setMobileChatOpen(false)}
            aria-label="Close chat"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>}
        <div
          ref={chatMessagesRef}
          className={cn(
            "flex-1 space-y-2 overflow-y-auto p-3 md:p-1.5",
            landing && "hidden",
          )}
        >
          {chatMessages.map((m, i) => {
            switch (m.role) {
              case 'tool': {
                // Tool-role messages are not rendered directly (handled via steps on assistant message)
                return null;
              }
              case 'assistant': {
                // Skip __DIFF_ACTIONS__ messages
                if (m.content === '__DIFF_ACTIONS__') {
                  return null;
                }
                
                // Render chain of thought steps (the ONLY rendering path for tool display)
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
                            const groupStatus = step.toolGroup.status === 'running' ? 'running' : 'completed';
                            return (
                              <ChainOfThoughtStep 
                                key={stepIdx} 
                                type="tool" 
                                status={groupStatus}
                                isLast={isLastStep}
                              >
                                <ToolGroupDisplay group={step.toolGroup} />
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
                      <div className="max-w-xs whitespace-pre-wrap rounded-lg px-2.5 py-1.5 text-sm bg-primary text-primary-foreground">
                        {m.channel === 'voice' && (
                          <span className="mr-1 inline-flex align-middle opacity-80">
                            <Mic className="h-3 w-3" />
                          </span>
                        )}
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
        <div className={cn(!landing && "p-2")}>
          <ChatComposer
            variant={landing ? "landing" : "panel"}
            value={chatInput}
            onValueChange={setChatInput}
            onSubmit={(sources) => {
              if (landing) {
                void beginFromLanding('send', sources);
                return;
              }
              void sendChat();
            }}
            onVoice={(sources) => {
              if (landing) {
                void beginFromLanding('voice', sources);
                return;
              }
              setInputMode((mode) => (mode === 'conversation' ? 'text' : 'conversation'));
            }}
            isLoading={landing ? launching : chatLoading}
            disabled={launching || (launchpad && !landing && !article?.article?.id)}
            placeholder={
              landing
                ? "Write an article about..."
                : inputMode === 'conversation'
                  ? "Talk with GPT-Live, or paste a link it should see…"
                  : "Ask the assistant or click a quick action above…"
            }
            voiceOn={inputMode === 'conversation'}
            voiceDisabled={landing ? launching : isNew || !article?.article?.id}
            voiceStatus={voiceStatus}
            accessory={!landing && !isNew && article?.article?.id ? (
              <Button
                type="button"
                size="sm"
                variant="ghost"
                className="h-7 shrink-0 px-2 text-xs"
                disabled={clearingChat}
                onClick={clearChat}
              >
                <Trash2 className="mr-1 h-3.5 w-3.5" />
                {clearingChat ? 'Clearing…' : 'Clear chat'}
              </Button>
            ) : undefined}
          />
          {landing && chatInput.length === 0 && !launching && (
            <WritingSuggestions onPick={setChatInput} />
          )}
        </div>
      </motion.div>
      {!landing && (
      <button
        type="button"
        className="article-editor-chat-fab"
        onClick={() => setMobileChatOpen(true)}
        aria-label="Open assistant chat"
      >
        <MessageSquare />
        {chatLoading ? "Working" : "Chat"}
      </button>
      )}

      {/* Version History Drawer */}
      <Drawer open={showVersions} onOpenChange={setShowVersions} direction="right">
        <DrawerContent className="ml-auto h-full w-full data-[vaul-drawer-direction=right]:w-full sm:max-w-md sm:data-[vaul-drawer-direction=right]:max-w-md">
          <DrawerHeader>
            <DrawerTitle>Version History</DrawerTitle>
            <DrawerDescription>
              Saved on the server before each agent edit. Revert restores that draft.
              {versionsData && (
                <span className="block mt-1">{versionsData.draft_count} drafts, {versionsData.published_count} published</span>
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
                No versions yet. Saving the draft stores one here.
              </div>
            ) : (
              <div className="space-y-2">
                {versionsData?.versions.map((version) => (
                  <div 
                    key={version.id} 
                    className="flex items-center justify-between py-3 px-3 border rounded-md hover:bg-accent"
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
              <pre className="max-h-64 overflow-y-auto whitespace-pre-wrap border rounded p-3 text-sm">
                {selectedVersion?.content}
              </pre>
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
    </motion.section>
  );
}
