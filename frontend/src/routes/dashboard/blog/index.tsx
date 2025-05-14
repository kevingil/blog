import { Card, CardContent } from '@/components/ui/card';
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { useAuth } from '@/services/auth/auth';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { MoreHorizontal, Pencil, Plus, Sparkles, Trash2 } from 'lucide-react';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { useState } from 'react';
import { getArticles, deleteArticle } from '@/services/blog';
import { ArticleRow } from '@/services/types';
import { generateArticle } from '@/services/llm/articles';
import { useNavigate } from '@tanstack/react-router';
import { Badge } from "@/components/ui/badge"
import { Article } from '@/services/types';
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Link } from '@tanstack/react-router';
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer"
import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';

type GetArticlesResponse = {
  articles: {
    id: number;
    title: string | null;
    slug: string | null;
    image: string;
    content: string;
    created_at: number;
    published_at: number;
    author: string;
    tags: string[];
    is_draft: boolean;
    image_generation_request_id: string;
  }[];
  totalPages: number;
};

export const Route = createFileRoute('/dashboard/blog/')({
  component: ArticlesPage,
});

function ArticlesPage() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const { data: articlesPayload, isLoading, error } = useQuery<GetArticlesResponse>({
    queryKey: ['articles', 1],
    queryFn: () => getArticles(1) as Promise<GetArticlesResponse>
  });
  const articles: ArticleRow[] = articlesPayload?.articles.map((article: any) => ({
    ...article,
    title: article.title ?? '',
    slug: article.slug ?? '',
    createdAt: article.created_at,
    publishedAt: article.published_at,
    isDraft: article.is_draft
  })) || [];

  const [isGenerating, setIsGenerating] = useState(false);
  const [aiArticleTitle, setAiArticleTitle] = useState<string>('');
  const [aiArticlePrompt, setAiArticlePrompt] = useState<string>('');

  const handleDelete = async (id: number) => {
    const result = await deleteArticle(id);
    if (result.success && articles) {
      // Filter out the deleted article from the articles state
      // This is a simple implementation. Depending on your use case, you might want to use a more robust state management solution
      // For example, you could use a state management library like Redux or a context to manage the state of your articles
      // or you could implement a more complex state management strategy based on your application's requirements
      // This is a placeholder and should be replaced with a more appropriate implementation
    } else {
      console.error('Failed to delete article');
    }
  };

  const handleGenerate = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsGenerating(true);
  
    try {
      if (!user?.id) {
        throw new Error("User not found");
      }
      const newGeneratedArticle: Article = await generateArticle(aiArticlePrompt, aiArticleTitle, user.id);
      navigate({ to: `/dashboard/blog/edit/${newGeneratedArticle.slug}` });
    } catch (err) {
      console.error("Generation failed:", err);
    } finally {
      setIsGenerating(false);
    }
  };

  function renderArticles(articles: ArticleRow[]) {
    return (
      <Table className="">
        <TableHeader>
          <TableRow>
            <TableHead>Title</TableHead>
            <TableHead>Tags</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {articles ? articles.map((article) => (
            <TableRow key={article.id}>
              <TableCell className="w-full flex flex-col gap-1">
                <div className="flex items-start gap-2">
                  <div className="flex items-start flex-wrap">{article.image && <img src={article.image} className="rounded-md mt-1 w-10 h-10 min-w-10 min-h-10 object-cover" />}</div>
                  <div className="flex flex-col">
                    <Link to={`/dashboard/blog/edit/$blogSlug`} params={{ blogSlug: article.slug || '' }} className="text-gray-900 text-md hover:underline dark:text-white">{article.title}</Link>
                    <p className="text-gray-500 text-xs">Published: {article.publishedAt ? new Date(article.publishedAt).toLocaleDateString() : 'Not published'}</p>
                    <div className="flex flex-wrap gap-2">{article.tags.map(tag => <Badge key={tag}
                  className="text-[0.6rem]" variant="outline"
                  >{tag}</Badge>)}</div>
                  <p className="text-gray-500 text-xs">{article.content ? article.content.slice(0, 150) + '...' : ''}</p>
                  </div>  
                </div>
              </TableCell>
              <TableCell className=""><p className="text-gray-500 text-xs">Created: {new Date(article.createdAt).toLocaleDateString()}</p></TableCell>
              <TableCell><Badge className={`text-[0.6rem] ${article.isDraft ? "bg-indigo-50 dark:bg-indigo-900" : "bg-orange-50 dark:bg-orange-900"}`} variant="outline">{article.isDraft ? 'Draft' : 'Published'}</Badge></TableCell>
              <TableCell className="text-right">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" className="h-8 w-8 p-0">
                      <span className="sr-only">Open menu</span>
                      <MoreHorizontal className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem asChild>
                      <Link to={`/dashboard/blog/edit/$blogSlug`} params={{ blogSlug: article.slug || '' }}>
                        <Pencil className="mr-2 h-4 w-4" />
                        Edit
                      </Link>
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => handleDelete(article.id)}>
                      <Trash2 className="mr-2 h-4 w-4" />
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableCell>
            </TableRow>
          )) : null}
        </TableBody>
      </Table>
    )
  }

  if (isLoading) return <div>Loading articles...</div>;
  if (error) return <div>Error loading articles</div>;
  return (
    <Drawer>
    <section className="flex-1 p-0 md:p-4">
        <div className="flex justify-between items-center">
          <h1 className="text-lg lg:text-2xl font-medium text-gray-900 dark:text-white mb-6">
            Articles
          </h1>
          <div className="flex justify-end items-center mb-6 gap-4">
            <DrawerTrigger asChild>
              <Button >
                <Sparkles className="mr-2 h-4 w-4" />
                Generate
              </Button>
            </DrawerTrigger>
            <DrawerContent className="w-full max-w-3xl mx-auto">
              <DrawerHeader>
                <DrawerTitle>Generate Article</DrawerTitle>
              </DrawerHeader>

              {/* Example of a simple form approach */}
              <form onSubmit={handleGenerate} className="space-y-4 px-4 pb-4">
                <div>
                  <p className="pb-4 text-gray-500 text-[0.9rem]">Both the title and prompt will be used to generate the article</p>
                  <label htmlFor="title" className="block font-bold text-gray-500 text-sm mb-2">
                    Title
                  </label>
                  <Input
                    id="title"
                    type="text"
                    placeholder="Title will be used by the AI to generate the article"
                    value={aiArticleTitle}
                    onChange={(e) => setAiArticleTitle(e.target.value)}
                    required
                  />
                </div>

                <div>
                  <label htmlFor="prompt" className="block font-bold text-gray-500 text-sm mb-2">
                    Prompt
                  </label>
                  <Textarea
                    id="prompt"
                    className="h-48"
                    placeholder="Addidional instructions"
                    value={aiArticlePrompt}
                    onChange={(e) => setAiArticlePrompt(e.target.value)}
                    required
                  />
                </div>

                <DrawerFooter>
                  <div className="w-full flex justify-end items-center gap-4">
                  <DrawerClose asChild>
                    <Button className="w-full" variant="outline" type="button">
                      Cancel
                    </Button>
                  </DrawerClose>
                  <Button className="w-full" type="submit" disabled={isGenerating}>
                    {isGenerating ? "Generating..." : "Generate"}
                  </Button>
                  </div>
                </DrawerFooter>
              </form>
            </DrawerContent>
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
            {renderArticles(articles.sort((a: ArticleRow, b: ArticleRow) => new Date(b.createdAt || 0).getTime() - new Date(a.createdAt || 0).getTime()))}
          </TabsContent>
          <TabsContent value="published" className="p-0 w-full">
            {renderArticles(articles.filter((article: ArticleRow) => article.isDraft === false))}
          </TabsContent>
          <TabsContent value="drafts" className="p-0 w-full">
            {renderArticles(articles.filter((article: ArticleRow) => article.isDraft === true))}
          </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </section>
    </Drawer>
  );
}
