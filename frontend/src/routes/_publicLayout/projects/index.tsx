import { createFileRoute, Link, useRouter } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { listProjects, type Project } from '@/services/projects';
import { Pagination, PaginationContent, PaginationItem, PaginationLink, PaginationNext, PaginationPrevious } from '@/components/ui/pagination';
import { useEffect, useState } from 'react';
import { ArrowLeft, Sparkles } from 'lucide-react';
import { cn } from '@/lib/utils';

export const Route = createFileRoute('/_publicLayout/projects/')({
  component: ProjectsPage,
});

const projectCardBase = "group flex flex-row-reverse overflow-hidden bg-card/90 dark:bg-card/80 backdrop-blur-md border border-border hover:border-primary hover:shadow-[0_0_25px_rgba(249,115,22,0.22)] transition-all duration-200";

function ProjectsPage() {
  const router = useRouter();
  const search = new URLSearchParams(router.state.location.search);
  const [page, setPage] = useState<number>(Number(search.get('page')) || 1);

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['public-projects', page],
    queryFn: () => listProjects(page, 8),
  });

  const projects = data?.projects ?? [];
  const total = data?.total ?? 0;
  const perPage = data?.per_page ?? 8;
  const totalPages = Math.max(1, Math.ceil(total / perPage));

  useEffect(() => {
    const params = new URLSearchParams(search);
    params.set('page', String(page));
    window.history.replaceState({}, '', `?${params.toString()}`);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  return (
    <div className="min-h-screen">
      <div className="relative px-4 sm:px-6 py-8 lg:px-8">
        <div className="mx-auto max-w-7xl">
          <Link to="/" className="inline-flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors mb-8 group">
            <ArrowLeft className="w-4 h-4 transition-transform group-hover:-translate-x-1" />
            <span className="text-sm font-medium">Back to Home</span>
          </Link>
          <div className="mb-10">
            <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground">Projects</h1>
            <p className="mt-1 text-sm text-muted-foreground">Fun experiments, hackathon projects, and other stuff</p>
          </div>

          {isLoading || (isFetching && !data) ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
              {Array.from({ length: 8 }).map((_, i) => (
                <div key={i} className="bg-card/80 border border-border overflow-hidden animate-pulse flex flex-row-reverse">
                  <div className="w-44 aspect-[4/3] shrink-0 bg-muted/60" />
                  <div className="flex-1 p-5 space-y-2">
                    <div className="h-6 w-2/3 bg-muted/60" />
                    <div className="h-4 w-full bg-muted/40" />
                  </div>
                </div>
              ))}
            </div>
          ) : projects.length === 0 ? (
            <div className="text-center py-20">
              <div className="mx-auto max-w-sm">
                <div className="w-16 h-16 mx-auto mb-6 rounded-lg bg-muted/40 border border-border flex items-center justify-center">
                  <Sparkles className="w-8 h-8 text-muted-foreground/50" />
                </div>
                <h3 className="text-lg font-semibold text-foreground mb-2">No projects yet</h3>
                <p className="text-sm text-muted-foreground">Check back soon for new projects.</p>
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
              {projects.map((project) => (
                <ProjectCard key={project.id} project={project} />
              ))}
            </div>
          )}

          {totalPages > 1 && (
            <div className="mt-12 flex justify-center">
              <div className={cn(
                "rounded-lg p-2",
                "bg-card/90 dark:bg-card/80 backdrop-blur-md border border-border"
              )}>
                <Pagination>
                  <PaginationContent className="gap-1">
                    <PaginationItem>
                      <PaginationPrevious
                        onClick={() => page > 1 && setPage(page - 1)}
                        className={cn(
                          "cursor-pointer transition-colors rounded-md border-0",
                          page <= 1 ? 'pointer-events-none opacity-40' : 'text-muted-foreground hover:bg-accent hover:text-foreground'
                        )}
                      />
                    </PaginationItem>
                    {Array.from({ length: totalPages }, (_, i) => i + 1).map((pageNumber) => (
                      <PaginationItem key={pageNumber}>
                        <PaginationLink
                          onClick={() => setPage(pageNumber)}
                          isActive={pageNumber === page}
                          className={cn(
                            "cursor-pointer transition-all duration-200 rounded-md border-0",
                            pageNumber === page
                              ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                              : 'text-muted-foreground hover:bg-accent hover:text-foreground'
                          )}
                        >
                          {pageNumber}
                        </PaginationLink>
                      </PaginationItem>
                    ))}
                    <PaginationItem>
                      <PaginationNext
                        onClick={() => page < totalPages && setPage(page + 1)}
                        className={cn(
                          "cursor-pointer transition-colors rounded-md border-0",
                          page >= totalPages ? 'pointer-events-none opacity-40' : 'text-muted-foreground hover:bg-accent hover:text-foreground'
                        )}
                      />
                    </PaginationItem>
                  </PaginationContent>
                </Pagination>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function ProjectCard({ project }: { project: Project }) {
  return (
    <Link
      to="/projects/$projectId"
      params={{ projectId: project.id }}
      className={projectCardBase}
    >
      <div className="relative w-44 shrink-0 aspect-[4/3] overflow-hidden">
        {project.image_url ? (
          <img
            src={project.image_url}
            alt={project.title}
            className="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
          />
        ) : (
          <div className="w-full h-full bg-muted/40 flex items-center justify-center">
            <Sparkles className="w-10 h-10 text-muted-foreground/40" />
          </div>
        )}
        {project.url && (
          <div className="absolute top-0.5 right-0.5 w-4 h-4 bg-foreground/70 text-background flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
            <svg className="w-1.5 h-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
          </div>
        )}
      </div>
      <div className="flex-1 p-5 flex flex-col min-w-0">
        <h3 className="text-xl font-semibold tracking-tight text-foreground group-hover:text-primary transition-colors line-clamp-1">
          {project.title}
        </h3>
        <p className="text-base text-muted-foreground line-clamp-2 leading-relaxed mt-2">
          {project.description}
        </p>
      </div>
    </Link>
  );
}
