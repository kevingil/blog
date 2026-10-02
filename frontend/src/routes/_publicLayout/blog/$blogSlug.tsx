import { Suspense, useEffect, useRef, useState } from 'react';
import { redirect, useParams } from '@tanstack/react-router';
import { format } from 'date-fns';
import { marked, Token } from 'marked';
import { Card, CardContent, CardFooter } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import { getArticleData, getRecommendedArticles } from '@/services/blog';
import hljs from 'highlight.js';
import { createFileRoute } from '@tanstack/react-router';
import { ArticleData, RecommendedArticle, isPublished, getDisplayTitle, getDisplayContent, getDisplayImage, externalArticleUrl } from '@/services/types';
import { BlurhashImage } from '@/components/media/BlurhashImage';
import { Link } from "@tanstack/react-router"
import { cn } from "@/lib/utils"

export const Route = createFileRoute('/_publicLayout/blog/$blogSlug')({
  component: Page,
});

function ArticleSkeleton() {
  return (
    <div className="max-w-6xl mx-auto">
      <Skeleton className="h-12 w-3/4 mb-4" />
      <div className="flex items-center mb-6">
        <Skeleton className="h-10 w-10 rounded-full mr-4" />
        <div>
          <Skeleton className="h-4 w-32 mb-2" />
          <Skeleton className="h-3 w-24" />
        </div>
      </div>
      <Skeleton className="h-64 w-full mb-6" />
      <Skeleton className="h-4 w-full mb-2" />
      <Skeleton className="h-4 w-5/6 mb-2" />
      <Skeleton className="h-4 w-4/6 mb-6" />
      <div className="flex flex-wrap gap-2 mb-8">
        <Skeleton className="h-6 w-16" />
        <Skeleton className="h-6 w-20" />
        <Skeleton className="h-6 w-24" />
      </div>
    </div>
  );
}

function RecommendedArticlesSkeleton() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
      {[1, 2, 3].map((i) => (
        <Card key={i}>
          <Skeleton className="h-48" />
          <CardContent className="p-4">
            <Skeleton className="h-6 w-3/4 mb-2" />
            <Skeleton className="h-4 w-1/2 mb-4" />
          </CardContent>
          <CardFooter className="p-4 pt-0">
            <Skeleton className="h-10 w-full" />
          </CardFooter>
        </Card>
      ))}
    </div>
  );
}

export default function Page() {
  const { blogSlug } = useParams({ from: '/_publicLayout/blog/$blogSlug' });
  const [articleData, setArticleData] = useState<ArticleData | null>(null);
  const searchParams = new URLSearchParams(window.location.search);
  const previewDraft = searchParams.get('previewDraft');
  const [isRecommendedAnimated, setIsRecommendedAnimated] = useState(false);

  const articleRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    window.scrollTo(0, 0);
  }, []);

  useEffect(() => {
    console.log("previewDraft", previewDraft);
    const loadData = async () => {
      const data = await getArticleData(blogSlug);
      setArticleData(data);
      if (!data) {
        redirect({ to: '/not-found' });
      }
      // If article is not published and not in preview mode, redirect to not-found
      if (!isPublished(data?.article) && (previewDraft !== 'true')) {
        console.log("Article not published and previewDraft not true, notFound");
        redirect({ to: '/not-found' });
      }
    };
    loadData();
  }, [blogSlug]);

  const externalUrl = externalArticleUrl(articleData?.article);

  useEffect(() => {
    if (externalUrl) {
      window.location.replace(externalUrl);
    }
  }, [externalUrl]);

  useEffect(() => {
    // Trigger recommended articles animation after main article loads
    const timer = setTimeout(() => {
      setIsRecommendedAnimated(true);
    }, 500); // Delay after main article animation
    return () => clearTimeout(timer);
  }, [articleData]);

  if (externalUrl) {
    return (
      <div className="container mx-auto py-16">
        <meta httpEquiv="refresh" content={`0;url=${externalUrl}`} />
        <article className="max-w-xl mx-auto bg-card/60 text-card-foreground border border-border/60 p-8 shadow-lg">
          <p className="text-sm text-muted-foreground mb-2">Opening the original article</p>
          <h1 className="text-2xl font-semibold mb-4">
            {getDisplayTitle(articleData!.article)}
          </h1>
          <a href={externalUrl} className="text-primary underline underline-offset-4">
            Continue to the article
          </a>
        </article>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-8" ref={articleRef}>
      <Suspense fallback={<ArticleSkeleton />}>
        <ArticleContent slug={blogSlug} articleData={articleData} />
      </Suspense>

      <Separator className="my-12" />

      <section className={cn(
        "max-w-4xl mx-auto bg-card/60 text-card-foreground border border-border/60 p-8 shadow-lg backdrop-blur-md",
        isRecommendedAnimated ? "card-animated" : "card-hidden"
      )}>
        <h2 className="text-2xl font-bold mb-6">Other Articles</h2>
        <Suspense fallback={<RecommendedArticlesSkeleton />}>
          <RecommendedArticles slug={blogSlug} articleData={articleData} />
        </Suspense>
      </section>
    </div>
  );
}

