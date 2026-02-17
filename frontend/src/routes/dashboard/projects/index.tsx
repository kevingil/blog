import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationPrevious,
  PaginationNext,
} from '@/components/ui/pagination';
import { createFileRoute, Link } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { listProjects, deleteProject } from '@/services/projects';
import { useState, useEffect } from 'react';
import { Plus, Trash2, Pencil } from 'lucide-react';
import { useAdminDashboard } from '@/services/dashboard/dashboard';

export const Route = createFileRoute('/dashboard/projects/')({
  component: ProjectsPage,
});

const PER_PAGE = 20;

function ProjectsPage() {
  const [page, setPage] = useState(1);
  const { setPageTitle } = useAdminDashboard();

  useEffect(() => {
    setPageTitle("Projects");
  }, [setPageTitle]);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['projects', page],
    queryFn: () => listProjects(page, PER_PAGE),
  });

  const projects = data?.projects ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PER_PAGE));

  const handlePageChange = (newPage: number) => {
    setPage(newPage);
  };

  if (isLoading) return <div>Loading projects...</div>;
  if (error) return <div>Error loading projects</div>;

  return (
    <section className="flex-1 min-h-0 overflow-auto p-0 md:p-4">
      <div className="flex justify-end items-center mb-6">
        <Link to="/dashboard/projects/new">
          <Button>
            <Plus className="mr-2 h-4 w-4" /> New Project
          </Button>
        </Link>
      </div>

      <Card>
        <CardContent className="p-0">
          <div className="divide-y">
            {projects.map((p) => (
              <div key={p.id} className="flex items-center justify-between p-4 gap-4">
                <div className="flex items-start gap-4">
                  {p.image_url ? (
                    <img src={p.image_url} alt={p.title} className="w-16 h-16 rounded object-cover" />
                  ) : (
                    <div className="w-16 h-16 rounded bg-muted" />
                  )}
                  <div>
                    <h2 className="font-semibold">{p.title}</h2>
                    <p className="text-sm text-muted-foreground line-clamp-2">{p.description}</p>
                    <p className="text-xs text-muted-foreground mt-1">{new Date(p.created_at).toLocaleDateString()}</p>
                    {p.url && (
                      <a href={p.url} target="_blank" rel="noreferrer" className="text-xs text-primary hover:underline">
                        {p.url}
                      </a>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Link to="/dashboard/projects/edit/$projectId" params={{ projectId: p.id }}>
                    <Button variant="outline" size="icon"><Pencil className="w-4 h-4"/></Button>
                  </Link>
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={async () => {
                      const res = await deleteProject(p.id);
                      if (res.success) refetch();
                    }}
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-between px-4 py-4 border-t">
              <div className="text-sm text-muted-foreground">
                Showing {(page - 1) * PER_PAGE + 1}-{Math.min(page * PER_PAGE, total)} of {total} projects
              </div>
              <Pagination>
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious
                      onClick={() => page > 1 && handlePageChange(page - 1)}
                      className={page <= 1 ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
                    />
                  </PaginationItem>
                  {Array.from({ length: totalPages }, (_, i) => i + 1).map((pageNumber) => (
                    <PaginationItem key={pageNumber}>
                      <PaginationLink
                        onClick={() => handlePageChange(pageNumber)}
                        isActive={pageNumber === page}
                        className="cursor-pointer"
                      >
                        {pageNumber}
                      </PaginationLink>
                    </PaginationItem>
                  ))}
                  <PaginationItem>
                    <PaginationNext
                      onClick={() => page < totalPages && handlePageChange(page + 1)}
                      className={page >= totalPages ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
                    />
                  </PaginationItem>
                </PaginationContent>
              </Pagination>
            </div>
          )}
        </CardContent>
      </Card>
    </section>
  );
}
