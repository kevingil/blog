import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from '@tanstack/react-router';
import { useAuth } from '@/services/auth/auth';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { format } from "date-fns"
import { Calendar as CalendarIcon, PencilIcon, SparklesIcon, RefreshCw } from "lucide-react"
import { ExternalLinkIcon, UploadIcon } from '@radix-ui/react-icons';
import { IconLoader, IconLoader2 } from '@tabler/icons-react';
import { useQuery } from '@tanstack/react-query';
import { useEditor, EditorContent } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import MarkdownIt from 'markdown-it';
import { diffWords } from 'diff';
import type { Editor as TiptapEditor } from '@tiptap/core';
import { VITE_API_BASE_URL } from "@/services/constants";
 
import { Card, CardContent, CardFooter } from "@/components/ui/card";
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
  image: z.union([z.string().url(), z.literal('')]).optional(),
  tags: z.array(z.string()),
  isDraft: z.boolean(),
});

type ArticleFormData = z.infer<typeof articleSchema>;

// Minimal shape for the conversational side panel. Mirrors what our backend
// expects/returns.
type ChatMessage = {
  role: 'user' | 'assistant';
  content: string;
};

// === Helper utilities for Markdown ↔️ HTML and diff markup ===================
function fromMarkdown(text: string) {
  const md = new MarkdownIt({ typographer: true, html: true });
  return md.render(text);
}

function diffPartialText(
  oldText: string,
  newText: string,
  isComplete: boolean = false,
): string {
  let oldTextToCompare = oldText;
  if (oldText.length > newText.length && !isComplete) {
    oldTextToCompare = oldText.slice(0, newText.length);
  }

  const changes = diffWords(oldTextToCompare, newText);

  let result = '';
  changes.forEach((part: any) => {
    if (part.added) {
      result += `<em>${part.value}</em>`;
    } else if (part.removed) {
      result += `<s>${part.value}</s>`;
    } else {
      result += part.value;
    }
  });

  if (oldText.length > newText.length && !isComplete) {
    result += oldText.slice(newText.length);
  }

  return result;
}

