import ArticleEditor from '@/components/blog/Editor';
import { redirect } from '@tanstack/react-router';
import { useAuth } from '@/services/auth/auth';
import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import { useAdminDashboard } from '@/services/dashboard/dashboard';

export const Route = createFileRoute('/dashboard/blog/edit/$blogSlug')({
  component: EditArticlePage,
});

function EditArticlePage() {  
  const { user } = useAuth();
  const { setPageTitle } = useAdminDashboard();

  useEffect(() => {
    setPageTitle("Edit Article");
  }, [setPageTitle]);

  if (!user) {
    redirect({ to: '/login' });
  }


  return (
    <ArticleEditor />
  );
}
