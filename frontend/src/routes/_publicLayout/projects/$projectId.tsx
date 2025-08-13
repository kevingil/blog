import { createFileRoute, Link, useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { getProject } from '@/services/projects';
import { format } from 'date-fns';
import { Card, CardContent } from '@/components/ui/card';
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

  const { data: project, isLoading, isFetching, error } = useQuery({
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

        {(isLoading || (isFetching && !project)) && <ProjectSkeleton />}

        {error && (
          <div className="text-center py-16">
            <p className="text-lg font-semibold mb-2">Project not found</p>
            <p className="text-muted-foreground">The project you are looking for does not exist.</p>
          </div>
        )}

        {!isLoading && project && (
          <Card className="overflow-hidden">
            <div className="relative">
              {project.image_url && (
                <img
                  src={project.image_url}
                  alt={project.title}
                  className="w-full object-cover aspect-video"
                />
              )}
            </div>
            <CardContent className="p-6">
              <h1 className="text-3xl font-bold mb-3">{project.title}</h1>
              <p className="text-sm text-muted-foreground mb-6">
                {(() => {
                  const date = project.created_at ? new Date(project.created_at) : null;
                  if (!date || isNaN(date.getTime())) return 'Unknown date';
                  const year = date.getFullYear();
                  if (year > 2100) return 'Unknown date';
                  return format(date, 'MMMM d, yyyy');
                })()}
              </p>
              <p className="text-muted-foreground leading-relaxed whitespace-pre-line">
                {project.description}
              </p>

              {project.url && (
                <div className="mt-8">
                  <Button asChild>
                    <a href={project.url} target="_blank" rel="noreferrer" className="inline-flex items-center gap-2">
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


