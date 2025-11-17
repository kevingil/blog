import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Plus } from 'lucide-react';
import { getAllPages, Page } from '@/services/pages';
import { Link } from '@tanstack/react-router';
import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { DataTable } from '@/components/blog/data-table/data-table';
import { createColumns } from '@/components/pages/data-table/columns';
import { useState, useMemo, useEffect } from 'react';
import { SortingState } from '@tanstack/react-table';
import { Input } from '@/components/ui/input';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Filter, X } from 'lucide-react';
import { useAdminDashboard } from '@/services/dashboard/dashboard';

export const Route = createFileRoute('/dashboard/pages/')({
  component: PagesPage,
});

function PagesPage() {
  const [page, setPage] = useState(1);
  const { setPageTitle } = useAdminDashboard();

  useEffect(() => {
    setPageTitle("Pages");
  }, [setPageTitle]);
  const [statusFilter, setStatusFilter] = useState<'all' | 'published' | 'draft'>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [sorting, setSorting] = useState<SortingState>([
    { id: 'updated_at', desc: true }
  ]);

  // Determine is_published filter based on statusFilter
  const isPublishedFilter = statusFilter === 'all' ? undefined : statusFilter === 'published';

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['pages', page, isPublishedFilter],
    queryFn: () => getAllPages(page, 20, isPublishedFilter),
  });

  const columns = useMemo(() => createColumns(() => refetch()), [refetch]);

  // Filter pages by search query (client-side for now)
  const filteredPages = useMemo(() => {
    if (!data?.pages) return [];
    if (!searchQuery) return data.pages;
    
    const query = searchQuery.toLowerCase();
    return data.pages.filter(
      (p: Page) =>
        p.title.toLowerCase().includes(query) ||
        p.slug.toLowerCase().includes(query) ||
        (p.description && p.description.toLowerCase().includes(query))
    );
  }, [data?.pages, searchQuery]);

  const handlePageChange = (newPage: number) => {
    setPage(newPage);
  };

  const handleStatusFilterChange = (newStatus: 'all' | 'published' | 'draft') => {
    setStatusFilter(newStatus);
    setPage(1);
  };

  const handleSortingChange = (updaterOrValue: SortingState | ((old: SortingState) => SortingState)) => {
    const newSorting = typeof updaterOrValue === 'function' 
      ? updaterOrValue(sorting) 
      : updaterOrValue;
    setSorting(newSorting);
  };

  const isFiltered = searchQuery.length > 0 || statusFilter !== 'all';

  if (error) return <div>Error loading pages</div>;

  return (
    <section className="flex flex-col flex-1 p-0 md:p-4 h-full overflow-hidden">
      <Card className="flex flex-col flex-1 overflow-hidden">
        <CardContent className="flex flex-col flex-1 py-0 px-6 overflow-hidden">
          {/* Custom toolbar for pages */}
          <div className="flex items-center justify-between py-4">
            <div className="flex flex-1 items-center space-x-2">
              <Input
                placeholder="Search pages..."
                value={searchQuery}
                onChange={(event) => setSearchQuery(event.target.value)}
                className="h-9 w-[150px] lg:w-[250px]"
              />
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="sm" className="h-9 border-dashed">
                    <Filter className="mr-2 h-4 w-4" />
                    Status
                    {statusFilter !== 'all' && (
                      <span className="ml-2 rounded-sm bg-primary px-1 text-[0.6rem] font-semibold text-primary-foreground">
                        {statusFilter === 'published' ? 'Published' : 'Draft'}
                      </span>
                    )}
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="w-[150px]">
                  <DropdownMenuRadioGroup
                    value={statusFilter}
                    onValueChange={(value) =>
                      handleStatusFilterChange(value as 'all' | 'published' | 'draft')
                    }
                  >
                    <DropdownMenuRadioItem value="all">All</DropdownMenuRadioItem>
                    <DropdownMenuRadioItem value="published">
                      Published
                    </DropdownMenuRadioItem>
                    <DropdownMenuRadioItem value="draft">
                      Draft
                    </DropdownMenuRadioItem>
                  </DropdownMenuRadioGroup>
                </DropdownMenuContent>
              </DropdownMenu>
              {isFiltered && (
                <Button
                  variant="ghost"
                  onClick={() => {
                    setSearchQuery("");
                    setStatusFilter("all");
                  }}
                  className="h-9 px-2 lg:px-3"
                >
                  Reset
                  <X className="ml-2 h-4 w-4" />
                </Button>
              )}
            </div>
            <div className="flex items-center gap-4">
              <Link to="/dashboard/pages/new">
                <Button>
                  <Plus className="mr-2 h-4 w-4" />
                  New Page
                </Button>
              </Link>
            </div>
          </div>

          {/* Reuse DataTable component */}
          <div className="flex-1 rounded-md border overflow-auto relative">
            <table className="w-full caption-bottom text-sm">
              <thead className="sticky top-0 bg-background z-10 border-b">
                {columns && filteredPages && (
                  <tr className="border-b transition-colors hover:bg-muted/50">
                    {columns.map((column: any, idx) => (
                      <th key={idx} className="h-10 px-2 text-left align-middle font-medium bg-background">
                        {typeof column.header === 'function' 
                          ? column.header({ column: { getIsSorted: () => false, toggleSorting: () => {} } })
                          : column.header
                        }
                      </th>
                    ))}
                  </tr>
                )}
              </thead>
              <tbody>
                {isLoading ? (
                  <tr>
                    <td colSpan={columns.length} className="h-24 text-center">
                      Loading...
                    </td>
                  </tr>
                ) : filteredPages.length > 0 ? (
                  filteredPages.map((pageData: Page) => (
                    <tr key={pageData.id} className="border-b transition-colors hover:bg-muted/50">
                      {columns.map((column: any, idx) => (
                        <td key={idx} className="p-2 align-middle">
                          {column.cell({ row: { original: pageData } })}
                        </td>
                      ))}
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={columns.length} className="h-24 text-center">
                      No pages found.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {/* Simple pagination */}
          <div className="flex items-center justify-between px-2 py-4">
            <div className="text-sm text-muted-foreground">
              Showing {filteredPages.length} of {data?.total || 0} pages
            </div>
            <div className="flex items-center space-x-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => handlePageChange(page - 1)}
                disabled={page === 1}
              >
                Previous
              </Button>
              <span className="text-sm">
                Page {page} of {data?.total_pages || 1}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handlePageChange(page + 1)}
                disabled={page === (data?.total_pages || 1)}
              >
                Next
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </section>
  );
}

