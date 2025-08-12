import { createFileRoute, useNavigate, useParams } from '@tanstack/react-router';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { getProject, updateProject } from '@/services/projects';
import { useQuery } from '@tanstack/react-query';

export const Route = createFileRoute('/dashboard/projects/edit/$projectId')({
  component: EditProjectPage,
});

const schema = z.object({
  title: z.string().min(1, 'Title is required'),
  description: z.string().min(1, 'Description is required'),
  image_url: z.string().url().optional().or(z.literal('')),
  url: z.string().url().optional().or(z.literal('')),
});

type FormData = z.infer<typeof schema>;

function EditProjectPage() {
  const { projectId } = useParams({ from: '/dashboard/projects/edit/$projectId' });
  const navigate = useNavigate();

  const { data: project, isLoading, error } = useQuery({
    queryKey: ['project', projectId],
    queryFn: () => getProject(projectId),
  });

  const { register, handleSubmit, formState: { errors, isSubmitting }, reset } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: { title: '', description: '', image_url: '', url: '' },
  });

  if (isLoading) return <div>Loading project...</div>;
  if (error || !project) return <div>Error loading project</div>;

  // Sync fetched data to form once after load
  // Avoid calling reset during render to prevent render loops
  useEffect(() => {
    if (project) {
      reset({
        title: project.title,
        description: project.description,
        image_url: project.image_url || '',
        url: project.url || '',
      });
    }
  }, [project, reset]);

  const onSubmit = async (data: FormData) => {
    await updateProject(projectId, {
      title: data.title,
      description: data.description,
      image_url: data.image_url || undefined,
      url: data.url || undefined,
    });
    navigate({ to: '/dashboard/projects' });
  };

  return (
    <section className="flex-1 p-0 md:p-4">
      <h1 className="text-lg lg:text-2xl font-medium text-gray-900 dark:text-white mb-6">Edit Project</h1>
      <Card>
        <CardContent className="p-4">
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div>
              <label className="block text-sm font-medium">Title</label>
              <Input {...register('title')} />
              {errors.title && <p className="text-sm text-red-500">{errors.title.message}</p>}
            </div>
            <div>
              <label className="block text-sm font-medium">Description</label>
              <Textarea rows={6} {...register('description')} />
              {errors.description && <p className="text-sm text-red-500">{errors.description.message}</p>}
            </div>
            <div>
              <label className="block text-sm font-medium">Image URL</label>
              <Input {...register('image_url')} />
              {errors.image_url && <p className="text-sm text-red-500">{errors.image_url.message as string}</p>}
            </div>
            <div>
              <label className="block text-sm font-medium">Project URL</label>
              <Input {...register('url')} />
              {errors.url && <p className="text-sm text-red-500">{errors.url.message as string}</p>}
            </div>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => navigate({ to: '/dashboard/projects' })}>Cancel</Button>
              <Button type="submit" disabled={isSubmitting}>Save</Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </section>
  );
}


