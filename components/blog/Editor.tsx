'use client'

import { useState, useEffect, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useUser } from '@/lib/auth';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { format } from "date-fns"
import { Calendar as CalendarIcon, PencilIcon } from "lucide-react"
 
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { useToast } from "@/hooks/use-toast";
import { cn } from '@/lib/utils';
import { updateArticle, getArticle, createArticle } from './actions';
import Link from 'next/link';
import { Article, ImageGeneration } from '@/db/schema';
import { Switch } from '@/components/ui/switch';
import { ExternalLinkIcon, UploadIcon } from '@radix-ui/react-icons';
import { SparklesIcon } from 'lucide-react';
import { Dialog, DialogTitle, DialogContent, DialogTrigger, DialogDescription, DialogFooter, DialogHeader, DialogClose } from '@/components/ui/dialog';
import { DEFAULT_IMAGE_PROMPT } from '@/lib/images/const';
import { generateArticleImage, getImageGeneration, getImageGenerationStatus } from '@/lib/images/generation';
import "@copilotkit/react-textarea/styles.css";
import { CopilotTextarea } from '@copilotkit/react-textarea';


const articleSchema = z.object({
  title: z.string().min(1, 'Title is required'),
  content: z.string().min(1, 'Content is required'),
  image: z.union([z.string().url(), z.literal('')]).optional(),
  tags: z.string(),
  isDraft: z.boolean(),
});

type ArticleFormData = z.infer<typeof articleSchema>;


