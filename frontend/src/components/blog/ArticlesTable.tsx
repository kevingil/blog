import { Button } from '@/components/ui/button';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { MoreHorizontal, Pencil, Trash2 } from 'lucide-react';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { deleteArticle } from '@/services/blog';
import { ArticleListItem } from '@/services/types';
import { Badge } from "@/components/ui/badge";
import { Link } from '@tanstack/react-router';

interface ArticlesTableProps {
  articles: ArticleListItem[];
  onArticleDeleted?: () => void;
}

export function ArticlesTable({ articles, onArticleDeleted }: ArticlesTableProps) {
  const handleDelete = async (id: number) => {
    const result = await deleteArticle(id);
    if (result.success) {
      onArticleDeleted?.(); // Callback to refresh data
    } else {
      console.error('Failed to delete article');
    }
  };

  return (
    <div className="overflow-hidden rounded-lg border">
      <Table>
        <TableHeader className="bg-muted sticky top-0 z-10">
          <TableRow>
            <TableHead>Title</TableHead>
            <TableHead>Tags</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {articles.map((article) => (
            <TableRow key={article.article.id}>
              <TableCell className="w-full flex flex-col gap-1">
                <div className="flex items-start gap-2">
                  <div className="flex items-start flex-wrap">
                    {article.article.image && (
                      <img 
                        src={article.article.image} 
                        className="rounded-md mt-1 w-10 h-10 min-w-10 min-h-10 object-cover" 
                        alt={article.article.title}
                      />
                    )}
                  </div>
                  <div className="flex flex-col">
                    <Link 
                      to={`/dashboard/blog/edit/$blogSlug`} 
                      params={{ blogSlug: article.article.slug || '' }} 
                      className="text-gray-900 text-md hover:underline dark:text-white"
                    >
                      {article.article.title}
                    </Link>
                    <p className="text-gray-500 text-xs">
                      Published: {article.article.published_at 
                        ? new Date(article.article.published_at).toLocaleDateString() 
                        : 'Not published'}
                    </p>
                    <div className="flex flex-wrap gap-2">
                      {article.tags?.map((tag: { tag_name: string }) => (
                        <Badge 
                          key={tag.tag_name}
                          className="text-[0.6rem]" 
                          variant="outline"
                        >
                          {tag.tag_name.toUpperCase()}
                        </Badge>
                      ))}
                    </div>
                    <p className="text-gray-500 text-xs">
                      {article.article.content 
                        ? article.article.content.slice(0, 150) + '...' 
                        : ''}
                    </p>
                  </div>  
                </div>
              </TableCell>
              <TableCell className="">
                <p className="text-gray-500 text-xs">
                  Created: {new Date(article.article.created_at).toLocaleDateString()}
                </p>
              </TableCell>
              <TableCell>
                <Badge 
                  className={`text-[0.6rem] ${
                    article.article.is_draft 
                      ? "bg-indigo-50 dark:bg-indigo-900" 
                      : "bg-orange-50 dark:bg-orange-900"
                  }`} 
                  variant="outline"
                >
                  {article.article.is_draft ? 'Draft' : 'Published'}
                </Badge>
              </TableCell>
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
                      <Link to={`/dashboard/blog/edit/$blogSlug`} params={{ blogSlug: article.article.slug || '' }}>
                        <Pencil className="mr-2 h-4 w-4" />
                        Edit
                      </Link>
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => handleDelete(article.article.id)}>
                      <Trash2 className="mr-2 h-4 w-4" />
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
} 
