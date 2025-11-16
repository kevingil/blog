import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { createProject } from '@/services/projects';
import { ChipInput } from '@/components/ui/chip-input';
import { useEffect } from 'react';
import { useAdminDashboard } from '@/services/dashboard/dashboard';

export const Route = createFileRoute('/dashboard/projects/new')({
  component: NewProjectPage,
});

const schema = z.object({
  title: z.string().min(1, 'Title is required'),
  description: z.string().min(1, 'Description is required'),
  content: z.string().optional(),
  tags: z.array(z.string()).default([]),
  image_url: z.string().url().optional().or(z.literal('')),
  url: z.string().url().optional().or(z.literal('')),
});

type FormData = z.infer<typeof schema>;

function NewProjectPage() {
  const navigate = useNavigate();
  const { setPageTitle } = useAdminDashboard();
  const { register, handleSubmit, formState: { errors, isSubmitting }, setValue, watch } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: { title: '', description: '', content: '', tags: [], image_url: '', url: '' },
  });
  const watchedTags = watch('tags');

  useEffect(() => {
    setPageTitle("New Project");
  }, [setPageTitle]);

  const onSubmit = async (data: FormData) => {
    await createProject({
      title: data.title,
      description: data.description,
      content: data.content,
      tags: data.tags,
      image_url: data.image_url || undefined,
      url: data.url || undefined,
    });
    navigate({ to: '/dashboard/projects' });
  };

  return (
    <section className="flex-1 p-0 md:p-4">
      <h1 className="text-lg lg:text-2xl font-medium text-gray-900 dark:text-white mb-6">New Project</h1>
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
                value={watchedTags}
                onChange={(tags) => setValue('tags', tags)}
                placeholder="Type and press Enter to add tags..."
              />
              {errors.tags && <p className="text-sm text-red-500">{errors.tags.message as string}</p>}
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
              <Button type="submit" disabled={isSubmitting}>Create</Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </section>
  );
}


