import { createFileRoute, Link, useRouter } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { listProjects } from '@/services/projects';
import { Card } from '@/components/ui/card';
import { Pagination, PaginationContent, PaginationItem, PaginationLink, PaginationNext, PaginationPrevious } from '@/components/ui/pagination';
import { useEffect, useState } from 'react';

export const Route = createFileRoute('/_publicLayout/projects/')({
  component: ProjectsPage,
});

function ProjectsPage() {
  const router = useRouter();
  const search = new URLSearchParams(router.state.location.search);
  const [page, setPage] = useState<number>(Number(search.get('page')) || 1);

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['public-projects', page],
    queryFn: () => listProjects(page, 12),
  });

  const projects = data?.projects ?? [];
  const total = data?.total ?? 0;
  const perPage = data?.per_page ?? 12;
  const totalPages = Math.max(1, Math.ceil(total / perPage));

  useEffect(() => {
    const params = new URLSearchParams(search);
    params.set('page', String(page));
    window.history.replaceState({}, '', `?${params.toString()}`);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  return (
    <div className="flex-1 p-0 md:p-4">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Projects</h1>
        <Link to="/">
          <span className="text-sm text-primary hover:underline">Back to Home</span>
        </Link>
      </div>

      {isLoading || (isFetching && !data) ? (
        <div>Loading projects...</div>
      ) : projects.length === 0 ? (
        <div className="text-muted-foreground">No projects found.</div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {projects.map((p) => (
            <Card key={p.id} className="group overflow-hidden border">
              <a href={p.url || '#'} target={p.url ? '_blank' : undefined} rel="noreferrer">
                {p.image_url ? (
                  <img src={p.image_url} alt={p.title} className="aspect-video w-full object-cover transition-transform duration-300 group-hover:scale-105" />
                ) : (
                  <div className="aspect-video w-full bg-gray-200 dark:bg-gray-800" />
                )}
                <div className="p-4">
                  <h3 className="font-semibold mb-1">{p.title}</h3>
                  <p className="text-sm text-muted-foreground line-clamp-2">{p.description}</p>
                </div>
              </a>
            </Card>
          ))}
        </div>
      )}

      {totalPages > 1 && (
        <div className="mt-6">
          <Pagination>
            <PaginationContent>
              <PaginationItem>
                <PaginationPrevious onClick={() => page > 1 && setPage(page - 1)} className={page <= 1 ? 'pointer-events-none opacity-50' : 'cursor-pointer'} />
              </PaginationItem>
              {Array.from({ length: totalPages }, (_, i) => i + 1).map((pageNumber) => (
                <PaginationItem key={pageNumber}>
                  <PaginationLink onClick={() => setPage(pageNumber)} isActive={pageNumber === page} className="cursor-pointer">
                    {pageNumber}
                  </PaginationLink>
                </PaginationItem>
              ))}
              <PaginationItem>
                <PaginationNext onClick={() => page < totalPages && setPage(page + 1)} className={page >= totalPages ? 'pointer-events-none opacity-50' : 'cursor-pointer'} />
              </PaginationItem>
            </PaginationContent>
          </Pagination>
        </div>
      )}
    </div>
  );
}


