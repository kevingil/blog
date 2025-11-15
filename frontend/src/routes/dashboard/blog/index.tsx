import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Plus, Sparkles } from 'lucide-react';
import { getArticles, searchArticles } from '@/services/blog';
import { ArticleListItem } from '@/services/types';
import { Link } from '@tanstack/react-router';
import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { GenerateArticleDrawer } from '@/components/blog/GenerateArticleDrawer';
import { DataTable } from '@/components/blog/data-table/data-table';
import { createColumns } from '@/components/blog/data-table/columns';
import { useState, useMemo, useEffect } from 'react';
import { SortingState } from '@tanstack/react-table';

export type GetArticlesResponse = {
  articles: ArticleListItem[];
  total_pages: number;
  include_drafts: boolean;
};

export const Route = createFileRoute('/dashboard/blog/')({
  component: ArticlesPage,
});

function ArticlesPage() {
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<'all' | 'published' | 'drafts'>('published');
  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState('');
  const [sorting, setSorting] = useState<SortingState>([
    { id: 'article.created_at', desc: true }
  ]);

  // Debounce search query
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearchQuery(searchQuery);
      setPage(1); // Reset to first page on search
    }, 300);

    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Extract sort parameters from sorting state
  const sortBy = sorting.length > 0 ? sorting[0].id.replace('article.', '') : 'created_at';
  const sortOrder = sorting.length > 0 ? (sorting[0].desc ? 'desc' : 'asc') : 'desc';

  // Single query that handles both search and regular listing
  const { data, isLoading, error, refetch } = useQuery<GetArticlesResponse>({
    queryKey: ['articles', page, statusFilter, debouncedSearchQuery, sortBy, sortOrder],
    queryFn: async () => {
      if (debouncedSearchQuery) {
        return searchArticles(
          debouncedSearchQuery,
          page,
          null,
          statusFilter,
          sortBy,
          sortOrder
        ) as Promise<GetArticlesResponse>;
      } else {
        return getArticles(
          page,
          null,
          statusFilter,
          20,
          sortBy,
          sortOrder
        ) as Promise<GetArticlesResponse>;
      }
    },
  });

  const columns = useMemo(() => createColumns(() => refetch()), [refetch]);

  const handlePageChange = (newPage: number) => {
    setPage(newPage);
  };

  const handleStatusFilterChange = (newStatus: 'all' | 'published' | 'drafts') => {
    setStatusFilter(newStatus);
    setPage(1); // Reset to first page on filter change
  };

  const handleSortingChange = (updaterOrValue: SortingState | ((old: SortingState) => SortingState)) => {
    const newSorting = typeof updaterOrValue === 'function' 
      ? updaterOrValue(sorting) 
      : updaterOrValue;
    setSorting(newSorting);
    setPage(1); // Reset to first page on sort change
  };

  if (error) return <div>Error loading articles</div>;

  return (
    <section className="flex flex-col flex-1 p-0 md:p-4 h-full overflow-hidden">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-lg lg:text-2xl font-medium text-gray-900 dark:text-white">
          Articles
        </h1>
        <div className="flex justify-end items-center gap-4">
          <GenerateArticleDrawer>
            <Button>
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

      <Card className="flex flex-col flex-1 overflow-hidden">
        <CardContent className="flex flex-col flex-1 p-6 overflow-hidden">
          <DataTable
            columns={columns}
            data={data?.articles || []}
            totalPages={data?.total_pages || 1}
            currentPage={page}
            onPageChange={handlePageChange}
            onSortingChange={handleSortingChange}
            sorting={sorting}
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            statusFilter={statusFilter}
            onStatusFilterChange={handleStatusFilterChange}
            isLoading={isLoading}
          />
        </CardContent>
      </Card>
    </section>
  );
}
