import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { createFileRoute, Link } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { listProjects, deleteProject } from '@/services/projects';
import { useState } from 'react';
import { Plus, Trash2, Pencil } from 'lucide-react';

export const Route = createFileRoute('/dashboard/projects/')({
  component: ProjectsPage,
});

function ProjectsPage() {
  const [page, setPage] = useState(1);
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['projects', page],
    queryFn: () => listProjects(page, 20),
  });

  const projects = data?.projects ?? [];

  if (isLoading) return <div>Loading projects...</div>;
  if (error) return <div>Error loading projects</div>;

  return (
    <section className="flex-1 p-0 md:p-4">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-lg lg:text-2xl font-medium text-gray-900 dark:text-white">Projects</h1>
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
        </CardContent>
      </Card>
    </section>
  );
}
