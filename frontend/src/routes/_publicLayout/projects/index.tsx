import { createFileRoute, Link, useRouter } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { listProjects } from '@/services/projects';
import { Pagination, PaginationContent, PaginationItem, PaginationLink, PaginationNext, PaginationPrevious } from '@/components/ui/pagination';
import { Card, CardContent } from '@/components/ui/card';
import { useEffect, useState } from 'react';
import { ExternalLink, ArrowLeft, Sparkles } from 'lucide-react';

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
    <div className="min-h-screen">
      {/* Hero Section */}
      <div className="relative overflow-hidden">
        <div className="absolute inset-0" />
        <div className="relative px-6 py-24 sm:py-32 lg:px-8">
          <div className="mx-auto max-w-4xl text-left">
            <Link to="/" className="inline-flex items-center gap-2 text-white/80 hover:text-white transition-colors mb-8 group">
              <ArrowLeft className="w-4 h-4 transition-transform group-hover:-translate-x-1" />
              Back to Home
            </Link>
            <div className="flex items-center justify-start gap-3 mb-6">
              <Sparkles className="w-4 h-4 text-yellow-400" />
              <p className="text-3xl font-bold tracking-tight text-zinc-900 dark:text-zinc-300">
                Projects
              </p>
            </div>
          </div>
        </div>
        <div className="absolute bottom-0 left-0 right-0 h-20" />
      </div>

      {/* Projects Grid */}
      <div className="relative px-6 py-16 lg:px-8">
        <div className="mx-auto max-w-7xl">
          {isLoading || (isFetching && !data) ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
              {Array.from({ length: 6 }).map((_, i) => (
                <Card key={i} animationDelay={i * 100} className="group relative overflow-hidden">
                  <div className="aspect-video w-full bg-gradient-to-br from-slate-200 to-slate-300 dark:from-slate-700 dark:to-slate-600 animate-pulse" />
                  <CardContent className="space-y-3">
                    <div className="h-6 bg-slate-200 dark:bg-slate-600 rounded animate-pulse" />
                    <div className="space-y-2">
                      <div className="h-4 bg-slate-200 dark:bg-slate-600 rounded animate-pulse" />
                      <div className="h-4 bg-slate-200 dark:bg-slate-600 rounded w-3/4 animate-pulse" />
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : projects.length === 0 ? (
            <div className="text-center py-16">
              <div className="mx-auto max-w-md">
                <div className="w-24 h-24 mx-auto mb-6 rounded-full bg-muted flex items-center justify-center">
                  <Sparkles className="w-12 h-12 text-muted-foreground" />
                </div>
                <h3 className="text-lg font-semibold mb-2">No projects yet</h3>
                <p className="text-muted-foreground">Check back soon for exciting new projects!</p>
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
              {projects.map((project, index) => (
                <Card
                  key={project.id}
                  animationDelay={index * 100}
                  className="group relative overflow-hidden hover:shadow-2xl transition-all duration-500 hover:border-primary/50 transform hover:-translate-y-2"
                >
                  <Link 
                    to="/projects/$projectId" 
                    params={{ projectId: project.id }}
                    className="block relative"
                  >
                    {/* Image */}
                    <div className="relative aspect-video overflow-hidden">
                      {project.image_url ? (
                        <img 
                          src={project.image_url} 
                          alt={project.title} 
                          className="w-full h-full object-cover transition-transform duration-700 group-hover:scale-110" 
                        />
                      ) : (
                        <div className="w-full h-full bg-muted flex items-center justify-center">
                          <Sparkles className="w-12 h-12 text-muted-foreground" />
                        </div>
                      )}
                      <div className="absolute inset-0 bg-gradient-to-t from-black/20 via-transparent to-transparent" />
                      
                      {/* External link indicator */}
                      {project.url && (
                        <div className="absolute top-4 right-4 w-8 h-8 bg-background/90 rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transition-all duration-300 transform translate-x-2 group-hover:translate-x-0">
                          <ExternalLink className="w-4 h-4" />
                        </div>
                      )}
                    </div>

                    {/* Content */}
                    <CardContent>
                      <h3 className="text-xl font-semibold mb-3 group-hover:text-primary transition-colors">
                        {project.title}
                      </h3>
                      <p className="text-muted-foreground leading-relaxed line-clamp-3">
                        {project.description}
                      </p>
                      
                      {/* Hover arrow */}
                      <div className="flex items-center gap-2 mt-4 text-primary font-medium opacity-0 group-hover:opacity-100 transition-all duration-300 transform translate-y-2 group-hover:translate-y-0">
                        <span className="text-sm">View Project</span>
                        <ExternalLink className="w-4 h-4" />
                      </div>
                    </CardContent>
                  </Link>
                </Card>
              ))}
            </div>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="mt-16 flex justify-center">
              <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-lg border border-slate-200 dark:border-slate-700 p-2">
                <Pagination>
                  <PaginationContent>
                    <PaginationItem>
                      <PaginationPrevious 
                        onClick={() => page > 1 && setPage(page - 1)} 
                        className={`${page <= 1 ? 'pointer-events-none opacity-50' : 'cursor-pointer hover:bg-indigo-50 dark:hover:bg-indigo-950'} transition-colors rounded-xl`} 
                      />
                    </PaginationItem>
                    {Array.from({ length: totalPages }, (_, i) => i + 1).map((pageNumber) => (
                      <PaginationItem key={pageNumber}>
                        <PaginationLink
                          onClick={() => setPage(pageNumber)}
                          isActive={pageNumber === page}
                          className={`cursor-pointer transition-all duration-200 rounded-xl ${
                            pageNumber === page 
                              ? 'bg-indigo-600 text-white hover:bg-indigo-700' 
                              : 'hover:bg-indigo-50 dark:hover:bg-indigo-950'
                          }`}
                        >
                          {pageNumber}
                        </PaginationLink>
                      </PaginationItem>
                    ))}
                    <PaginationItem>
                      <PaginationNext 
                        onClick={() => page < totalPages && setPage(page + 1)} 
                        className={`${page >= totalPages ? 'pointer-events-none opacity-50' : 'cursor-pointer hover:bg-indigo-50 dark:hover:bg-indigo-950'} transition-colors rounded-xl`} 
                      />
                    </PaginationItem>
                  </PaginationContent>
                </Pagination>
              </div>
            </div>
          )}
        </div>
      </div>

      <style>{`
        @keyframes fadeInUp {
          from {
            opacity: 0;
            transform: translateY(30px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }
      `}</style>
    </div>
  );
}


