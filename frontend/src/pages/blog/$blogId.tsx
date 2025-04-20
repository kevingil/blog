'use client'

import { Suspense, useEffect, useRef, useState } from 'react';
import { redirect, useParams } from 'next/navigation';
import { format } from 'date-fns';
import { marked, Token } from 'marked';
import { Card, CardContent, CardFooter } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import { ArticleData, getArticleData, getRecommendedArticles, RecommendedArticle } from '@/app/dashboard/blog/actions';
import { useSearchParams } from 'next/navigation';
import hljs from 'highlight.js';
import { Analytics } from '@vercel/analytics/react';
import type { BeforeSendEvent } from '@vercel/analytics/react';

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
          <Skeleton className="h-48 rounded-t-lg" />
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
  const { slug } = useParams();
  const [articleData, setArticleData] = useState<ArticleData | null>(null);
  const content = articleData?.article;
  const searchParams = useSearchParams();
  const previewDraft = searchParams.get('previewDraft');


  // State to control the animation
  const articleRef = useRef<HTMLDivElement | null>(null);
  const [animate, setAnimate] = useState(false);

  // Intersection Observer
  useEffect(() => {
    setAnimate(true);
    window.scrollTo(0, 0);
  }, []);

  useEffect(() => {
    console.log("previewDraft", previewDraft);
    const loadData = async () => {
      const data = await getArticleData(slug as string);
      setArticleData(data);
      if (!data) {
        redirect('/404');
      }
      if (data.article.isDraft && (previewDraft !== 'true')) {
        console.log("previewDraft not true, notFound");
        redirect('/404');
      }
    };
    loadData();
  }, [slug]);

  return (
    <div className={`container mx-auto py-8 delay-1000 ${animate ? 'animate' : 'hide-down'}`} ref={articleRef}>
      <Suspense fallback={<ArticleSkeleton />}>
        <ArticleContent slug={slug as string} articleData={articleData} />
      </Suspense>

      <Separator className="my-12" />

      <section className="max-w-4xl mx-auto">
        <h2 className="text-2xl font-bold mb-6">Other Articles</h2>
        <Suspense fallback={<RecommendedArticlesSkeleton />}>
          <RecommendedArticles slug={slug as string} articleData={articleData} />
        </Suspense>
      </section>
      <Analytics
        beforeSend={(event: BeforeSendEvent) => {
          if (event.url.includes('previewDraft')) {
            return null;
          }
          return event;
        }}
      />
    </div>
  );
}

 function ArticleContent({ slug, articleData }: { slug: string, articleData: ArticleData | null }) {
  const content = articleData?.article;


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

  return (
    <article className="max-w-4xl mx-auto">
      <h1 className="text-4xl font-bold mb-4">{content?.title}</h1>
      {content?.image && (
        <img
          src={content.image}
          alt={content.title}
          className="rounded-2xl mb-6 object-cover aspect-video"
        />
      )}
      <div className="flex items-center mb-6">
        <div>
          <p className="font-semibold">{articleData?.author_name}</p>
          <p className="text-sm text-muted-foreground">
            { content?.publishedAt ? format(new Date(content?.publishedAt), 'MMMM d, yyyy') : 'Unknown'}
          </p>
        </div>
      </div>
      <div className="blog-post prose max-w-none mb-8" dangerouslySetInnerHTML={{ __html: marked(content?.content || '') }}
      />
      <div className="flex flex-wrap gap-2 mb-8">
        {articleData?.tags?.map((tag) => (
          <Badge key={tag.tagId} variant="secondary" className='text-primary'>{tag.tagName}</Badge>
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
      {recommendedArticles?.map((article: RecommendedArticle) => (
        <a href={`/blog/${article.slug}`} key={article.id} >
        <Card className="p-0">
          {article.image && (
            <img
              src={article.image}
              alt={article.title}
              className="rounded-t-lg object-cover h-48 w-full"
            />
          )}
          <CardContent className="p-4">
            <h3 className="text-lg font-semibold mb-2">{article.title}</h3>
            <p className="text-sm text-muted-foreground mb-4">
              {article.publishedAt ? format(new Date(article.publishedAt), 'MMMM d, yyyy') : 'Unknown'}
            </p>
          </CardContent>
        </Card>
        </a>
      ))}
    </div>
  );
}
