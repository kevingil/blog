'use server'

import ArticleEditor from '@/components/blog/Editor';
import { getUser } from '@/services/user';
import { redirect } from '@tanstack/react-router';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/dashboard/blog/new')({
  component: NewArticlePage,
});

async function NewArticlePage() {
  const user = await getUser();

  if (!user) {
    redirect({ to: '/login' });
  }

  return (
    <ArticleEditor isNew={true} />
  );
}
