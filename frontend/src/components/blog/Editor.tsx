import { useState, useEffect} from 'react';
import { useParams, useNavigate } from '@tanstack/react-router';
import { useAuth } from '@/services/auth/auth';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { format } from "date-fns"
import { Calendar as CalendarIcon, PencilIcon, SparklesIcon, RefreshCw } from "lucide-react"
import { ExternalLinkIcon, UploadIcon } from '@radix-ui/react-icons';
import { useQuery } from '@tanstack/react-query';
 
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
  const { blogSlug } = useParams({ from: '/dashboard/blog/edit/$blogSlug' });
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [generatingImage, setGeneratingImage] = useState(false);
  const [newImageGenerationRequestId, setNewImageGenerationRequestId] = useState<string | null>(null);
  const [stagedImageUrl, setStagedImageUrl] = useState<string | null | undefined>(undefined);
  const [generateImageOpen, setGenerateImageOpen] = useState(false);
  const [generatingRewrite, setGeneratingRewrite] = useState(false);

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
      reset({
        title: article.article.title || '',
        content: article.article.content || '',
        image: article.article.image || '',
        tags: tagNames,
        isDraft: article.article.is_draft,
      });
    } else if (isNew) {
      console.log("Resetting form for new article");
      reset({
        title: '',
        content: '',
        image: '',
        tags: [],
        isDraft: false,
      });
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
    if (!article?.article.id) return;
    
    setGeneratingRewrite(true);
    try {
      const result = await updateArticleWithContext(article.article.id);
      if (result.success) {
        setValue('content', result.content);
        console.log("Updated content via rewrite:", result.content);
      }
    } catch (error) {
      console.error('Error rewriting article:', error);
      toast({ title: "Error", description: "Failed to rewrite article. Please try again." });
    } finally {
      setGeneratingRewrite(false);
    }
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
    <section className="flex-1 p-0 md:p-4">
      <h1 className="text-lg lg:text-2xl font-medium text-gray-900 dark:text-white mb-6">
        Edit Article 
      </h1>
      <Card>
        <form className="mt-6">
          <CardContent className="space-y-4">
            <div>
              <div className='flex items-center justify-between gap-2 my-4'>
                <label className="block text-sm font-medium leading-6 text-gray-900 dark:text-white">Title</label>
                <Link to="/blog" params={{ slug: article?.article.slug || '' }} search={{ page: undefined, tag: undefined, search: undefined }} target="_blank" className="flex items-center gap-2 text-sm text-gray-900 dark:text-white">
                  See Article <ExternalLinkIcon className="w-4 h-4" />
                </Link>
              </div>
              <Input
                {...register('title')}
                value={watchedValues.title}
                placeholder="Article Title"
              />
              {errors.title && <p className="text-red-500">{errors.title.message}</p>}
            </div>

            <div className='flex items-center justify-center flex-col sm:flex-row '>
              <div className='flex items-center justify-center flex-col w-full sm:w-1/2 gap-2 mb-auto h-full min-h-[250px]'>
                <ImageLoader
                  article={article}
                  newImageGenerationRequestId={newImageGenerationRequestId}
                  stagedImageUrl={stagedImageUrl}
                  setStagedImageUrl={setStagedImageUrl}
                />
              </div>
              <div className='flex items-center justify-between flex-col w-full sm:w-1/2 gap-2 h-full min-h-[250px] '>
              <div className='flex flex-col items-start mr-auto w-full ml-2 gap-2'>
                <div className='flex flex-col items-start mr-auto w-full ml-2 gap-2'>
                  <label className="block text-md font-medium leading-6 text-gray-900 dark:text-white mr-auto mr-2">Image</label>
                  <div className='flex items-center justify-center w-full'>
                  <Input
                    className='w-full'
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
                </div>              
                <div className='flex items-center justify-between w-full ml-2'>
                  <div className='flex gap-2 w-full'>
                    <Button variant="outline" size="icon" disabled>
                      <UploadIcon className="w-4 h-4" />
                    </Button>
                    <div className='flex justify-end gap-2 w-full'>
                    <Dialog open={generateImageOpen} onOpenChange={setGenerateImageOpen}>
                      <DialogTrigger asChild>
                        <Button variant="outline" className=''>
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
                            className='h-[300px] w-full'
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
                                console.log("image prompt", imagePrompt);
                                const result = await generateArticleImage(imagePrompt || "", article?.article.id || 0);

                                if (result.success) {
                                  setNewImageGenerationRequestId(result.generationRequestId);
                                  toast({ title: "Success", description: "Image generated successfully." });
                                  setGenerateImageOpen(false);
                                } else {
                                  toast({ title: "Error", description: "Failed to generate image. Please try again." });
                                }
                              }}>Generate</Button>
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
                          console.log("image prompt", imagePrompt);
                          const result = await generateArticleImage(article?.article.title || "", article?.article.id || 0);

                          if (result.success) {
                            setNewImageGenerationRequestId(result.generationRequestId);
                            toast({ title: "Success", description: "Image generated successfully." });
                          } else {
                            toast({ title: "Error", description: "Failed to generate image. Please try again." });
                          }
                          setGeneratingImage(false);
                        }}>
                        <SparklesIcon className={cn("w-4 h-4 text-indigo-500", generatingImage && "animate-spin")} />
                      </Button>
                    </div>
                    
                  </div>
                </div>
              </div>
                <div style={{marginLeft: '2rem'}} className='flex w-full items-center flex-col gap-2 mt-auto'>
              <div className='mr-auto flex flex-row'>
              <label htmlFor="isDraft" className='text-sm font-medium flex flex-row mr-2'>Published </label>
              <Switch 
                id="isDraft"
                checked={!watchedValues.isDraft} 
                onCheckedChange={(checked) => {
                  setValue('isDraft', !checked);
                }} 
              />
              </div>
            <div className='flex w-full flex-col'>
                <div>
                  <label htmlFor="publishedAt" className='text-sm font-medium'>Published Date</label>
                </div>
                <Popover>
                  <PopoverTrigger asChild>
                    <Button
                  variant={"outline"}
                  className={cn(
                    "w-full justify-start text-left font-normal",
                    !article?.article.published_at && "text-muted-foreground"
                  )}
                >
                  <CalendarIcon className="mr-2 h-4 w-4" />
                  {article?.article.published_at ? format(article.article.published_at, "PPP") : <span>Pick a date</span>}
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-auto p-0">
                <Calendar
                  mode="single"
                  selected={article?.article.published_at ? new Date(article.article.published_at) : undefined}
                  onSelect={(date: Date | undefined) => {
                    // Date selection handled by the calendar component
                    // Published date is not part of the form schema
                  }}
                  initialFocus
                />
              </PopoverContent>
                </Popover>
            </div>
            </div>
              </div>
            </div>

            <div className='flex flex-row w-full justify-between'>

            <label className="block my-auto text-md font-medium leading-6 text-gray-900 dark:text-white ">Content</label>
            
              <Button
                type="button"
                variant="outline"
                className='text-sm font-medium text-gray-900 dark:text-white flex flex-row gap-2'
                onClick={async () => {
                  rewriteArticle()
                }}>
                <RefreshCw className={cn("w-4 h-4 text-indigo-500", generatingRewrite && "animate-spin")} /> Regenerate
              </Button>
            
            </div>
            <div>
              <Textarea
                defaultValue={article?.article.content || ''}
                className="w-full p-4 border border-gray-300 rounded-md h-[60vh]"
                {...register('content')}
                value={watchedValues.content}
                onChange={(e) => setValue('content', e.target.value)}
              />
              {errors.content && <p className="text-red-500">{errors.content.message}</p>}
            </div>
            <label className="block text-sm font-medium leading-6 text-gray-900 dark:text-white">Tags</label>
            <div>
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
    </section>
  );
}