// Simple overlay component to confirm/reject AI changes
function ConfirmChanges({ onReject, onConfirm }: { onReject: () => void; onConfirm: () => void; }) {
  return (
    <div className="absolute inset-0 flex items-center justify-center bg-white/80 dark:bg-gray-900/80 backdrop-blur-sm z-10">
      <div className="bg-white dark:bg-gray-800 p-6 rounded shadow-lg border border-gray-200 space-y-4">
        <h2 className="text-lg font-bold">Accept AI changes?</h2>
        <div className="flex justify-end space-x-4">
          <Button variant="outline" onClick={onReject}>Reject</Button>
          <Button onClick={onConfirm}>Confirm</Button>
        </div>
      </div>
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
    } else if (article && article.article.image) {
      setImageUrl(article.article.image);
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
  
  // Only use useParams when editing an existing article
  const params = !isNew ? useParams({ from: '/dashboard/blog/edit/$blogSlug' }) : null;
  const blogSlug = params?.blogSlug;
  
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [generatingImage, setGeneratingImage] = useState(false);
  const [newImageGenerationRequestId, setNewImageGenerationRequestId] = useState<string | null>(null);
  const [stagedImageUrl, setStagedImageUrl] = useState<string | null | undefined>(undefined);
  const [generateImageOpen, setGenerateImageOpen] = useState(false);
  const [generatingRewrite, setGeneratingRewrite] = useState(false);

  /* --------------------------------------------------------------------- */
  /* Chat (right-hand panel)                                               */
  /* --------------------------------------------------------------------- */
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([
    { role: 'assistant', content: 'Hi! I can help you improve your article. Try asking me to "rewrite the introduction" or "make the content more engaging".' },
  ]);
  const [chatLoading, setChatLoading] = useState(false);
  const [chatInput, setChatInput] = useState('');
  const chatInputRef = useRef<HTMLTextAreaElement>(null);
  
  // Document editing state
  const [pendingEdit, setPendingEdit] = useState<{
    newContent: string;
    summary: string;
  } | null>(null);

  // Use React Query to fetch article data
  const { data: article, isLoading: articleLoading, error } = useQuery({
    queryKey: ['article', blogSlug],
    queryFn: () => getArticle(blogSlug as string),
    enabled: !isNew && !!blogSlug,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });

  const { register, handleSubmit, setValue, formState: { errors }, watch, reset } = useForm<ArticleFormData>({
    resolver: zodResolver(articleSchema),
    defaultValues: {
      title: '',
      content: '',
      image: '',
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

  const editor = useEditor({
    extensions: [StarterKit],
    content: mdParser.render(watchedValues.content || ''),
    editorProps: {
      attributes: {
        class:
          'w-full p-4 border border-gray-300 rounded-md h-[calc(100vh-425px)] focus:outline-none',
      },
    },
    onUpdate({ editor }: { editor: TiptapEditor }) {
      if (!diffing) {
        setValue('content', editor.getText());
      }
    },
  });

  // When form values are externally reset (e.g., after fetching the article or clearing for a new one) we
  // synchronise those changes into the editor exactly once.
  useEffect(() => {
    if (!editor) return;
    // If the change came from user typing inside the editor, `editor.getText()` already matches `watchedValues.content`.
    // We only want to update when the two differ – i.e., an external change.
    if (watch('content') !== editor.getText()) {
      editor.commands.setContent(mdParser.render(watch('content') || ''));
    }
  }, [watch('content'), editor, mdParser]);

  if (!user) {
    return <div>Please log in to edit articles.</div>;
  }

  // Consume from ImageLoader
  useEffect(() => {
    if (stagedImageUrl) {
      setValue('image', stagedImageUrl);
    }
  }, [stagedImageUrl, setValue]);

  // Populate form when article data is loaded
  useEffect(() => {
    if (article && !isNew) {
      console.log("Populating form with article data:", article);
      // Extract tag names from the server response format
      const tagNames = article.tags ? article.tags.map((tag: any) => tag?.tag_name?.toUpperCase() || tag) : [];
      const newValues = {
        title: article.article.title || '',
        content: article.article.content || '',
        image: article.article.image || '',
        tags: tagNames,
        isDraft: article.article.is_draft,
      } as ArticleFormData;
      reset(newValues);
      // Sync editor with fresh content
      if (editor) {
        editor.commands.setContent(mdParser.render(newValues.content));
      }
    } else if (isNew) {
      console.log("Resetting form for new article");
      const blank: ArticleFormData = {
        title: '',
        content: '',
        image: '',
        tags: [],
        isDraft: false,
      };
      reset(blank);
      if (editor) {
        editor.commands.setContent('');
      }
    }
  }, [article, isNew, reset]);

  // Debug: Log current form values
  useEffect(() => {
    console.log("Current form values:", watchedValues);
  }, [watchedValues]);

  const onSubmit = async (data: ArticleFormData, returnToDashboard: boolean = true) => {
    if (!user) {
      toast({ title: "Error", description: "You must be logged in to edit an article." });
      return;
    }

    try {
      if (isNew) {
        const newArticle = await createArticle({
          title: data.title,
          content: data.content,
          image: data.image,
          tags: data.tags,
          isDraft: data.isDraft,
          authorId: user.id,
        });
        toast({ title: "Success", description: "Article created successfully." });
        if (returnToDashboard) {
          navigate({ to: '/dashboard/blog' });
        }
      } else {
        await updateArticle(blogSlug as string, {
          title: data.title,
          content: data.content,
          image: data.image,
          tags: data.tags,
          is_draft: data.isDraft,
          published_at: (() => {
            // If it's a draft, don't set a published date
            if (data.isDraft) {
              return null;
            }
            
            // If article has a valid published_at date (not 0, null, undefined), use it
            if (article?.article.published_at && article.article.published_at > 0) {
              return article.article.published_at;
            }
            
            // Otherwise, use current time for newly published articles
            return new Date().getTime();
          })(),
        });
        if (returnToDashboard) {
          navigate({ to: '/dashboard/blog' });
        } else {
          // If we are *not* navigating away, refresh local state:
          setValue('title', data.title);
          setValue('content', data.content);
          setValue('image', data.image || '');
          setValue('tags', data.tags);
          setValue('isDraft', data.isDraft);
        }
      }
    } catch (error) {
      console.error('Error saving article:', error);
      toast({ title: "Error", description: "Failed to save article. Please try again." });
    }
  };

  const rewriteArticle = async () => {
    if (!article?.article.id || !editor) return;
    setGeneratingRewrite(true);
    try {
      const oldText = editor.getText();
      const result = await updateArticleWithContext(article.article.id);
      if (result.success) {
        const newText = result.content;
        const diff = diffPartialText(oldText, newText, true);
        const diffHtml = mdParser.render(diff);
        setOriginalDocument(oldText);
        setPendingNewDocument(newText);
        setDiffing(true);
        editor.commands.setContent(diffHtml);
      }
    } catch (error) {
      console.error('Error rewriting article:', error);
      toast({ title: 'Error', description: 'Failed to rewrite article. Please try again.' });
    } finally {
      setGeneratingRewrite(false);
    }
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

    // Create messages for API (send original user message without document content)
    const apiMessages = [...chatMessages, { role: 'user', content: text } as ChatMessage];

    // Rest of the chat logic...
    await performChatRequest(apiMessages, assistantIndex, isEditRequest, text, currentContent);
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

  const performChatRequest = async (apiMessages: ChatMessage[], assistantIndex: number, isEditRequest: boolean, originalText: string, documentContent: string) => {
    setChatLoading(true);
    try {
      const apiUrl = `${VITE_API_BASE_URL}/agent/writing_copilot`;
      console.log('API Base URL:', VITE_API_BASE_URL);
      console.log('Full API URL:', apiUrl);
      console.log('Sending chat request:', { messages: apiMessages, documentContent });
      
      // Submit the request and get immediate response with request ID
      const resp = await fetch(apiUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          messages: apiMessages,
          documentContent: documentContent 
        }),
      });

      console.log('Response status:', resp.status);
      
      if (!resp.ok) {
        const errorText = await resp.text();
        console.error('Response error:', errorText);
        toast({ 
          title: "Connection Error", 
          description: `Failed to connect to writing assistant: ${resp.status} ${errorText}`,
          variant: "destructive"
        });
        throw new Error(`HTTP ${resp.status}: ${errorText}`);
      }

      const result = await resp.json();
      console.log('Got request response:', result);
      
      if (!result.requestId) {
        throw new Error('No request ID received');
      }

      // Connect to WebSocket and stream the response
      await streamChatResponse(result.requestId, assistantIndex, isEditRequest, originalText);

    } catch (err) {
      console.error('Chat error:', err);
      
      // Remove the optimistic message on error
      setChatMessages((prev) => prev.slice(0, -1));
      
      // Show user-friendly error
      if (err instanceof Error) {
        if (err.message.includes('Failed to fetch') || err.message.includes('NetworkError')) {
          toast({ 
            title: "Connection Error", 
            description: "Cannot connect to the writing assistant. Make sure the backend server is running on http://localhost:8080",
            variant: "destructive"
          });
        }
      }
    } finally {
      setChatLoading(false);
    }
  };

  const streamChatResponse = async (requestId: string, assistantIndex: number, isEditRequest: boolean, originalText: string) => {
    return new Promise<void>((resolve, reject) => {
      const wsUrl = `${VITE_API_BASE_URL.replace('http://', 'ws://').replace('https://', 'wss://')}/websocket`;
      console.log('Connecting to WebSocket:', wsUrl);
      
      const ws = new WebSocket(wsUrl);
      let acc = '';

      ws.onopen = () => {
        console.log('WebSocket connected, subscribing to request:', requestId);
        ws.send(JSON.stringify({
          action: 'subscribe',
          requestId: requestId
        }));
      };

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
          
          if (msg.role === 'assistant' && msg.content) {
            acc += msg.content;
            setChatMessages((prev) => {
              const updated = [...prev];
              updated[assistantIndex] = { role: 'assistant', content: acc } as ChatMessage;
              return updated;
            });
          }
          
          if (msg.done) {
            console.log('Stream completed');
            ws.close();
            
            // After response is complete, check if we should show a document edit option
            if (isEditRequest && acc.length > 100) {
              const codeBlockMatch = acc.match(/```(?:markdown|md)?\n([\s\S]*?)\n```/);
              if (codeBlockMatch) {
                const suggestedContent = codeBlockMatch[1].trim();
                if (suggestedContent.length > 50) {
                  setPendingEdit({
                    newContent: suggestedContent,
                    summary: `Suggested changes from: "${originalText}"`
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
        {/* Heading + Title input */}
        <div className="flex flex-col sm:flex-row items-center gap-4 mb-4">
          <h1 className="text-lg lg:text-2xl font-medium text-gray-900 dark:text-white whitespace-nowrap">
            {isNew ? 'New Article' : 'Edit Article'}
          </h1>
          <Input
            {...register('title')}
            value={watchedValues.title}
            onChange={(e) => setValue('title', e.target.value)}
            placeholder="Article Title"
            className="flex-1"
          />
          {errors.title && <p className="text-red-500 text-sm">{errors.title.message}</p>}
        </div>

        {/* Image card + publish controls + See Article */}
        <div className="flex flex-col sm:flex-row items-center gap-4 mb-6">
          {/* Small image card that opens the drawer */}
          <Drawer direction="right">
            <DrawerTrigger asChild>
              <Card className="w-24 h-16 flex items-center justify-center overflow-hidden cursor-pointer">
                <ImageLoader
                  article={article}
                  newImageGenerationRequestId={newImageGenerationRequestId}
                  stagedImageUrl={stagedImageUrl}
                  setStagedImageUrl={setStagedImageUrl}
                />
                {(!stagedImageUrl && !article?.article.image) && (
                  <span className="text-xs text-muted-foreground">Add Image</span>
                )}
              </Card>
            </DrawerTrigger>

            {/* Drawer content reused from previous implementation */}
            <DrawerContent className="w-full sm:max-w-sm ml-auto">
              <DrawerHeader>
                <DrawerTitle>Edit Header Image</DrawerTitle>
                <DrawerDescription>Update or generate a header image for your article.</DrawerDescription>
              </DrawerHeader>
              <div className="space-y-4 px-4">
                <div className="space-y-2">
                  <label className="block text-md font-medium leading-6 text-gray-900 dark:text-white">Image URL</label>
                  <Input
                    className="w-full"
                    {...register('image')}
                    value={watchedValues.image}
                    onChange={(e) => {
                      setValue('image', e.target.value);
                      setStagedImageUrl(e.target.value);
                    }}
                    placeholder="Optional, for header"
                  />
                  {errors.image && <p className="text-red-500">{errors.image.message}</p>}
                </div>
                <div className="flex items-center gap-2">
                  <Dialog open={generateImageOpen} onOpenChange={setGenerateImageOpen}>
                    <DialogTrigger asChild>
                      <Button variant="outline" className="">
                        <PencilIcon className="w-4 h-4 text-indigo-500" /> Edit Prompt
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
                        <div className="flex items-center gap-2 w-full">
                          <DialogClose asChild>
                            <Button variant="outline" className="w-full">Cancel</Button>
                          </DialogClose>
                          <Button
                            type="submit"
                            className="w-full"
                            onClick={async () => {
                              const result = await generateArticleImage(imagePrompt || '', article?.article.id || 0);
                              if (result.success) {
                                setNewImageGenerationRequestId(result.generationRequestId);
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
                      const result = await generateArticleImage(article?.article.title || '', article?.article.id || 0);
                      if (result.success) {
                        setNewImageGenerationRequestId(result.generationRequestId);
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
              <DrawerFooter>
                <DrawerClose asChild>
                  <Button variant="outline" className="w-full">Close</Button>
                </DrawerClose>
              </DrawerFooter>
            </DrawerContent>
          </Drawer>

          {/* Publish toggle, date picker, and See Article */}
          <div className="flex flex-col sm:flex-row items-center gap-4">
            <div className="flex items-center gap-2">
              <label htmlFor="isDraft" className="text-sm font-medium">Published</label>
              <Switch
                id="isDraft"
                checked={!watchedValues.isDraft}
                onCheckedChange={(checked) => {
                  setValue('isDraft', !checked);
                }}
              />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <label htmlFor="publishedAt" className="text-sm font-medium">Published Date</label>
                <Popover>
                  <PopoverTrigger asChild>
                    <Button
                      variant={"outline"}
                      className={cn(
                        'justify-start text-left font-normal',
                        !article?.article.published_at && 'text-muted-foreground'
                      )}
                    >
                      <CalendarIcon className="mr-2 h-4 w-4" />
                      {article?.article.published_at ? format(article.article.published_at, 'PPP') : <span>Pick a date</span>}
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-auto p-0">
                    <Calendar
                      mode="single"
                      selected={article?.article.published_at ? new Date(article.article.published_at) : undefined}
                      onSelect={(date: Date | undefined) => {
                        /* Not a form field; selection handled elsewhere if needed */
                      }}
                      initialFocus
                    />
                  </PopoverContent>
                </Popover>
              </div>
            </div>
            {!isNew && (
              <>
                <Link
                  to="/blog"
                  params={{ slug: article?.article.slug || '' }}
                  search={{ page: undefined, tag: undefined, search: undefined }}
                  target="_blank"
                  className="flex items-center gap-1 text-sm text-gray-900 dark:text-white"
                >
                  See Article <ExternalLinkIcon className="w-4 h-4" />
                </Link>
                <Button
                  type="button"
                  variant="outline"
                  className="text-sm flex flex-row gap-2"
                  onClick={rewriteArticle}
                  disabled={generatingRewrite}
                >
                  <RefreshCw className={cn('w-4 h-4 text-indigo-500', generatingRewrite && 'animate-spin')} /> Regenerate
                </Button>
              </>
            )}
          </div>
        </div>

        <Card>
          <form className="">
            <CardContent className="space-y-4">
              <div>
                <EditorContent
                  editor={editor}
                  className="w-full border-none rounded-md h-[calc(100vh-425px)] overflow-y-auto focus:outline-none"
                />
                {/* Hidden input to keep react-hook-form registration for content */}
                <input type="hidden" {...register('content')} value={watchedValues.content} />
                {errors.content && <p className="text-red-500">{errors.content.message}</p>}
                {diffing && (
                  <ConfirmChanges
                    onReject={() => {
                      if (editor) {
                        editor.commands.setContent(mdParser.render(originalDocument));
                      }
                      setValue('content', originalDocument);
                      setDiffing(false);
                      setPendingNewDocument('');
                    }}
                    onConfirm={() => {
                      if (editor) {
                        editor.commands.setContent(mdParser.render(pendingNewDocument));
                      }
                      setValue('content', pendingNewDocument);
                      setDiffing(false);
                      setPendingNewDocument('');
                    }}
                  />
                )}
                {pendingEdit && !diffing && (
                  <div className="absolute inset-0 flex items-center justify-center bg-white/80 dark:bg-gray-900/80 backdrop-blur-sm z-10">
                    <div className="bg-white dark:bg-gray-800 p-6 rounded shadow-lg border border-gray-200 space-y-4 max-w-md">
                      <h2 className="text-lg font-bold">Apply AI Suggestions?</h2>
                      <p className="text-sm text-gray-600 dark:text-gray-300">{pendingEdit.summary}</p>
                      <div className="flex justify-end space-x-4">
                        <Button variant="outline" onClick={() => setPendingEdit(null)}>
                          Ignore
                        </Button>
                        <Button onClick={() => {
                          const currentText = editor?.getText() || '';
                          const diff = diffPartialText(currentText, pendingEdit.newContent, true);
                          const diffHtml = mdParser.render(diff);
                          setOriginalDocument(currentText);
                          setPendingNewDocument(pendingEdit.newContent);
                          setDiffing(true);
                          if (editor) {
                            editor.commands.setContent(diffHtml);
                          }
                          setPendingEdit(null);
                        }}>
                          Preview Changes
                        </Button>
                      </div>
                    </div>
                  </div>
                )}
              </div>
              <label className="block text-sm font-medium leading-6 text-gray-900 dark:text-white">Tags</label>
              <div className='mb-2'>
                <ChipInput
                  value={watchedValues.tags}
                  onChange={(tags) => setValue('tags', tags.map((tag: string) => tag.toUpperCase()))}
                  placeholder="Type and press Enter to add tags..."
                />
                {errors.tags && <p className="text-red-500">{errors.tags.message}</p>}
              </div>
            </CardContent>
            <CardFooter className="flex justify-between">
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
                    setIsSaving(true);
                    handleSubmit((data) => onSubmit(data, false))();
                  }}
                  disabled={isSaving}>
                   {isSaving ? 'Saving...' : 'Save'}
                  </Button>
                }
              <Button type='submit' disabled={isLoading} onClick={() => {
                setIsLoading(true);
                handleSubmit((data) => onSubmit(data, true))();
              }}>
                {isLoading ? 'Updating...' : isNew ? 'Create Article' : 'Save & Return'}
              </Button>
              </div>
            </CardFooter>
          </form>
        </Card>
      </div>

      {/* Chat side-panel */}
      <div className="hidden xl:flex flex-col w-96 border rounded-md">
        <div className="flex-1 overflow-y-auto p-4 space-y-3">
          {chatMessages.map((m, i) => (
            <div key={i} className={`w-full flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div
                className={`max-w-xs whitespace-pre-wrap rounded-lg px-3 py-2 text-sm ${
                  m.role === 'user'
                    ? 'bg-indigo-500 text-white'
                    : 'bg-gray-200 dark:bg-gray-700 dark:text-white'
                }`}
              >
                {m.content || (m.role === 'assistant' && chatLoading ? (
                  <div className="flex items-center gap-1">
                    <div className="flex space-x-1">
                      <IconLoader2 className="w-4 h-4 text-indigo-500 animate-spin" />
                    </div>
                    <span className="text-xs opacity-75">thinking...</span>
                  </div>
                ) : m.content)}
              </div>
            </div>
          ))}
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
    </section>
  );
}
