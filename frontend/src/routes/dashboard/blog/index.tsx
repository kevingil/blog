import { Card, CardContent } from '@/components/ui/card';
import { useAuth } from '@/services/auth/auth';
import { Button } from '@/components/ui/button';
import { Plus, Sparkles } from 'lucide-react';
import { getArticles } from '@/services/blog';
import { ArticleListItem } from '@/services/types';
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Link } from '@tanstack/react-router';
import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { GenerateArticleDrawer } from '@/components/blog/GenerateArticleDrawer';
import { ArticlesTable } from '@/components/blog/ArticlesTable';

export type GetArticlesResponse = {
  articles: ArticleListItem[];
  total_pages: number;
  include_drafts: boolean;
};

export const Route = createFileRoute('/dashboard/blog/')({
  component: ArticlesPage,
});

function ArticlesPage() {
  const { data: articlesPayload, isLoading, error, refetch } = useQuery<{ articles: ArticleListItem[], total_pages: number }>({
    queryKey: ['articles', 0, 20],
    queryFn: () => getArticles(0, null, true, 20) as Promise<{ articles: ArticleListItem[], total_pages: number }>
  });
  console.log("articlesPayload error", error);

  console.log("articlesPayload", articlesPayload);

  if (isLoading) return <div>Loading articles...</div>;
  if (error) return <div>Error loading articles</div>;
  return (
    <section className="flex-1 p-0 md:p-4">
        <div className="flex justify-between items-center">
          <h1 className="text-lg lg:text-2xl font-medium text-gray-900 dark:text-white mb-6">
            Articles
          </h1>
          <div className="flex justify-end items-center mb-6 gap-4">
            <GenerateArticleDrawer>
              <Button >
                <Sparkles className="mr-2 h-4 w-4" />
                Generate
              </Button>
            </GenerateArticleDrawer>
            <Link to="/dashboard/blog/new">
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                New Article
              </Button>
            </Link>
          </div>
        </div>

      <Card>
        <CardContent>
        <Tabs defaultValue="published">
          <TabsList className="mt-4">
            <TabsTrigger value="all">All</TabsTrigger>
            <TabsTrigger value="published">Published</TabsTrigger>
            <TabsTrigger value="drafts">Drafts</TabsTrigger>
          </TabsList>
          <TabsContent value="all" className="p-0 w-full">
            <ArticlesTable 
              articles={articlesPayload?.articles.sort((a: ArticleListItem, b: ArticleListItem) => new Date(b.article.created_at || 0).getTime() - new Date(a.article.created_at || 0).getTime()) || []}
              onArticleDeleted={() => refetch()}
            />
          </TabsContent>
          <TabsContent value="published" className="p-0 w-full">
            <ArticlesTable 
              articles={articlesPayload?.articles.filter((article: ArticleListItem) => article.article.is_draft === false) || []}
              onArticleDeleted={() => refetch()}
            />
          </TabsContent>
          <TabsContent value="drafts" className="p-0 w-full">
            <ArticlesTable 
              articles={articlesPayload?.articles.filter((article: ArticleListItem) => article.article.is_draft === true) || []}
              onArticleDeleted={() => refetch()}
            />
          </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </section>
  );
}
