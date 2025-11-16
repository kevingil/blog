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
import { ChipInput } from '@/components/ui/chip-input';
import { useAdminDashboard } from '@/services/dashboard/dashboard';

export const Route = createFileRoute('/dashboard/projects/edit/$projectId')({
  component: EditProjectPage,
});

const schema = z.object({
  title: z.string().min(1, 'Title is required'),
  description: z.string().min(1, 'Description is required'),
  content: z.string().optional(),
  tags: z.array(z.string()),
  image_url: z.string().url().optional().or(z.literal('')),
  url: z.string().url().optional().or(z.literal('')),
  created_at: z.string().optional(),
});

type FormData = z.infer<typeof schema>;

function EditProjectPage() {
  const { projectId } = useParams({ from: '/dashboard/projects/edit/$projectId' });
  const navigate = useNavigate();
  const { setPageTitle } = useAdminDashboard();

  useEffect(() => {
    setPageTitle("Edit Project");
  }, [setPageTitle]);

  const { data: detail, isLoading, error } = useQuery({
    queryKey: ['project', projectId],
    queryFn: () => getProject(projectId),
  });

  const { register, handleSubmit, formState: { errors, isSubmitting }, reset, setValue, watch } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: { title: '', description: '', content: '', tags: [], image_url: '', url: '', created_at: '' },
  });
  const watchedTags = watch('tags');

  // Sync fetched data to form once after load
  // Avoid calling reset during render to prevent render loops
  useEffect(() => {
    if (detail) {
      reset({
        title: detail.project.title,
        description: detail.project.description,
        content: detail.project.content || '',
        tags: detail.tags || [],
        image_url: detail.project.image_url || '',
        url: detail.project.url || '',
        created_at: detail.project.created_at ? detail.project.created_at.slice(0, 10) : '',
      });
    }
  }, [detail, reset]);

  const onSubmit = async (data: FormData) => {
    await updateProject(projectId, {
      title: data.title,
      description: data.description,
      content: data.content,
      tags: data.tags,
      image_url: data.image_url || undefined,
      url: data.url || undefined,
      created_at: data.created_at ? new Date(`${data.created_at}T00:00:00Z`).toISOString() : undefined,
    });
    navigate({ to: '/dashboard/projects' });
  };

  return (
    <section className="flex-1 p-0 md:p-4">
      <h1 className="text-lg lg:text-2xl font-medium text-gray-900 dark:text-white mb-6">Edit Project</h1>
      {isLoading ? (
        <div>Loading project...</div>
      ) : (error || !detail) ? (
        <div>Error loading project</div>
      ) : (
      <Card>
        <CardContent className="p-4">
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div>
              <label className="block text-sm font-medium">Title</label>
              <Input {...register('title')} />
              {errors.title && <p className="text-sm text-red-500">{errors.title.message}</p>}
            </div>
            <div>
              <label className="block text-sm font-medium">Short Description</label>
              <Textarea rows={4} {...register('description')} />
              {errors.description && <p className="text-sm text-red-500">{errors.description.message}</p>}
            </div>
            <div>
              <label className="block text-sm font-medium">README Content</label>
              <Textarea rows={12} placeholder="Longer README-style content" {...register('content')} />
              {errors.content && <p className="text-sm text-red-500">{errors.content.message as string}</p>}
            </div>
            <div>
              <label className="block text-sm font-medium">Tags</label>
              <ChipInput
                value={watchedTags ?? []}
                onChange={(tags) => setValue('tags', tags)}
                placeholder="Type and press Enter to add tags..."
              />
              {errors.tags && <p className="text-sm text-red-500">{errors.tags.message as string}</p>}
            </div>
            <div>
              <label className="block text-sm font-medium">Date</label>
              <Input type="date" {...register('created_at')} />
              {errors.created_at && <p className="text-sm text-red-500">{errors.created_at.message as string}</p>}
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
      )}
    </section>
  );
}


