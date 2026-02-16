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

const projectCardBase = "group flex overflow-hidden border-l-2 border-l-transparent hover:border-l-primary/60 bg-black/30 dark:bg-black/40 backdrop-blur-md border border-white/[0.08] rounded-lg hover:border-primary/30 hover:shadow-[0_0_15px_-5px_rgba(0,200,200,0.12)] transition-all duration-300";

function ProjectsPage() {
  const router = useRouter();
  const search = new URLSearchParams(router.state.location.search);
  const [page, setPage] = useState<number>(Number(search.get('page')) || 1);

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['public-projects', page],
    queryFn: () => listProjects(page, 5),
  });

  const projects = data?.projects ?? [];
  const total = data?.total ?? 0;
  const perPage = data?.per_page ?? 5;
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
          <Link to="/" className="inline-flex items-center gap-2 text-white/60 hover:text-white transition-colors mb-8 group">
            <ArrowLeft className="w-4 h-4 transition-transform group-hover:-translate-x-1" />
            <span className="text-sm font-medium">Back to Home</span>
          </Link>
          <div className="mb-10">
            <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-white">Projects</h1>
            <p className="mt-1 text-sm text-white/40">Things I&apos;ve built and experiments I&apos;ve run</p>
          </div>

          {isLoading || (isFetching && !data) ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="rounded-lg bg-white/[0.03] border border-white/[0.05] overflow-hidden animate-pulse flex">
                  <div className="w-24 aspect-square shrink-0 bg-white/[0.06]" />
                  <div className="flex-1 p-3 space-y-2">
                    <div className="h-4 w-2/3 bg-white/[0.06] rounded" />
                    <div className="h-3 w-full bg-white/[0.04] rounded" />
                  </div>
                </div>
              ))}
            </div>
          ) : projects.length === 0 ? (
            <div className="text-center py-20">
              <div className="mx-auto max-w-sm">
                <div className="w-16 h-16 mx-auto mb-6 rounded-lg bg-white/[0.06] border border-white/[0.06] flex items-center justify-center">
                  <Sparkles className="w-8 h-8 text-white/30" />
                </div>
                <h3 className="text-lg font-semibold text-white mb-2">No projects yet</h3>
                <p className="text-sm text-white/40">Check back soon for new projects.</p>
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {projects.map((project) => (
                <ProjectCard key={project.id} project={project} />
              ))}
            </div>
          )}

          {totalPages > 1 && (
            <div className="mt-12 flex justify-center">
              <div className={cn(
                "rounded-lg p-2",
                "bg-black/30 backdrop-blur-md border border-white/[0.08]"
              )}>
                <Pagination>
                  <PaginationContent className="gap-1">
                    <PaginationItem>
                      <PaginationPrevious
                        onClick={() => page > 1 && setPage(page - 1)}
                        className={cn(
                          "cursor-pointer transition-colors rounded-md border-0",
                          page <= 1 ? 'pointer-events-none opacity-40' : 'hover:bg-white/10 text-white/70'
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
                              : 'hover:bg-white/10 text-white/70'
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
                          page >= totalPages ? 'pointer-events-none opacity-40' : 'hover:bg-white/10 text-white/70'
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
      <div className="relative w-24 sm:w-28 shrink-0 aspect-square overflow-hidden">
        {project.image_url ? (
          <img
            src={project.image_url}
            alt={project.title}
            className="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
          />
        ) : (
          <div className="w-full h-full bg-white/[0.04] flex items-center justify-center">
            <Sparkles className="w-6 h-6 text-white/15" />
          </div>
        )}
        {project.url && (
          <div className="absolute bottom-1.5 right-1.5 w-6 h-6 rounded bg-black/60 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
            <svg className="w-2.5 h-2.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
          </div>
        )}
      </div>
      <div className="flex-1 p-3 flex flex-col min-w-0">
        <h3 className="text-sm font-semibold tracking-tight text-white group-hover:text-primary transition-colors line-clamp-1">
          {project.title}
        </h3>
        <p className="text-xs text-white/40 line-clamp-2 leading-relaxed mt-0.5 flex-1">
          {project.description}
        </p>
        <span className="mt-2 text-[11px] text-primary/80 font-medium opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
          View
          <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </span>
      </div>
    </Link>
  );
}


