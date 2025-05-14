'use server'

import ArticleEditor from '@/components/blog/Editor';
import { redirect } from '@tanstack/react-router';
import { useAuth } from '@/services/auth/auth';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/dashboard/blog/edit/$blogSlug')({
  component: EditArticlePage,
});

function EditArticlePage() {  
  const { user } = useAuth();

  if (!user) {
    redirect({ to: '/login' });
  }


  return (
    <ArticleEditor />
  );
}
