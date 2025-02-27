'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { useUser } from '@/lib/auth';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { MoreHorizontal, Pencil, Plus, Sparkles, Trash2 } from 'lucide-react';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { useEffect, useState } from 'react';
import { getArticles, deleteArticle, ArticleRow } from './actions';
import { generateArticle } from '@/lib/llm/articles';
import { redirect, useRouter } from 'next/navigation';
import { Badge } from "@/components/ui/badge"
import { Article } from '@/db/schema';
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import Link from 'next/link';
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer"


export default function ArticlesPage() {
  const { user } = useUser();
  if (!user) {
    redirect('/login');
  }

  const router = useRouter();
  const [articles, setArticles] = useState<ArticleRow[] | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);
  const [aiArticleTitle, setAiArticleTitle] = useState<string>('');
  const [aiArticlePrompt, setAiArticlePrompt] = useState<string>('');

  useEffect(() => {
    const fetchArticles = async () => {
      const fetchedArticles = await getArticles();
      setArticles(fetchedArticles);
    };
    fetchArticles();
  }, []);

  const handleDelete = async (id: number) => {
    const result = await deleteArticle(id);
    if (result.success && articles) {
      setArticles(articles.filter(article => article.id !== id));
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
      router.push(`/dashboard/blog/edit/${newGeneratedArticle.slug}`);
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
                  <div className="flex items-start flex-wrap">{article.image && <img src={article.image} width={50} height={50} className="rounded-md mt-1" />}</div>
                  <div className="flex flex-col">
                    <Link href={`/dashboard/blog/edit/${article.slug}`} className="text-gray-900 text-md hover:underline">{article.title}</Link>
                    <p className="text-gray-500 text-xs">Published: {article.publishedAt ? new Date(article.publishedAt).toLocaleDateString() : 'Not published'}</p>
                    <div className="flex flex-wrap gap-2">{article.tags.map(tag => <Badge key={tag}
                  className="text-[0.6rem]" variant="outline"
                  >{tag}</Badge>)}</div>
                  <p className="text-gray-500 text-xs">{article.content ? article.content.slice(0, 150) + '...' : ''}</p>
                  </div>  
                </div>
              </TableCell>
              <TableCell className=""><p className="text-gray-500 text-xs">Created: {new Date(article.createdAt).toLocaleDateString()}</p></TableCell>
              <TableCell><Badge className={`text-[0.6rem] ${article.isDraft ? "bg-indigo-50" : "bg-orange-50"}`} variant="outline">{article.isDraft ? 'Draft' : 'Published'}</Badge></TableCell>
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
                      <Link href={`/dashboard/blog/edit/${article.slug}`}>
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
            <Link href="/dashboard/blog/new">
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
            {renderArticles(articles?.sort((a, b) => new Date(b.createdAt || 0).getTime() - new Date(a.createdAt || 0).getTime()) || [])}
          </TabsContent>
          <TabsContent value="published" className="p-0 w-full">
            {renderArticles(articles?.filter(article => article.isDraft === false) || [])}
          </TabsContent>
          <TabsContent value="drafts" className="p-0 w-full">
            {renderArticles(articles?.filter(article => article.isDraft === true) || [])}
          </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </section>
    </Drawer>
  );
}