export function ImageLoader({ article, newImageGenerationRequestId, stagedImageUrl, setStagedImageUrl }: {
  article: Article | null | undefined,
  newImageGenerationRequestId: string | null | undefined,
  stagedImageUrl: string | null | undefined,
  setStagedImageUrl: (url: string | null | undefined) => void
}) {
  const [imageUrl, setImageUrl] = useState<string | null>(null);


  useEffect(() => {
    const requestToFetch = newImageGenerationRequestId || article?.imageGenerationRequestId || null;
    console.log("imgen requestToFetch", requestToFetch);
    async function fetchImageGeneration() {
      console.log("imgen requestToFetch", requestToFetch);
      if (requestToFetch) {
        console.log("imgen requesting fetch", requestToFetch);
        const imgGen = await getImageGeneration(requestToFetch);
        console.log("imgen imgGen", imgGen);
        if (imgGen) {
          if (imgGen.outputUrl) {
            setImageUrl(imgGen.outputUrl);
          } else {
            // TODO check Fal subscription status
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
      console.log("imgen stagedImageUrl", stagedImageUrl);
      setImageUrl(stagedImageUrl);
    } else if (article && article.image) {
      console.log("imgen article", article);
      setImageUrl(article.image);
    }
  }, [article, stagedImageUrl, newImageGenerationRequestId]);

  if (!article) {
    return null;
  }

  if (imageUrl) {
    return (
      <div className='flex items-center justify-center'>
        <img className='rounded-md aspect-video object-cover' src={imageUrl} alt={article.title} width={'100%'} />
      </div>
    )
  }

  return null;

}

export default function ArticleEditor({ isNew }: { isNew?: boolean }) {
  const { toast } = useToast()
  const router = useRouter();
  const { user } = useUser();
  const { slug } = useParams();
  const [isLoading, setIsLoading] = useState(false);
  const [article, setArticle] = useState<Article | null>(null);
  const [newImageGenerationRequestId, setNewImageGenerationRequestId] = useState<string | null>(null);
  const [stagedImageUrl, setStagedImageUrl] = useState<string | null | undefined>(undefined);
  const [generateImageOpen, setGenerateImageOpen] = useState(false);
  

  const { register, handleSubmit, setValue, getValues, formState: { errors } } = useForm<ArticleFormData>({
    resolver: zodResolver(articleSchema),
  });

  const [imagePrompt, setImagePrompt] = useState<string | null>(DEFAULT_IMAGE_PROMPT[Math.floor(Math.random() * DEFAULT_IMAGE_PROMPT.length)]);

  // Consume from ImageLoader
  useEffect(() => {
    if (stagedImageUrl) {
      setValue('image', stagedImageUrl);
    }
  }, [stagedImageUrl, setValue]);

  useEffect(() => {
    async function fetchArticle() {
      if (isNew) {
        setValue('title', '');
        setValue('content', '');
        setValue('image', '');
        setValue('tags', '');
        setValue('isDraft', false);
        return;
      }

      if (slug) {
        const article = await getArticle(slug as string);
        if (article) {
          setArticle(article);
          console.log("is draft", article.isDraft);
          setValue('title', article.title);
          setValue('content', article.content);
          setValue('image', article.image || '');
          setValue('tags', article.tags ? article.tags.map(tag => tag.name).join(', ') : '');
          setValue('isDraft', article.isDraft);
        }
      }
    }
    fetchArticle();
  }, [slug, setValue, isNew]);

  const onSubmit = async (data: ArticleFormData) => {
    if (!user) {
      toast({ title: "Error", description: "You must be logged in to edit an article." });
      return;
    }

    setIsLoading(true);
    try {
      if (isNew) {
        const newArticle: Article = await createArticle({
          title: data.title,
          content: data.content,
          image: data.image,
          tags: data.tags.split(',').map(tag => tag.trim()),
          isDraft: data.isDraft,
          authorId: user.id,
        });
        toast({ title: "Success", description: "Article created successfully." });
        router.push(`/dashboard/blog`);
      } else {
        await updateArticle({
          slug: slug as string,
          title: data.title,
          content: data.content,
          image: data.image,
          tags: data.tags.split(',').map(tag => tag.trim()),
          isDraft: data.isDraft,
          publishedAt: article?.publishedAt || new Date().getTime(),
        });
        toast({ title: "Success", description: "Article updated successfully." });
        router.push(`/dashboard/blog`);
      }

    } catch (error) {
      console.error('Failed to update article:', error);
      toast({ title: "Error", description: "Failed to update article. Please try again." });
    } finally {
      setIsLoading(false);
    }
  };

  if (!user) {
    return <div>Please log in to edit articles.</div>;
  }

  return (
    <section className="flex-1 p-0 md:p-4">
      <h1 className="text-lg lg:text-2xl font-medium text-gray-900 dark:text-white mb-6">
        Edit Article
      </h1>
      <Card>
        <form onSubmit={handleSubmit(onSubmit)} className="mt-6">
          <CardContent className="space-y-4">
            <div>
              <div className='flex items-center justify-between gap-2 my-4'>
                <label className="block text-sm font-medium leading-6 text-gray-900 dark:text-white">Title</label>
                <Link href={`/blog/${slug}${article?.isDraft ? '?previewDraft=true' : ''}`} target="_blank" className="flex items-center gap-2 text-sm text-gray-900 dark:text-white">
                  See Article <ExternalLinkIcon className="w-4 h-4" />
                </Link>
              </div>
              <Input
                {...register('title')}
                placeholder="Article Title"
              />
              {errors.title && <p className="text-red-500">{errors.title.message}</p>}
            </div>
            <ImageLoader
              article={article}
              newImageGenerationRequestId={newImageGenerationRequestId}
              stagedImageUrl={stagedImageUrl}
              setStagedImageUrl={setStagedImageUrl}
            />
            <div className='flex items-center justify-between'>
              <label className="block text-md font-medium leading-6 text-gray-900 dark:text-white">Image</label>
              <div className='flex items-center gap-2'>
                <Button variant="outline" size="icon" disabled>
                  <UploadIcon className="w-4 h-4" />
                </Button>
                <Dialog open={generateImageOpen} onOpenChange={setGenerateImageOpen}>
                  <DialogTrigger asChild>
                    <Button variant="outline">
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
                        <Button type="submit" className="w-full"
                          onClick={async () => {
                            console.log("image prompt", imagePrompt);
                            const result = await generateArticleImage(imagePrompt || "", article?.id);

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
                <div>
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={async (e) => {
                      e.preventDefault();
                      console.log("image prompt", imagePrompt);
                      const result = await generateArticleImage(article?.title || "", article?.id, true);

                      if (result.success) {
                        setNewImageGenerationRequestId(result.generationRequestId);
                        toast({ title: "Success", description: "Image generated successfully." });
                      } else {
                        toast({ title: "Error", description: "Failed to generate image. Please try again." });
                      }
                    }}>
                    <SparklesIcon className="w-4 h-4 text-indigo-500" />
                  </Button>
                </div>
                
              </div>
            </div>
            <div>
              <Input
                {...register('image')}
                onChange={(e) => setStagedImageUrl(e.target.value)}
                placeholder="Optional, for header"
              />
              {errors.image && <p className="text-red-500">{errors.image.message}</p>}
            </div>
            <label className="block text-sm font-medium leading-6 text-gray-900 dark:text-white">Content</label>
            <div>
              <CopilotTextarea
                className="w-full p-4 border border-gray-300 rounded-md"
                value={article?.content || ''}
                onValueChange={(value) => {
                  setValue('content', value);
                }}
                autosuggestionsConfig={{
                  textareaPurpose: "the body of a blog post",
                  chatApiConfigs: {},
                }}
              />
              {errors.content && <p className="text-red-500">{errors.content.message}</p>}
            </div>
            <label className="block text-sm font-medium leading-6 text-gray-900 dark:text-white">Tags</label>
            <div>
              <Input
                {...register('tags')}
                placeholder="Tags (comma-separated)"
              />
              {errors.tags && <p className="text-red-500">{errors.tags.message}</p>}
            </div>
            <div className='flex items-center gap-2 py-2'>
              <div>
              <label htmlFor="isDraft">Published </label>
              <Switch {...register('isDraft')} checked={!article?.isDraft} onCheckedChange={(checked) => {
                if (article) {
                  setArticle({ ...article, isDraft: !checked });
                }
                setValue('isDraft', !checked);
              }} />
            </div>
            <div>
              <div>
                <label htmlFor="publishedAt">Published Date</label>
              </div>
                <Popover>
                  <PopoverTrigger asChild>
                    <Button
                  variant={"outline"}
                  className={cn(
                    "w-[280px] justify-start text-left font-normal",
                    !article?.publishedAt && "text-muted-foreground"
                  )}
                >
                  <CalendarIcon className="mr-2 h-4 w-4" />
                  {article?.publishedAt ? format(article.publishedAt, "PPP") : <span>Pick a date</span>}
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-auto p-0">
                <Calendar
                  mode="single"
                  selected={article?.publishedAt ? new Date(article.publishedAt) : undefined}
                  onSelect={(date) => {
                    if (article) {
                      setArticle({ ...article, publishedAt: date?.getTime() || 0 });
                    }
                  }}
                  initialFocus
                />
              </PopoverContent>
            </Popover>
            </div>
            </div>
          </CardContent>
          <CardFooter className="flex justify-between">
            <Button variant="secondary">
              <Link href="/dashboard/blog">
                Cancel
              </Link>
            </Button>
            <Button type='submit' disabled={isLoading}>
              {isLoading ? 'Updating...' : isNew ? 'Create Article' : 'Update Article'}
            </Button>
          </CardFooter>
        </form>
      </Card>
    </section>
  );
}