function ArticleContent({ slug, articleData }: { slug: string, articleData: ArticleData | null }) {
  const article = articleData?.article;
  const [isAnimated, setIsAnimated] = useState(false);
  const searchParams = new URLSearchParams(window.location.search);
  const isPreview = searchParams.get('previewDraft') === 'true';

  useEffect(() => {
    // Trigger animation with a delay
    const timer = setTimeout(() => {
      setIsAnimated(true);
    }, 100);
    return () => clearTimeout(timer);
  }, []);

  marked.use({
    renderer: {
      code(this: any, token: Token & {lang?: string, text: string}) {
        const lang = token.lang && hljs.getLanguage(token.lang) ? token.lang : 'plaintext';
        const highlighted = hljs.highlight(token.text, { language: lang }).value;
        console.log("highlighted", highlighted);
        return `<pre><code class="hljs language-${lang}">${highlighted}</code></pre>`;
      }
    }
  });

  // Use draft content for preview mode, otherwise prefer published content
  const displayTitle = article ? (isPreview ? article.draft_title : getDisplayTitle(article)) : '';
  const displayContent = article ? (isPreview ? article.draft_content : getDisplayContent(article)) : '';
  const displayImage = article
    ? (isPreview ? getDisplayImage(article, false) : getDisplayImage(article))
    : null;

  return (
    <article className={cn(
      "max-w-4xl mx-auto bg-card/60 text-card-foreground border border-border/60 p-8 shadow-lg backdrop-blur-md",
      isAnimated ? "card-animated" : "card-hidden"
    )}>
      {isPreview && (
        <div className="mb-4 p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
          <p className="text-sm text-amber-800 dark:text-amber-200 font-medium">
            Draft Preview - This content is not yet published
          </p>
        </div>
      )}
      <h1 className="text-4xl font-bold mb-4">{displayTitle}</h1>
      {displayImage?.url && (
        <div className="mb-6">
          <BlurhashImage
            src={displayImage.url}
            alt={displayTitle}
            blurhash={displayImage.blurhash}
            className="aspect-video"
            imgClassName="h-full w-full object-cover"
          />
          {displayImage.blurhash ? (
            <p className="mt-2 font-mono text-xs text-muted-foreground break-all" data-testid="article-blurhash">
              blurhash {displayImage.blurhash}
            </p>
          ) : null}
        </div>
      )}
      <div className="flex items-center mb-6">
        <div>
          <p className="font-semibold">{articleData?.author?.name}</p>
          <p className="text-sm text-muted-foreground">
            {(() => {
              const date = article?.published_at ? new Date(article.published_at) : null;
              if (!date || isNaN(date.getTime())) return isPreview ? 'Draft' : 'Unknown';
              const year = date.getFullYear();
              if (year > 2100) return 'Unknown';
              return format(date, 'MMMM d, yyyy');
            })()}
          </p>
        </div>
      </div>
      <div className="blog-post prose max-w-none mb-8" dangerouslySetInnerHTML={{ __html: marked(displayContent || '') }}
      />
      <div className="flex flex-wrap gap-2 mb-8">
        {articleData?.tags?.map((tag) => (
          <Badge key={tag.tag_id} variant="secondary" className='text-primary border-solid border-1 border-indigo-500'>{tag.tag_name?.toUpperCase()}</Badge>
        ))}
      </div>
    </article>
  );
}

function RecommendedArticles({ slug, articleData }: { slug: string, articleData: ArticleData | null }) {
  const [recommendedArticles, setRecommendedArticles] = useState<RecommendedArticle[] | null>(null);
  const content = articleData?.article;
  
  console.log("RecommendedArticles", recommendedArticles);
  console.log("content", content);

  useEffect(() => {
    const loadData = async () => {
      if (!content) {
        return;
      }
      const data = await getRecommendedArticles(content?.id);
      setRecommendedArticles(data);
    };
    loadData();
  }, [slug, content]);
  
  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
      {recommendedArticles?.map((article: RecommendedArticle, index: number) => {
        const externalUrl = externalArticleUrl(article);
        const card = (
        <Card className="rounded-none p-0" animationDelay={index * 100}>
          {(article.image?.url || article.image_url) && (
            <BlurhashImage
              src={article.image?.url || article.image_url || ''}
              alt={article.title}
              blurhash={article.image?.blurhash}
              className="h-48 w-full"
              imgClassName="h-full w-full object-cover"
            />
          )}
          <CardContent className="p-4">
            <h3 className="text-lg font-semibold mb-2">{article.title}</h3>
            <p className="text-sm text-muted-foreground mb-4">
              {(() => {
                const date = article.published_at ? new Date(article.published_at) : null;
                if (!date || isNaN(date.getTime())) return 'Unknown';
                const year = date.getFullYear();
                if (year > 2100) return 'Unknown';
                return format(date, 'MMMM d, yyyy');
              })()}
            </p>
          </CardContent>
        </Card>
        );
        if (externalUrl) {
          return (
            <a key={article.slug} href={externalUrl} target="_blank" rel="noopener noreferrer">
              {card}
            </a>
          );
        }
        return (
        <Link key={article.slug} to="/blog/$blogSlug" params={{ blogSlug: article.slug }} >
          {card}
        </Link>
        );
      })}
    </div>
  );
}
