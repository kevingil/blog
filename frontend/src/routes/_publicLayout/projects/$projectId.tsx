import { createFileRoute, Link, useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { getProject } from '@/services/projects';
import { format } from 'date-fns';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { ArrowLeft, ExternalLink } from 'lucide-react';

export const Route = createFileRoute('/_publicLayout/projects/$projectId')({
  component: ProjectPage,
});

function ProjectSkeleton() {
  return (
    <div className="max-w-4xl mx-auto">
      <Skeleton className="h-10 w-2/3 mb-4" />
      <Skeleton className="h-64 w-full mb-6" />
      <Skeleton className="h-4 w-full mb-2" />
      <Skeleton className="h-4 w-5/6 mb-2" />
      <Skeleton className="h-4 w-4/6 mb-6" />
    </div>
  );
}

function ProjectPage() {
  const { projectId } = useParams({ from: '/_publicLayout/projects/$projectId' });

  const { data: detail, isLoading, isFetching, error } = useQuery({
    queryKey: ['public-project', projectId],
    queryFn: () => getProject(projectId),
  });

  return (
    <div className="container mx-auto px-6 py-12">
      <div className="max-w-4xl mx-auto">
        <Link to="/projects/" className="inline-flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors mb-6 group">
          <ArrowLeft className="w-4 h-4 transition-transform group-hover:-translate-x-1" />
          Back to Projects
        </Link>

        {(isLoading || (isFetching && !detail)) && <ProjectSkeleton />}

        {error && (
          <div className="text-center py-16">
            <p className="text-lg font-semibold mb-2">Project not found</p>
            <p className="text-muted-foreground">The project you are looking for does not exist.</p>
          </div>
        )}

        {!isLoading && detail && (
          <Card className="group relative overflow-hidden shadow-sm border hover:shadow-md transition-shadow">
            <div className="relative p-2">
              {detail.project.image_url && (
                <div className="relative">
                  <img
                    src={detail.project.image_url}
                    alt={detail.project.title}
                    className="w-full object-contain aspect-video"
                  />
                  <div className="absolute inset-0 bg-gradient-to-t from-black/20 via-transparent to-transparent" />
                </div>
              )}
            </div>
            <CardContent className="p-6">
              <h1 className="text-3xl font-bold mb-3 group-hover:text-primary transition-colors">{detail.project.title}</h1>
              <p className="text-sm text-muted-foreground mb-6">
                {(() => {
                  const date = detail.project.created_at ? new Date(detail.project.created_at) : null;
                  if (!date || isNaN(date.getTime())) return 'Unknown date';
                  return format(date, 'MMM d, yyyy');
                })()}
              </p>
              <p className="text-muted-foreground leading-relaxed mb-6">
                {detail.project.description}
              </p>

              {detail.tags && detail.tags.length > 0 && (
                <div className="flex flex-wrap gap-2 mb-6">
                  {detail.tags.map((t) => (
                    <Badge key={t} variant="secondary" className="text-primary">
                      {t.toUpperCase()}
                    </Badge>
                  ))}
                </div>
              )}

              {detail.project.content && (
                <div className="prose dark:prose-invert max-w-none border-t pt-6">
                  <h2 className="text-xl font-semibold mb-3">README</h2>
                  <div className="text-foreground/90 whitespace-pre-wrap">
                    {detail.project.content}
                  </div>
                </div>
              )}

              {detail.project.url && (
                <div className="mt-8">
                  <Button asChild>
                    <a href={detail.project.url} target="_blank" rel="noreferrer" className="inline-flex items-center gap-2">
                      Visit Project
                      <ExternalLink className="w-4 h-4" />
                    </a>
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}

export default ProjectPage;


