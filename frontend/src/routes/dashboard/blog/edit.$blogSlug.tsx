'use server'

import ArticleEditor from '@/components/blog/Editor';
import { redirect } from '@tanstack/react-router';
import { getUser } from '@/services/user';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/dashboard/blog/edit/$blogSlug')({
  component: EditArticlePage,
});

async function EditArticlePage() {  
  const user = await getUser();

  if (!user) {
    redirect({ to: '/login' });
  }


  return (
    <ArticleEditor />
  );
}
