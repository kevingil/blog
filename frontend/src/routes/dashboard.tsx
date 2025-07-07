import { redirect, useLocation } from '@tanstack/react-router';
import { createFileRoute } from '@tanstack/react-router';
import { Outlet } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { AppSidebar } from "@/components/app-sidebar"
import { ArticlesTable } from "@/components/blog/ArticlesTable"
import { SiteHeader } from "@/components/site-header"
import { UserProfileBanner } from "@/components/user-profile-banner"
import {
  SidebarInset,
  SidebarProvider,
} from "@/components/ui/sidebar"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Card, CardContent } from '@/components/ui/card'
import { getArticles } from '@/services/blog';
import { ArticleListItem } from '@/services/types';

export const Route = createFileRoute('/dashboard')({
  component: DashboardLayout,
  beforeLoad: ({ context, location }) => {
    if (!context.auth || !context.auth.isAuthenticated) {
      console.log("user dashboard beforeLoad", JSON.stringify(context.auth));
      throw redirect({
        to: '/login',
        search: {
          redirect: location.href,
        },
      });
    }
  },
});

function DashboardLayout() {
  const location = useLocation();
  const isRootDashboard = location.pathname === '/dashboard';

  // Separate queries for each tab
  const { data: allArticles, isLoading: allLoading, error: allError, refetch: refetchAll } = useQuery<{ articles: ArticleListItem[], total_pages: number }>({
    queryKey: ['dashboard-articles', 'all', 0, 20],
    queryFn: () => getArticles(0, null, 'all', 20) as Promise<{ articles: ArticleListItem[], total_pages: number }>,
    enabled: isRootDashboard
  });

  const { data: publishedArticles, isLoading: publishedLoading, error: publishedError, refetch: refetchPublished } = useQuery<{ articles: ArticleListItem[], total_pages: number }>({
    queryKey: ['dashboard-articles', 'published', 0, 20],
    queryFn: () => getArticles(0, null, 'published', 20) as Promise<{ articles: ArticleListItem[], total_pages: number }>,
    enabled: isRootDashboard
  });

  const { data: draftArticles, isLoading: draftLoading, error: draftError, refetch: refetchDrafts } = useQuery<{ articles: ArticleListItem[], total_pages: number }>({
    queryKey: ['dashboard-articles', 'drafts', 0, 20],
    queryFn: () => getArticles(0, null, 'drafts', 20) as Promise<{ articles: ArticleListItem[], total_pages: number }>,
    enabled: isRootDashboard
  });

  const isLoading = allLoading || publishedLoading || draftLoading;
  const error = allError || publishedError || draftError;
  const refetch = () => {
    refetchAll();
    refetchPublished();
    refetchDrafts();
  };

  return (
    <SidebarProvider
      style={
        {
          "--sidebar-width": "calc(var(--spacing) * 72)",
          "--header-height": "calc(var(--spacing) * 12)",
        } as React.CSSProperties
      }
    >
      <AppSidebar variant="inset" />
      <SidebarInset>
        <SiteHeader />
        <div className="flex flex-1 flex-col">
          {isRootDashboard ? (
            <div className="@container/main flex flex-1 flex-col gap-2">
              <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
                <div className="px-4 lg:px-6">
                  <UserProfileBanner />
                </div>
                <div className="px-4 lg:px-6">
                  <Card>
                    <CardContent>
                      {isLoading ? (
                        <div className="p-4">Loading articles...</div>
                      ) : error ? (
                        <div className="p-4">Error loading articles</div>
                      ) : (
                        <Tabs defaultValue="published">
                          <TabsList className="mt-4">
                            <TabsTrigger value="all">All</TabsTrigger>
                            <TabsTrigger value="published">Published</TabsTrigger>
                            <TabsTrigger value="drafts">Drafts</TabsTrigger>
                          </TabsList>
                          <TabsContent value="all" className="p-0 w-full">
                            <ArticlesTable 
                              articles={allArticles?.articles.sort((a: ArticleListItem, b: ArticleListItem) => new Date(b.article.created_at || 0).getTime() - new Date(a.article.created_at || 0).getTime()) || []}
                              onArticleDeleted={() => refetch()}
                            />
                          </TabsContent>
                          <TabsContent value="published" className="p-0 w-full">
                            <ArticlesTable 
                              articles={publishedArticles?.articles.sort((a: ArticleListItem, b: ArticleListItem) => new Date(b.article.created_at || 0).getTime() - new Date(a.article.created_at || 0).getTime()) || []}
                              onArticleDeleted={() => refetch()}
                            />
                          </TabsContent>
                          <TabsContent value="drafts" className="p-0 w-full">
                            <ArticlesTable 
                              articles={draftArticles?.articles.sort((a: ArticleListItem, b: ArticleListItem) => new Date(b.article.created_at || 0).getTime() - new Date(a.article.created_at || 0).getTime()) || []}
                              onArticleDeleted={() => refetch()}
                            />
                          </TabsContent>
                        </Tabs>
                      )}
                    </CardContent>
                  </Card>
                </div>
              </div>
            </div>
          ) : (
            <main className="flex-1 overflow-y-auto p-0">
              <Outlet />
            </main>
          )}
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}
